package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbadhree/timeline/core"
)

const testConfig = "version: 1\ntypes:\n  task: { folder: tasks }\n"

const draftTask = `---
type: task
timeline_at: 2099-01-01T00:00:00Z
created_by: test
task: Do it
owner: Ada
status: open
---

## Why

Needed.

## How

- Run it.
`

// fixedEffects makes stamping deterministic for the duration of a test.
func fixedEffects(t *testing.T) {
	t.Helper()
	oldNow, oldEntropy, oldBefore := now, entropy, beforeStamp
	now = func() time.Time { return time.Date(2026, 10, 13, 6, 0, 0, 0, time.UTC) }
	entropy = bytes.NewReader(bytes.Repeat([]byte{7}, 1000))
	t.Cleanup(func() { now, entropy, beforeStamp = oldNow, oldEntropy, oldBefore })
}

func writeRoot(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["timeline.yaml"] = testConfig
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

func runJSON(t *testing.T, args ...string) (int, checkOutput) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(context.Background(), args, &out, &errOut)
	var res checkOutput
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("exit %d, stdout not JSON: %v\nstdout: %s\nstderr: %s", code, err, out.String(), errOut.String())
	}
	return code, res
}

func TestCheckStampsDrafts(t *testing.T) {
	fixedEffects(t)
	root := writeRoot(t, map[string]string{"tasks/a.md": draftTask})

	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitOK || !res.Recorded || len(res.Stamped) != 1 {
		t.Fatalf("exit %d, result %+v", code, res)
	}
	st := res.Stamped[0]
	if st.EnteredAt != "2026-10-13T06:00:00Z" || st.EntryTiming != "future" || len(st.ID) != 26 {
		t.Errorf("stamp = %+v", st)
	}
	got, _ := os.ReadFile(filepath.Join(root, "tasks", "a.md"))
	want := "---\nid: " + st.ID + "\nentered_at: 2026-10-13T06:00:00Z\nentry_timing: future\n" + strings.TrimPrefix(draftTask, "---\n")
	if string(got) != want {
		t.Errorf("file =\n%s\nwant\n%s", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".tline", "lock")); !os.IsNotExist(err) {
		t.Errorf("lock not released: %v", err)
	}

	// A second check finds nothing to stamp and changes nothing.
	code, res = runJSON(t, "--root", root, "--json", "check")
	again, _ := os.ReadFile(filepath.Join(root, "tasks", "a.md"))
	if code != ExitOK || res.Recorded || len(res.Stamped) != 0 || string(again) != string(got) {
		t.Errorf("second check: exit %d, %+v", code, res)
	}
}

func TestCheckStampsNothingWhenAnyFileIsInvalid(t *testing.T) {
	fixedEffects(t)
	root := writeRoot(t, map[string]string{
		"tasks/a.md":   draftTask,
		"tasks/bad.md": strings.Replace(draftTask, "owner: Ada\n", "", 1),
	})
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitValidation || res.Recorded || len(res.Stamped) != 0 {
		t.Fatalf("exit %d, %+v", code, res)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "tasks", "a.md")); string(got) != draftTask {
		t.Error("a valid draft was stamped although another file is invalid")
	}
}

func TestCheckFutureDraftThatHasPassed(t *testing.T) {
	fixedEffects(t)
	past := strings.Replace(draftTask, "2099-01-01T00:00:00Z", "2026-10-13T06:00:00Z", 1) // equal to now
	root := writeRoot(t, map[string]string{"tasks/a.md": past})
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitValidation || len(res.Stamped) != 0 {
		t.Fatalf("exit %d, %+v", code, res)
	}
}

func TestCheckStopsWhenAFileChangesDuringTheCheck(t *testing.T) {
	fixedEffects(t)
	root := writeRoot(t, map[string]string{"tasks/a.md": draftTask, "tasks/b.md": draftTask})
	edited := draftTask + "\nAn edit made while the check was running.\n"
	beforeStamp = func(rel string) {
		if rel == "tasks/b.md" {
			if err := os.WriteFile(filepath.Join(root, "tasks", "b.md"), []byte(edited), 0o644); err != nil {
				t.Error(err)
			}
		}
	}
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitConflict || !strings.Contains(res.Conflict, "tasks/b.md changed") {
		t.Fatalf("exit %d, %+v", code, res)
	}
	if len(res.Stamped) != 1 || res.Stamped[0].Path != "tasks/a.md" {
		t.Errorf("stamped = %+v", res.Stamped)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "tasks", "b.md")); string(got) != edited {
		t.Error("the edited file was overwritten")
	}
}

func TestCheckWaitsForTheLockThenReportsConflict(t *testing.T) {
	fixedEffects(t)
	old := lockWait
	lockWait = 100 * time.Millisecond
	t.Cleanup(func() { lockWait = old })
	root := writeRoot(t, map[string]string{"tasks/a.md": draftTask})
	if err := os.MkdirAll(filepath.Join(root, ".tline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".tline", "lock"), []byte("pid 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitConflict || !strings.Contains(res.Conflict, "held by another tline check") {
		t.Fatalf("exit %d, %+v", code, res)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "tasks", "a.md")); string(got) != draftTask {
		t.Error("stamped without the lock")
	}
	// A dry run never takes the lock.
	if code, _ := runJSON(t, "--root", root, "--json", "check", "--dry-run"); code != ExitOK {
		t.Errorf("dry run blocked by the lock: exit %d", code)
	}
}

// taskOwnerPlugin is a test plugin that rejects tasks owned by "Nobody", or
// fails outright when broken is set.
type taskOwnerPlugin struct{ broken bool }

func (taskOwnerPlugin) Name() string          { return "owners" }
func (taskOwnerPlugin) Types() []core.TypeDef { return []core.TypeDef{{Name: "task"}} }
func (p taskOwnerPlugin) Validate(in core.PluginInput) ([]core.Diagnostic, error) {
	if p.broken {
		return nil, errors.New("plugin crashed")
	}
	var out []core.Diagnostic
	for _, e := range in.Events {
		if f, ok := e.Doc.Field("owner"); ok && f.Value.Text == "Nobody" {
			out = append(out, core.Diagnostic{Path: e.Path, Line: f.Line, Field: "owner", Message: "every task needs a real owner"})
		}
	}
	return out, nil
}

func usePlugins(t *testing.T, ps ...core.Plugin) {
	t.Helper()
	old := plugins
	plugins = func() []core.Plugin { return ps }
	t.Cleanup(func() { plugins = old })
}

func TestCheckRunsPlugins(t *testing.T) {
	fixedEffects(t)
	nobody := strings.Replace(draftTask, "owner: Ada", "owner: Nobody", 1)
	root := writeRoot(t, map[string]string{"tasks/a.md": draftTask, "tasks/b.md": nobody})

	usePlugins(t, taskOwnerPlugin{})
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitValidation || len(res.Diagnostics) != 1 || res.Diagnostics[0].Path != "tasks/b.md" || len(res.Stamped) != 0 {
		t.Fatalf("exit %d, %+v", code, res)
	}

	usePlugins(t, taskOwnerPlugin{broken: true})
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"--root", root, "check", "--dry-run"}, &out, &errOut); code != ExitFailure ||
		!strings.Contains(errOut.String(), "plugin owners: plugin crashed") {
		t.Fatalf("broken plugin: exit %d, stderr %q", code, errOut.String())
	}
}
