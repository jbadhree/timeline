package goals_test

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/document"
	"github.com/jbadhree/timeline/internal/plugins/goals"
)

// These tests run the whole stack: timeline.yaml through config with the
// plugin's templates, Markdown through the document parser, then the core
// rules and the plugin.

const rootConfig = `version: 1
types:
  goal: { folder: goals }
  task: { folder: tasks }
  news: { folder: news, entry_timing: past }
`

var now = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

func goalMD(how string, extra ...string) string {
	sections := map[string]string{
		"Why":              "Because.",
		"How":              how,
		"Done when":        "- It shipped.",
		"Why it will work": "It is simple.",
	}
	for i := 0; i+1 < len(extra); i += 2 {
		if extra[i+1] == "<omit>" {
			delete(sections, extra[i])
		} else {
			sections[extra[i]] = extra[i+1]
		}
	}
	var b strings.Builder
	b.WriteString("---\ntype: goal\ntimeline_at: 2026-12-31T23:59:59Z\ncreated_by: t\ngoal: Ship it\nowner: Ada\nstatus: open\n---\n")
	for _, h := range []string{"Why", "How", "Done when", "Why it will work"} {
		if body, ok := sections[h]; ok {
			b.WriteString("\n## " + h + "\n\n" + body + "\n")
		}
	}
	return b.String()
}

func taskMD(sections string) string {
	return "---\ntype: task\ntimeline_at: 2026-12-01T00:00:00Z\ncreated_by: t\ntask: List files\nowner: Ada\nstatus: open\n---\n" + sections
}

const stepTask = "\n## How\n\n- Do the work.\n"

func check(t *testing.T, files map[string]string, selected ...string) core.CheckResult {
	t.Helper()
	plugins := []core.Plugin{goals.New()}
	cfg, err := config.Decode([]byte(rootConfig), plugins)
	if err != nil {
		t.Fatal(err)
	}
	var docs []core.Document
	for path, md := range files {
		docs = append(docs, document.Parse(path, []byte(md)))
	}
	in := core.CheckInput{Config: cfg, Docs: docs, Now: now, Plugins: plugins}
	if len(selected) > 0 {
		in.Selected = selected
	}
	res, err := core.Check(in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// want asserts that diagnostics are exactly the given "path: substring" set.
func want(t *testing.T, res core.CheckResult, expected ...string) {
	t.Helper()
	var got []string
	for _, d := range res.Diagnostics {
		got = append(got, d.String())
	}
	all := strings.Join(got, "\n")
	if len(got) != len(expected) {
		t.Errorf("got %d diagnostics, want %d:\n%s", len(got), len(expected), all)
	}
	for _, e := range expected {
		path, sub, _ := strings.Cut(e, ": ")
		found := false
		for _, g := range got {
			if strings.HasPrefix(g, path) && strings.Contains(g, sub) {
				found = true
			}
		}
		if !found {
			t.Errorf("no diagnostic for %s containing %q in:\n%s", path, sub, all)
		}
	}
}

func TestTemplates(t *testing.T) {
	types := goals.New().Types()
	if len(types) != 2 || types[0].Name != "goal" || types[1].Name != "task" {
		t.Fatalf("types = %+v", types)
	}
	for _, typ := range types {
		if typ.Timing != core.Future || typ.Status == nil {
			t.Errorf("%s: timing %q, status %v", typ.Name, typ.Timing, typ.Status)
		}
	}
	cfg, err := config.Decode([]byte(rootConfig), []core.Plugin{goals.New()})
	if err != nil {
		t.Fatal(err)
	}
	if g, _ := cfg.TypeByName("goal"); g.Plugin != "goals" || g.Folder != "goals" {
		t.Errorf("configured goal = %+v", g)
	}
}

func TestCompleteGoalAndTaskAreValid(t *testing.T) {
	res := check(t, map[string]string{
		"goals/g.md": goalMD("- [List files](../tasks/t.md)"),
		"tasks/t.md": taskMD("\n## How\n\n- Run `ls`.\n"), // why inherited from goals/g.md
	})
	want(t, res)
}

func TestGoalMissingAnswers(t *testing.T) {
	md := strings.Replace(goalMD("", "Done when", "<omit>", "Why", "<omit>", "Why it will work", "  "), "goal: Ship it\nowner: Ada\nstatus: open\n", "", 1)
	want(t, check(t, map[string]string{"goals/g.md": md}),
		`goals/g.md: field "goal": is required`,
		`goals/g.md: field "owner": is required`,
		`goals/g.md: field "status": is required`,
		`goals/g.md: section "How": must list at least one step`,
		`goals/g.md: section "Done when": is required`,
		`goals/g.md: section "Why it will work": must not be empty`,
		`goals/g.md: section "Why": is required because no goal links to this goal`,
	)
}

func TestTaskRules(t *testing.T) {
	want(t, check(t, map[string]string{
		"goals/g.md":    goalMD("- [Step](../tasks/step.md)"),
		"tasks/step.md": taskMD(stepTask),
		"tasks/t.md":    taskMD("\n## Why\n\nNeeded.\n\n## How\n\n- See [goal](../goals/g.md)\n"),
		"tasks/u.md":    taskMD("\n## How\n\n- Run it.\n"),
	}),
		"tasks/t.md: a task's steps must not link to a goal (goals/g.md)",
		"tasks/u.md: is required because no goal links to this task",
	)
}

func TestGoalStepsMustLinkToGoalOrTask(t *testing.T) {
	want(t, check(t, map[string]string{
		"news/n.md":     "---\ntype: news\ntimeline_at: 2026-10-01T13:00:00Z\ncreated_by: t\n---\n",
		"tasks/step.md": taskMD(stepTask),
		"goals/g.md": goalMD("- Start with [the step](../tasks/step.md) first\n" +
			"- Plain text with no link\n" +
			"- [Background reading](https://example.com)\n" +
			"- [A news event](../news/n.md)\n" +
			"- [Missing](../tasks/missing.md)\n" +
			"- Wrapped step that continues\n  onto [the step](../tasks/step.md) here\n"),
	}),
		`goals/g.md: step "Plain text with no link" must link to a goal or task`,
		`goals/g.md: step "[Background reading](https://example.com)" must link to a goal or task`,
		`goals/g.md: step "[A news event](../news/n.md)" must link to a goal or task`,
		// The missing link is reported once by the core, not also as unlinked.
		`goals/g.md: link "../tasks/missing.md" does not resolve`,
	)
}

func TestGoalCycle(t *testing.T) {
	res := check(t, map[string]string{
		"goals/a.md": goalMD("- [B](b.md)"),
		"goals/b.md": goalMD("- [A](a.md#how)"),
	})
	want(t, res,
		"goals/a.md: goal steps form a cycle: goals/a.md -> goals/b.md -> goals/a.md",
		"goals/b.md: goal steps form a cycle: goals/a.md -> goals/b.md -> goals/a.md",
	)
}

func TestWhyInheritedFromAnUnselectedGoal(t *testing.T) {
	res := check(t, map[string]string{
		"goals/g.md":      goalMD("- [Task](../tasks/t.md)"),
		"tasks/t.md":      taskMD("\n## How\n\n- Run it.\n"),
		"tasks/broken.md": taskMD(""),
	}, "tasks/t.md")
	want(t, res)
	if len(res.Events) != 1 || res.Events[0].Path != "tasks/t.md" {
		t.Errorf("events = %+v", res.Events)
	}
}

func TestEmptyWhyIsRejectedEvenWithAParent(t *testing.T) {
	want(t, check(t, map[string]string{
		"goals/g.md": goalMD("- [Task](../tasks/t.md)"),
		"tasks/t.md": taskMD("\n## Why\n\n\n## How\n\n- Run it.\n"),
	}), `tasks/t.md: section "Why": must not be empty`)
}

func TestExtraConfiguredFieldsAndSections(t *testing.T) {
	plugins := []core.Plugin{goals.New()}
	cfg, err := config.Decode([]byte("version: 1\ntypes:\n  goal:\n    folder: goals\n    fields: { priority: { type: string, values: [low, high] } }\n    sections: [Risks]\n"), plugins)
	if err != nil {
		t.Fatal(err)
	}
	md := strings.Replace(goalMD("- [Sub](sub.md)"), "status: open\n", "status: open\npriority: urgent\n", 1)
	res, err := core.Check(core.CheckInput{Config: cfg, Now: now, Plugins: plugins, Docs: []core.Document{
		document.Parse("goals/g.md", []byte(md)),
		document.Parse("goals/sub.md", []byte(goalMD("- [G](g.md)", "Why", "<omit>"))),
	}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range res.Diagnostics {
		got = append(got, d.String())
	}
	all := strings.Join(got, "\n")
	for _, w := range []string{`field "priority": is "urgent"; allowed values are low, high`, `section "Risks": is required`, "cycle"} {
		if !strings.Contains(all, w) {
			t.Errorf("missing %q in:\n%s", w, all)
		}
	}
}

func TestRenamedGoalTypesKeepTheirRules(t *testing.T) {
	plugins := []core.Plugin{goals.New()}
	cfg, err := config.Decode([]byte(`version: 1
types:
  goal: { folder: goals }
  objective: { from: goals/goal, folder: objectives }
  chore: { from: goals/task, folder: chores }
`), plugins)
	if err != nil {
		t.Fatal(err)
	}
	objective := strings.Replace(goalMD("- [Sweep](../chores/sweep.md)\n- Plain step"), "type: goal\n", "type: objective\n", 1)
	objective = strings.Replace(objective, "goal: Ship it\n", "goal: Keep the floor clean\n", 1)
	chore := strings.Replace(taskMD("\n## How\n\n- Sweep.\n"), "type: task\n", "type: chore\n", 1)
	chore = strings.Replace(chore, "task: List files\n", "task: Sweep the floor\n", 1)
	res, err := core.Check(core.CheckInput{Config: cfg, Now: now, Plugins: plugins, Docs: []core.Document{
		document.Parse("goals/g.md", []byte(goalMD("- [Objective](../objectives/o.md)"))),
		document.Parse("objectives/o.md", []byte(objective)),
		document.Parse("chores/sweep.md", []byte(chore)),
	}})
	if err != nil {
		t.Fatal(err)
	}
	// The objective is a goal: its plain step must link, and it is a child
	// of goals/g.md; the chore is a task with an inherited why.
	want(t, res, `objectives/o.md: step "Plain step" must link to a goal or task`)

	out, err := core.RunCommand(context.Background(), core.CheckInput{Config: cfg, Now: now, Plugins: plugins, Docs: []core.Document{
		document.Parse("goals/g.md", []byte(goalMD("- [Objective](../objectives/o.md)"))),
		document.Parse("objectives/o.md", []byte(strings.Replace(objective, "\n- Plain step", "", 1))),
		document.Parse("chores/sweep.md", []byte(chore)),
	}}, goals.New(), goals.New().Commands()[0], []string{"tree"}, false)
	if err != nil || !strings.Contains(out.Text, "└─ [open] Keep the floor clean") || !strings.Contains(out.Text, "└─ [open] Sweep the floor") {
		t.Errorf("tree over renamed types:\n%s (%v)", out.Text, err)
	}
}

func TestLongStepTextIsShortenedWithoutSplittingCharacters(t *testing.T) {
	step := strings.Repeat("é", 31) // 62 bytes; byte 57 is inside a character
	res := check(t, map[string]string{"goals/g.md": goalMD("- " + step + "\n  more text\n")})
	if len(res.Diagnostics) != 1 {
		t.Fatalf("got %v", res.Diagnostics)
	}
	msg := res.Diagnostics[0].Message
	if !utf8.ValidString(msg) || strings.Contains(msg, `\x`) || !strings.Contains(msg, `..." must link`) || strings.Contains(msg, "more text") {
		t.Errorf("message = %q", msg)
	}
}
