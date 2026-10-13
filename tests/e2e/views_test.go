package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The plan fixture: goals/top.md -> goals/child.md -> tasks/late.md, plus
// tasks/done.md under both goals/top.md and goals/sub/top.md (which shares a
// file name with goals/top.md). tasks/late.md is recorded with a 2026-02-01
// due date and is still open, so it is past due.

func TestVersionText(t *testing.T) {
	r := tline(t, t.TempDir(), "--version")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "tline ") || r.stderr != "" {
		t.Fatalf("got %+v", r)
	}
}

func TestGoalTreeOfOneGoal(t *testing.T) {
	r := tline(t, fixture(t, "plan"), "goal", "tree", "goals/top.md")
	want := []string{
		"[open] Top goal  (2099-12-31)",
		"├─ [open] Child goal  (2099-11-30)",
		"│  └─ [open] Late task  (2026-02-01)  PAST DUE",
		"└─ [done] Finished task  (2099-01-01)",
	}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	if r.code != 0 || len(lines) != len(want) {
		t.Fatalf("got %+v", r)
	}
	for i, w := range want {
		if !strings.HasPrefix(lines[i], w) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], w)
		}
	}
	if r = tline(t, fixture(t, "plan"), "goal", "tree"); r.code != 0 || strings.Count(r.stdout, "Finished task") != 2 {
		t.Errorf("full tree should show the shared task under both goals:\n%s", r.stdout)
	}
}

func TestListFilters(t *testing.T) {
	root := fixture(t, "plan")
	paths := func(args ...string) []string {
		t.Helper()
		r := tline(t, root, append(args, "--json")...)
		if r.code != 0 {
			t.Fatalf("%v: %+v", args, r)
		}
		var out map[string][]struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(r.stdout), &out); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, r.stdout)
		}
		var ps []string
		for _, list := range out {
			for _, e := range list {
				ps = append(ps, e.Path)
			}
		}
		return ps
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"goal", "list"}, "goals/sub/top.md goals/child.md goals/top.md"}, // by due date
		{[]string{"goal", "list", "--owner", "Bo"}, "goals/child.md"},
		{[]string{"task", "list", "--status", "done"}, "tasks/done.md"},
		{[]string{"task", "list", "--past-due"}, "tasks/late.md"},
		{[]string{"task", "list", "--owner", "Ada", "--status", "open"}, ""},
	} {
		if got := strings.Join(paths(tc.args...), " "); got != tc.want {
			t.Errorf("%v = %q, want %q", tc.args, got, tc.want)
		}
	}

	r := tline(t, root, "task", "list", "--past-due")
	if r.code != 0 || !strings.Contains(r.stdout, "PAST DUE") || !strings.Contains(r.stdout, "goals/child.md") {
		t.Errorf("past-due text: %+v", r)
	}
	if r = tline(t, root, "task", "list", "--owner", "Nobody"); r.stdout != "No tasks match.\n" {
		t.Errorf("empty list: %+v", r)
	}
}

func TestGoalShow(t *testing.T) {
	root := fixture(t, "plan")
	r := tline(t, root, "goal", "show", "child")
	for _, want := range []string{
		"Goal: Child goal",
		"Owner:  Bo",
		"Inherited from:\n  - Top goal (goals/top.md)",
		"  - [open] task: Late task  (2026-02-01)  PAST DUE",
		"Done when\n  - It is done.",
		"Why it will work\n  It is small.",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("goal show lacks %q:\n%s", want, r.stdout)
		}
	}

	r = tline(t, root, "goal", "show", "goals/top.md", "--json")
	var js struct {
		Goal struct {
			Title string `json:"title"`
		} `json:"goal"`
		Why string `json:"why"`
		How []struct {
			Targets []struct {
				Path string `json:"path"`
			} `json:"targets"`
		} `json:"how"`
	}
	if r.code != 0 || json.Unmarshal([]byte(r.stdout), &js) != nil || js.Goal.Title != "Top goal" || js.Why != "It matters." ||
		len(js.How) != 2 || js.How[0].Targets[0].Path != "goals/child.md" {
		t.Errorf("goal show --json: %+v", r)
	}

	r = tline(t, root, "task", "show", "tasks/late.md")
	if r.code != 0 || !strings.Contains(r.stdout, "id 01J00000000000000000000ABC") || !strings.Contains(r.stdout, "- Child goal (goals/child.md)") {
		t.Errorf("task show by path: %+v", r)
	}
}

func TestViewUsageErrors(t *testing.T) {
	root := fixture(t, "plan")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"goal", "show", "top"}, `"top" matches several goals: goals/sub/top.md, goals/top.md`},
		{[]string{"goal", "show", "done"}, `no goal matches "done"`}, // a task
		{[]string{"task", "tree"}, "expected list or show"},
		{[]string{"goal", "list", "--sideways"}, "flag provided but not defined"},
		{[]string{"goal"}, "expected list, show, or tree"},
	} {
		r := tline(t, root, tc.args...)
		if r.code != 2 || r.stdout != "" || !strings.Contains(r.stderr, tc.want) {
			t.Errorf("%v: %+v", tc.args, r)
		}
	}

	r := tline(t, root, "goal", "--help")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "Usage:\n  tline goal list") {
		t.Errorf("goal --help: %+v", r)
	}
}

func TestExplicitRootWithoutConfig(t *testing.T) {
	r := tline(t, t.TempDir(), "--root", filepath.Join("..", "..", "examples"), "check", "--dry-run")
	if r.code != 2 || !strings.Contains(r.stderr, "no timeline.yaml found in --root") {
		t.Fatalf("got %+v", r)
	}
}

func TestHeldLockIsAConflict(t *testing.T) {
	root := copyFixture(t, "valid")
	if err := os.MkdirAll(filepath.Join(root, ".tline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".tline", "lock"), []byte("pid 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := readTree(t, root)

	start := time.Now()
	r := tlineEnv(t, root, []string{"TLINE_LOCK_WAIT=300ms"}, "check")
	if r.code != 4 || !strings.Contains(r.stdout, "held by another tline check") || !strings.Contains(r.stdout, ".tline/lock") {
		t.Fatalf("got %+v", r)
	}
	if waited := time.Since(start); waited < 300*time.Millisecond {
		t.Errorf("did not wait for the lock: %v", waited)
	}
	if after := readTree(t, root); after["goals/ship.md"] != before["goals/ship.md"] {
		t.Error("stamped without the lock")
	}
	if r = tline(t, root, "check", "--dry-run"); r.code != 0 {
		t.Errorf("dry run blocked by the lock: %+v", r)
	}
	if r = tlineEnv(t, root, []string{"TLINE_LOCK_WAIT=soon"}, "check"); r.code != 2 || !strings.Contains(r.stderr, "TLINE_LOCK_WAIT") {
		t.Errorf("bad TLINE_LOCK_WAIT: %+v", r)
	}
}

// TestThisRepositorysPlanIsValid guards the repository's own goals, tasks,
// and decisions. They are all recorded, so the result does not depend on the
// date: recorded events are judged against their own entry times.
func TestThisRepositorysPlanIsValid(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	r := tline(t, t.TempDir(), "--root", root, "check", "--dry-run", "--json")
	var out struct {
		Valid  bool `json:"valid"`
		Events []struct {
			Draft bool `json:"draft"`
		} `json:"events"`
	}
	if r.code != 0 || json.Unmarshal([]byte(r.stdout), &out) != nil || !out.Valid || len(out.Events) == 0 {
		t.Fatalf("this repository's plan does not check:\n%s%s", r.stdout, r.stderr)
	}
	for _, e := range out.Events {
		if e.Draft {
			t.Errorf("the plan has an unstamped draft; run tline check to record it")
			break
		}
	}
	if r = tline(t, root, "goal", "tree"); r.code != 0 || !strings.HasPrefix(r.stdout, "[") {
		t.Errorf("goal tree on this repository: %+v", r)
	}
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTimelineWithoutTheGoalsPlugin(t *testing.T) {
	// With an empty plugins list, "task" is an ordinary configured type:
	// no goal or task rules apply, and the goal commands refuse to run.
	root := writeFiles(t, map[string]string{
		"timeline.yaml": "version: 1\nplugins: []\ntypes:\n  task: { folder: tasks, entry_timing: future }\n",
		"tasks/a.md":    "---\ntype: task\ntimeline_at: 2099-01-01T00:00:00Z\ncreated_by: e2e\n---\n\nNo owner, status, or How: fine for a plain configured type.\n",
	})
	if r := tline(t, root, "check", "--dry-run"); r.code != 0 {
		t.Fatalf("check: %+v", r)
	}
	r := tline(t, root, "goal", "list")
	if r.code != 2 || !strings.Contains(r.stderr, "the goals plugin is not enabled in timeline.yaml; add it to plugins") {
		t.Errorf("goal list: %+v", r)
	}
}

func TestExplicitPluginConfig(t *testing.T) {
	goal := func(typ, title, how string) string {
		return "---\ntype: " + typ + "\ntimeline_at: 2099-12-31T00:00:00Z\ncreated_by: e2e\ngoal: " + title + "\nowner: Ada\nstatus: open\n---\n\n## Why\n\nBecause.\n\n## How\n\n" + how + "\n\n## Done when\n\n- Done.\n\n## Why it will work\n\nIt will.\n"
	}
	root := writeFiles(t, map[string]string{
		"timeline.yaml": `version: 1
plugins: [goals]
types:
  goal: { from: goals/goal, folder: goals }
  objective: { from: goals/goal, folder: objectives }
  task: { from: goals/task, folder: tasks }
`,
		"goals/g.md":      goal("goal", "Company goal", "- [Team objective](../objectives/o.md)"),
		"objectives/o.md": goal("objective", "Team objective", "- [Do it](../tasks/t.md)"),
		"tasks/t.md":      "---\ntype: task\ntimeline_at: 2099-06-01T00:00:00Z\ncreated_by: e2e\ntask: Do it\nowner: Bo\nstatus: open\n---\n\n## How\n\n- Just do it.\n",
	})
	if r := tline(t, root, "check", "--dry-run"); r.code != 0 {
		t.Fatalf("check: %+v", r)
	}
	r := tline(t, root, "goal", "tree", "g")
	want := []string{"[open] Company goal", "└─ [open] Team objective", "   └─ [open] Do it"}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	if r.code != 0 || len(lines) != 3 {
		t.Fatalf("goal tree: %+v", r)
	}
	for i, w := range want {
		if !strings.HasPrefix(lines[i], w) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], w)
		}
	}
	if r = tline(t, root, "goal", "list"); r.code != 0 || !strings.Contains(r.stdout, "objectives/o.md") {
		t.Errorf("goal list should include the renamed objective type: %+v", r)
	}
}

// TestQuickstartGuide keeps docs/quickstart.md honest: the quickstart
// fixture is the guide's example timeline after "tline check", and the
// outputs below are quoted in the guide.
func TestQuickstartGuide(t *testing.T) {
	root := fixture(t, "quickstart")
	if r := tline(t, root, "check", "--dry-run"); r.code != 0 || r.stdout != "Dry run: checked 3 events (0 drafts, 3 recorded): all valid. Nothing was written.\n" {
		t.Fatalf("check: %+v", r)
	}
	r := tline(t, root, "goal", "tree")
	if want := "[open] Ship the first version  (2026-12-31)\n└─ [open] Write the user guide  (2026-12-15)\n"; r.code != 0 || r.stdout != want {
		t.Errorf("goal tree =\n%s\nwant\n%s", r.stdout, want)
	}
	r = tline(t, root, "task", "list")
	want := "DUE         STATUS  FLAGS  OWNER  TASK                  GOAL                    PATH\n" +
		"2026-12-15  open           Ada    Write the user guide  goals/first-release.md  tasks/user-guide.md\n"
	if r.code != 0 || r.stdout != want {
		t.Errorf("task list =\n%s\nwant\n%s", r.stdout, want)
	}

	// The guide's broken task, added to a copy.
	dir := copyFixture(t, "quickstart")
	oops := "---\ntype: task\ntimeline_at: 2099-12-15T23:59:59Z\ncreated_by: Ada\ntask: Something unplanned\nonwer: Ada\nstatus: someday\n---\n\n## How\n\n- Do it.\n"
	if err := os.WriteFile(filepath.Join(dir, "tasks", "oops.md"), []byte(oops), 0o644); err != nil {
		t.Fatal(err)
	}
	r = tline(t, dir, "check", "--dry-run")
	wantErrs := `tasks/oops.md: section "Why": is required because no goal links to this task from its How section
tasks/oops.md: field "owner": is required
tasks/oops.md:6: field "onwer": is not a field of type "task"
tasks/oops.md:7: field "status": is "someday"; allowed values are open, done, dropped
`
	if r.code != 3 || !strings.HasPrefix(r.stdout, wantErrs) {
		t.Errorf("broken task:\n%s", r.stdout)
	}
}
