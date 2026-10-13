package goals_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/document"
	"github.com/jbadhree/timeline/internal/plugins/goals"
)

// viewFiles is a small plan: a top goal with a child goal and a task, and a
// child task with two parents. now is 2026-10-12; the child task is due
// 2026-10-10 and still open, so it is past due.
func viewFiles() map[string]string {
	g := func(title, due, status, how string, why bool) string {
		md := "---\ntype: goal\ntimeline_at: " + due + "T00:00:00Z\ncreated_by: t\ngoal: " + title + "\nowner: Ada\nstatus: " + status + "\n---\n"
		if why {
			md += "\n## Why\n\nIt matters.\n"
		}
		return md + "\n## How\n\n" + how + "\n\n## Done when\n\n- Done.\n\n## Why it will work\n\nIt will.\n"
	}
	t := func(title, due, status, owner string) string {
		return "---\ntype: task\ntimeline_at: " + due + "T00:00:00Z\ncreated_by: t\ntask: " + title + "\nowner: " + owner + "\nstatus: " + status + "\n---\n\n## How\n\n- Do it.\n"
	}
	return map[string]string{
		"goals/top.md":   g("Ship", "2026-12-31", "open", "- [Child goal](child.md)\n- [Write docs](../tasks/docs.md)", true),
		"goals/child.md": g("Build the core", "2026-11-30", "open", "- [Lint](../tasks/lint.md)\n- [Write docs](../tasks/docs.md)", false),
		"tasks/docs.md":  t("Write docs", "2026-10-10", "open", "Ada"),
		"tasks/lint.md":  t("Add lint", "2026-10-01", "done", "Bo"),
	}
}

func runView(t *testing.T, files map[string]string, name string, asJSON bool, args ...string) (core.CommandOutput, error) {
	t.Helper()
	p := goals.New()
	plugins := []core.Plugin{p}
	cfg, err := config.Decode([]byte(rootConfig), plugins)
	if err != nil {
		t.Fatal(err)
	}
	var docs []core.Document
	for path, md := range files {
		docs = append(docs, document.Parse(path, []byte(md)))
	}
	// Drafts are checked against now, so pick a now before every due date
	// for validity, then judge past due at viewNow through the facts.
	in := core.CheckInput{Config: cfg, Docs: docs, Now: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Plugins: plugins}
	var cmd core.Command
	for _, c := range p.Commands() {
		if c.Name == name {
			cmd = c
		}
	}
	viewCmd := cmd
	viewCmd.Run = func(ctx context.Context, ci core.CommandInput) (core.CommandOutput, error) {
		ci.Now = viewNow
		for i := range ci.Events {
			e := &ci.Events[i]
			e.PastDue = core.PastDue(e.Type, e.Doc, e.Time, viewNow)
		}
		return cmd.Run(ctx, ci)
	}
	return core.RunCommand(context.Background(), in, p, viewCmd, args, asJSON)
}

var viewNow = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)

func TestGoalTree(t *testing.T) {
	out, err := runView(t, viewFiles(), "goal", false, "tree")
	if err != nil {
		t.Fatal(err)
	}
	if !treeMatches(out.Text) {
		t.Errorf("tree =\n%s", out.Text)
	}
	if out.Incomplete {
		t.Error("valid plan reported incomplete")
	}

	var js struct {
		Goals []struct {
			Path     string
			Children []struct {
				Path     string
				PastDue  bool `json:"past_due"`
				Children []struct{ Path string }
			}
		}
	}
	out, _ = runView(t, viewFiles(), "goal", true, "tree")
	b, _ := json.Marshal(out.JSON)
	if err := json.Unmarshal(b, &js); err != nil || len(js.Goals) != 1 || len(js.Goals[0].Children) != 2 ||
		js.Goals[0].Children[0].Path != "goals/child.md" || len(js.Goals[0].Children[0].Children) != 2 ||
		!js.Goals[0].Children[1].PastDue {
		t.Errorf("tree JSON = %s", b)
	}
}

// treeMatches checks the tree line by line; each line may end with more
// flags, such as "draft".
func treeMatches(got string) bool {
	lines := strings.Split(strings.TrimSpace(got), "\n")
	want := []string{
		"[open] Ship  (2026-12-31)",
		"├─ [open] Build the core  (2026-11-30)",
		"│  ├─ [done] Add lint  (2026-10-01)",
		"│  └─ [open] Write docs  (2026-10-10)  PAST DUE",
		"└─ [open] Write docs  (2026-10-10)  PAST DUE",
	}
	if len(lines) != len(want) {
		return false
	}
	for i := range want {
		if !strings.HasPrefix(lines[i], want[i]) {
			return false
		}
	}
	return true
}

func TestGoalAndTaskLists(t *testing.T) {
	out, err := runView(t, viewFiles(), "task", false, "list")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.Text), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "2026-10-01  done") || !strings.Contains(lines[2], "PAST DUE") ||
		!strings.Contains(lines[2], "goals/child.md, goals/top.md") {
		t.Errorf("task list =\n%s", out.Text)
	}

	out, _ = runView(t, viewFiles(), "task", true, "list", "--past-due")
	b, _ := json.Marshal(out.JSON)
	var js struct{ Tasks []struct{ Path string } }
	if json.Unmarshal(b, &js) != nil || len(js.Tasks) != 1 || js.Tasks[0].Path != "tasks/docs.md" {
		t.Errorf("--past-due = %s", b)
	}

	out, _ = runView(t, viewFiles(), "task", true, "list", "--owner", "Bo", "--status", "done")
	b, _ = json.Marshal(out.JSON)
	if json.Unmarshal(b, &js) != nil || len(js.Tasks) != 1 || js.Tasks[0].Path != "tasks/lint.md" {
		t.Errorf("--owner --status = %s", b)
	}

	out, _ = runView(t, viewFiles(), "goal", false, "list", "--owner", "Nobody")
	if out.Text != "No goals match.\n" {
		t.Errorf("empty list = %q", out.Text)
	}

	out, _ = runView(t, viewFiles(), "goal", false, "list")
	lines = strings.Split(strings.TrimSpace(out.Text), "\n")
	if len(lines) != 3 || !strings.Contains(lines[1], "goals/child.md") || !strings.Contains(lines[2], "goals/top.md") {
		t.Errorf("goal list not ordered by due date:\n%s", out.Text)
	}
}

func TestShowViews(t *testing.T) {
	out, err := runView(t, viewFiles(), "goal", false, "show", "child")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Goal: Build the core",
		"Inherited from:\n  - Ship (goals/top.md)",
		"  - [done] task: Add lint  (2026-10-01)",
		"  - [open] task: Write docs  (2026-10-10)  PAST DUE",
		"Done when\n  - Done.",
		"Why it will work\n  It will.",
	} {
		if !strings.Contains(out.Text, want) {
			t.Errorf("goal show lacks %q:\n%s", want, out.Text)
		}
	}

	out, _ = runView(t, viewFiles(), "task", true, "show", "tasks/docs.md")
	b, _ := json.Marshal(out.JSON)
	var js struct {
		Task struct {
			PastDue bool `json:"past_due"`
			Parents []string
		}
		WhyInheritedFrom []string `json:"why_inherited_from"`
		How              string
	}
	if json.Unmarshal(b, &js) != nil || !js.Task.PastDue || len(js.WhyInheritedFrom) != 2 || js.How != "- Do it." {
		t.Errorf("task show JSON = %s", b)
	}
}

func TestViewArguments(t *testing.T) {
	files := viewFiles()
	files["goals/other/top.md"] = files["goals/top.md"]
	for _, args := range [][]string{
		{"goal"},
		{"goal", "show"},
		{"goal", "show", "missing"},
		{"goal", "show", "top"},  // ambiguous
		{"goal", "show", "docs"}, // a task, not a goal
		{"goal", "list", "--sideways"},
		{"goal", "tree", "a", "b"},
		{"task", "tree"},
	} {
		_, err := runView(t, files, args[0], false, args[1:]...)
		var uerr core.UsageError
		if !errors.As(err, &uerr) {
			t.Errorf("%v: want a usage error, got %v", args, err)
		}
	}
	_, err := runView(t, files, "goal", false, "show", "top")
	if err == nil || !strings.Contains(err.Error(), "goals/other/top.md, goals/top.md") {
		t.Errorf("ambiguous name error = %v", err)
	}
}

func TestViewsAreIncompleteWhenAGoalIsInvalid(t *testing.T) {
	files := viewFiles()
	files["goals/bad.md"] = "---\ntype: goal\n---\n"
	out, err := runView(t, files, "goal", false, "list")
	if err != nil || !out.Incomplete {
		t.Errorf("out %+v, err %v", out, err)
	}
}
