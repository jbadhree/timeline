package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jbadhree/timeline/core"
)

// decisionsPlugin provides a decision type and a "decision" command:
// "tline decision current" lists current decisions.
type decisionsPlugin struct{ seen *core.CommandInput }

func (decisionsPlugin) Name() string { return "decisions" }

func (decisionsPlugin) Types() []core.TypeDef {
	return []core.TypeDef{{Name: "decision", Timing: core.Past,
		Status: &core.StatusDef{Values: []string{"current", "superseded"}, Closed: []string{"superseded"}}}}
}

func (p decisionsPlugin) Commands() []core.Command {
	return []core.Command{{
		Name:    "decision",
		Summary: "list decisions",
		Usage:   "Usage:\n  tline decision current\n",
		Run: func(_ context.Context, in core.CommandInput) (core.CommandOutput, error) {
			if p.seen != nil {
				*p.seen = in
			}
			if len(in.Args) != 1 || in.Args[0] != "current" {
				return core.CommandOutput{}, core.UsageError{Msg: "expected 'current'"}
			}
			var current []string
			for _, e := range in.Events {
				if f, ok := e.Doc.Field("status"); ok && f.Value.Text == "current" {
					current = append(current, e.Path)
				}
			}
			sort.Strings(current)
			return core.CommandOutput{
				Text:       strings.Join(current, "\n"),
				JSON:       map[string]any{"current": current},
				Incomplete: len(in.Diagnostics) > 0,
			}, nil
		},
	}}
}

func decisionRoot(t *testing.T, extra map[string]string) string {
	t.Helper()
	d := func(status string) string {
		return "---\ntype: decision\ntimeline_at: 2026-10-01T00:00:00Z\ncreated_by: t\nstatus: " + status + "\n---\n"
	}
	files := map[string]string{
		"decisions/a.md": d("current"),
		"decisions/b.md": d("superseded"),
		"decisions/c.md": d("current"),
		"tasks/t.md":     draftTask,
	}
	for k, v := range extra {
		files[k] = v
	}
	root := writeRoot(t, files)
	if err := os.WriteFile(filepath.Join(root, "timeline.yaml"), []byte(testConfig+"  decision: { folder: decisions }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func run3(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := Run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestPluginCommandRuns(t *testing.T) {
	fixedEffects(t)
	var seen core.CommandInput
	usePlugins(t, decisionsPlugin{seen: &seen})
	root := decisionRoot(t, nil)

	code, stdout, stderr := run3("--root", root, "decision", "current")
	if code != ExitOK || stdout != "decisions/a.md\ndecisions/c.md\n" || stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	// The command saw only its own type's events, the clock, and no --json.
	if len(seen.Events) != 3 || seen.JSON || !seen.Now.Equal(clock()) {
		t.Errorf("input = %+v", seen)
	}

	// --json works after the command too, and is not passed as an argument.
	code, stdout, _ = run3("--root", root, "decision", "current", "--json")
	var got struct{ Current []string }
	if code != ExitOK || json.Unmarshal([]byte(stdout), &got) != nil || len(got.Current) != 2 || !seen.JSON || len(seen.Args) != 1 {
		t.Fatalf("json: exit %d, stdout %q, args %v", code, stdout, seen.Args)
	}
}

func TestPluginCommandHelpAndErrors(t *testing.T) {
	usePlugins(t, decisionsPlugin{})
	root := decisionRoot(t, nil)

	if code, stdout, _ := run3("--help"); code != ExitOK || !strings.Contains(stdout, "Commands from plugins:\n  decision        list decisions (decisions plugin)") {
		t.Errorf("tline --help: exit %d\n%s", code, stdout)
	}
	if code, stdout, _ := run3("decision", "--help"); code != ExitOK || stdout != "Usage:\n  tline decision current\n" {
		t.Errorf("decision --help: exit %d, %q", code, stdout)
	}
	if code, stdout, stderr := run3("--root", root, "decision", "everything"); code != ExitUsage || stdout != "" || !strings.Contains(stderr, "decision: expected 'current'") {
		t.Errorf("bad args: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if code, _, _ := run3("--root", root, "nope"); code != ExitUsage {
		t.Errorf("unknown command: exit %d", code)
	}
}

func TestPluginCommandIncomplete(t *testing.T) {
	usePlugins(t, decisionsPlugin{})
	bad := "---\ntype: decision\ntimeline_at: 2026-10-01T00:00:00Z\ncreated_by: t\nstatus: maybe\n---\n"
	root := decisionRoot(t, map[string]string{"decisions/bad.md": bad, "tasks/broken.md": "no frontmatter"})

	code, stdout, stderr := run3("--root", root, "decision", "current")
	// Only problems in the plugin's own events count: tasks/broken.md does not.
	if code != ExitIncomplete || stdout != "decisions/a.md\ndecisions/c.md\n" || !strings.Contains(stderr, "result is incomplete") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

type clashCommands struct {
	name string
	cmd  string
}

func (c clashCommands) Name() string { return c.name }
func (c clashCommands) Commands() []core.Command {
	return []core.Command{{Name: c.cmd, Run: func(context.Context, core.CommandInput) (core.CommandOutput, error) {
		return core.CommandOutput{}, nil
	}}}
}

func TestPluginCommandClashes(t *testing.T) {
	for _, tc := range []struct {
		plugins []core.Plugin
		want    string
	}{
		{[]core.Plugin{clashCommands{"x", "check"}}, `plugin x registers command "check", which is a core command`},
		{[]core.Plugin{clashCommands{"x", "view"}, clashCommands{"y", "view"}}, `command "view" is registered by both plugin x and plugin y`},
	} {
		usePlugins(t, tc.plugins...)
		code, _, stderr := run3("anything")
		if code != ExitFailure || !strings.Contains(stderr, tc.want) {
			t.Errorf("exit %d, stderr %q, want %q", code, stderr, tc.want)
		}
	}
}

type failingCommand struct{}

func (failingCommand) Name() string { return "failing" }
func (failingCommand) Commands() []core.Command {
	return []core.Command{{Name: "fail", Run: func(context.Context, core.CommandInput) (core.CommandOutput, error) {
		return core.CommandOutput{}, errors.New("disk on fire")
	}}}
}

func TestPluginCommandFailure(t *testing.T) {
	usePlugins(t, failingCommand{})
	root := writeRoot(t, map[string]string{})
	code, stdout, stderr := run3("--root", root, "fail")
	if code != ExitFailure || stdout != "" || stderr != fmt.Sprintf("tline: %s\n", "fail: disk on fire") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

// A command sees the problems tline check reports in history types, such as
// a changed frozen field, and so marks its result incomplete.
func TestPluginCommandSeesHistoryProblems(t *testing.T) {
	fixedEffects(t)
	usePlugins(t, decisionsPlugin{})
	root := decisionRoot(t, nil)
	cfg := testConfig + "  decision: { folder: decisions, history: true, frozen: [status] }\n"
	if err := os.WriteFile(filepath.Join(root, "timeline.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := run3("--root", root, "check", filepath.Join(root, "decisions")); code != ExitOK {
		t.Fatalf("recording: exit %d\n%s%s", code, stdout, stderr)
	}
	if code, _, _ := run3("--root", root, "decision", "current"); code != ExitOK {
		t.Fatalf("before the change: exit %d", code)
	}
	edit(t, filepath.Join(root, "decisions", "b.md"), "status: superseded", "status: current")
	if code, _, stderr := run3("--root", root, "decision", "current"); code != ExitIncomplete || !strings.Contains(stderr, "incomplete") {
		t.Errorf("after changing a frozen field: exit %d, stderr %q", code, stderr)
	}
}
