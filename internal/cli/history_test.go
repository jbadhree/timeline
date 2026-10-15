package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const historyConfig = `version: 1
types:
  prediction:
    folder: predictions
    entry_timing: future
    history: true
    frozen: [pred, "## Reasoning"]
    fields:
      pred: { type: string, required: true }
      confidence: { type: number }
`

const draftPrediction = `---
type: prediction
timeline_at: 2099-12-31T00:00:00Z
created_by: test
pred: It rises
confidence: 0.6
---

## Reasoning

Demand.
`

// at sets the injected clock for the steps that follow.
func at(t *testing.T, s string) {
	t.Helper()
	when, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	now = func() time.Time { return when }
}

func historyRoot(t *testing.T, cfg string, files map[string]string) string {
	t.Helper()
	root := writeRoot(t, files)
	if err := os.WriteFile(filepath.Join(root, "timeline.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func edit(t *testing.T, path, old, new string) {
	t.Helper()
	s := readFile(t, path)
	if !strings.Contains(s, old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(s, old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// historyFiles lists the files under history/, relative to the root.
func historyFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(filepath.Join(root, "history"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}

func eventID(t *testing.T, path string) string {
	t.Helper()
	for _, line := range strings.Split(readFile(t, path), "\n") {
		if id, ok := strings.CutPrefix(line, "id: "); ok {
			return id
		}
	}
	t.Fatalf("%s has no id", path)
	return ""
}

func TestHistoryVersions(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	file := filepath.Join(root, "predictions", "p.md")

	// A dry run writes nothing.
	if code, res := runJSON(t, "--root", root, "--json", "check", "--dry-run"); code != ExitOK || len(res.History) != 0 || historyFiles(t, root) != nil {
		t.Fatalf("dry run: exit %d, %+v, files %v", code, res, historyFiles(t, root))
	}

	// Stamping saves v1, an exact copy of the stamped file.
	at(t, "2026-10-14T01:00:00Z")
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitOK || len(res.History) != 1 || res.History[0].Version != 1 || res.History[0].Reason != "recorded" {
		t.Fatalf("stamp: exit %d, %+v", code, res)
	}
	id := eventID(t, file)
	dir := filepath.Join(root, "history", "predictions", id)
	if readFile(t, filepath.Join(dir, "v1.md")) != readFile(t, file) {
		t.Error("v1 is not an exact copy of the stamped file")
	}

	// No change, no copy.
	at(t, "2026-10-14T02:00:00Z")
	if code, res := runJSON(t, "--root", root, "--json", "check"); code != ExitOK || len(res.History) != 0 || res.Recorded {
		t.Fatalf("unchanged: exit %d, %+v", code, res)
	}

	// An allowed edit saves v2.
	edit(t, file, "confidence: 0.6", "confidence: 0.8")
	at(t, "2026-10-14T03:00:00Z")
	if code, res := runJSON(t, "--root", root, "--json", "check"); code != ExitOK || len(res.History) != 1 || res.History[0].Version != 2 || res.History[0].Reason != "changed" {
		t.Fatalf("edit: exit %d, %+v", code, res)
	}
	if readFile(t, filepath.Join(dir, "v2.md")) != readFile(t, file) {
		t.Error("v2 is not an exact copy of the edited file")
	}
	log := readFile(t, filepath.Join(dir, "versions.log"))
	want := "v1 2026-10-14T01:00:00Z recorded predictions/p.md\nv2 2026-10-14T03:00:00Z changed predictions/p.md\n"
	if log != want {
		t.Errorf("versions.log =\n%s\nwant\n%s", log, want)
	}
}

func TestFrozenFieldsSectionsAndStamps(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	file := filepath.Join(root, "predictions", "p.md")
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("stamp failed")
	}
	before := historyFiles(t, root)

	for _, tc := range []struct {
		name, old, new, want string
	}{
		{"frozen field", "pred: It rises", "pred: It falls", `field "pred": is frozen; the first recorded version had "It rises"`},
		{"frozen section", "Demand.", "Supply.", `section "Reasoning": is frozen; the first recorded version had "Demand."`},
		{"removed frozen section", "## Reasoning\n\nDemand.\n", "", `section "Reasoning": is frozen; the first recorded version had "Demand."`},
		{"stamp", "entered_at: 2026-10-13T06:00:00Z", "entered_at: 2026-10-01T06:00:00Z", `field "entered_at": changed since the event was recorded`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := readFile(t, file)
			t.Cleanup(func() { _ = os.WriteFile(file, []byte(original), 0o644) })
			edit(t, file, tc.old, tc.new)
			for _, args := range [][]string{{"check", "--dry-run"}, {"check"}} {
				code, res := runJSON(t, append([]string{"--root", root, "--json"}, args...)...)
				found := false
				for _, d := range res.Diagnostics {
					found = found || strings.Contains(d.String(), tc.want)
				}
				if code != ExitValidation || !found {
					t.Errorf("%v: exit %d, diagnostics %v", args, code, res.Diagnostics)
				}
			}
			if after := historyFiles(t, root); len(after) != len(before) {
				t.Errorf("a rejected change was saved: %v", after)
			}
		})
	}

	// Changes to other fields and sections are accepted.
	edit(t, file, "confidence: 0.6", "confidence: 0.9")
	edit(t, file, "Demand.\n", "Demand.\n\n## Notes\n\nRevised.\n")
	if code, res := runJSON(t, "--root", root, "--json", "check"); code != ExitOK || len(res.History) != 1 {
		t.Fatalf("allowed change: exit %d, %+v", code, res)
	}
}

func TestEnablingHistoryAndRepairingAMissingCopy(t *testing.T) {
	fixedEffects(t)
	noHistory := strings.Replace(historyConfig, "    history: true\n    frozen: [pred, \"## Reasoning\"]\n", "", 1)
	root := historyRoot(t, noHistory, map[string]string{"predictions/a.md": draftPrediction, "predictions/b.md": draftPrediction})
	if code, res := runJSON(t, "--root", root, "--json", "check"); code != ExitOK || len(res.History) != 0 {
		t.Fatalf("stamp without history: exit %d, %+v", code, res)
	}

	// Turning history on copies every recorded event as v1.
	if err := os.WriteFile(filepath.Join(root, "timeline.yaml"), []byte(historyConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	at(t, "2026-10-14T05:00:00Z")
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitOK || len(res.History) != 2 || res.History[0].Reason != "history-enabled" || res.History[1].Reason != "history-enabled" {
		t.Fatalf("enable: exit %d, %+v", code, res)
	}

	// A copy missing after a stopped check is saved by the next check.
	idB := eventID(t, filepath.Join(root, "predictions", "b.md"))
	if err := os.RemoveAll(filepath.Join(root, "history", "predictions", idB)); err != nil {
		t.Fatal(err)
	}
	code, res = runJSON(t, "--root", root, "--json", "check")
	if code != ExitOK || len(res.History) != 1 || res.History[0].Path != "predictions/b.md" || res.History[0].Reason != "late" {
		t.Fatalf("repair: exit %d, %+v", code, res)
	}
}

// An event is removed and a new one recorded at its path, but the check
// stops before saving the new event's v1. The next check must save it, not
// mistake the new event for the old one with its ID changed.
func TestNewEventAtARemovedPathRepairsItsMissingCopy(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	file := filepath.Join(root, "predictions", "p.md")
	at(t, "2026-10-13T05:00:00Z")
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("recording the first event failed")
	}
	if err := os.WriteFile(file, []byte(draftPrediction), 0o644); err != nil {
		t.Fatal(err)
	}
	at(t, "2026-10-14T05:00:00Z")
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("recording the second event failed")
	}
	if err := os.RemoveAll(filepath.Join(root, "history", "predictions", eventID(t, file))); err != nil {
		t.Fatal(err)
	}
	code, res := runJSON(t, "--root", root, "--json", "check")
	if code != ExitOK || len(res.History) != 1 || res.History[0].Reason != "late" {
		t.Fatalf("repair: exit %d, %+v", code, res)
	}
}

func TestHistoryFollowsMovesAndKeepsRemovedEvents(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("stamp failed")
	}
	id := eventID(t, filepath.Join(root, "predictions", "p.md"))
	if err := os.MkdirAll(filepath.Join(root, "predictions", "2099"), 0o755); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "predictions", "2099", "renamed.md")
	if err := os.Rename(filepath.Join(root, "predictions", "p.md"), moved); err != nil {
		t.Fatal(err)
	}
	// A move alone changes no content, so it saves no copy.
	if code, res := runJSON(t, "--root", root, "--json", "check"); code != ExitOK || len(res.History) != 0 {
		t.Fatalf("move: exit %d, %+v", code, res)
	}
	code, stdout, _ := run3("--root", root, "history", "renamed")
	if code != ExitOK || !strings.Contains(stdout, "predictions/2099/renamed.md (prediction, id "+id+")") ||
		!strings.Contains(stdout, "(saved from predictions/p.md)") {
		t.Errorf("history after move: %s", stdout)
	}

	if err := os.Remove(moved); err != nil {
		t.Fatal(err)
	}
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("check after removal failed")
	}
	code, stdout, _ = run3("--root", root, "history", id)
	if code != ExitOK || !strings.Contains(stdout, "removed; no event file has this id, and its copies remain") || !strings.Contains(stdout, "v1") {
		t.Errorf("history after removal: %s", stdout)
	}
}

func TestChangedIDIsRejected(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	if code, _ := runJSON(t, "--root", root, "--json", "check"); code != ExitOK {
		t.Fatal("stamp failed")
	}
	file := filepath.Join(root, "predictions", "p.md")
	id := eventID(t, file)
	edit(t, file, "id: "+id, "id: 01J00000000000000000000ABC")
	code, res := runJSON(t, "--root", root, "--json", "check", "--dry-run")
	if code != ExitValidation || len(res.Diagnostics) != 1 || !strings.Contains(res.Diagnostics[0].Message, "this file was recorded as "+id) {
		t.Fatalf("exit %d, %+v", code, res.Diagnostics)
	}
}

func TestHistoryCommand(t *testing.T) {
	fixedEffects(t)
	root := historyRoot(t, historyConfig, map[string]string{"predictions/p.md": draftPrediction})
	file := filepath.Join(root, "predictions", "p.md")
	at(t, "2026-10-14T01:00:00Z")
	runJSON(t, "--root", root, "--json", "check")
	edit(t, file, "confidence: 0.6", "confidence: 0.7")
	edit(t, file, "Demand.\n", "Demand.\n\n## Notes\n\nAdded.\n")
	at(t, "2026-10-14T02:00:00Z")
	runJSON(t, "--root", root, "--json", "check")
	id := eventID(t, file)

	code, stdout, _ := run3("--root", root, "history", "p")
	want := "predictions/p.md (prediction, id " + id + ")\n" +
		"  v1  2026-10-14T01:00:00Z  recorded\n" +
		"  v2  2026-10-14T02:00:00Z  changed: confidence, ## Notes\n"
	if code != ExitOK || stdout != want {
		t.Errorf("history =\n%s\nwant\n%s", stdout, want)
	}

	edit(t, file, "confidence: 0.7", "confidence: 0.5")
	if _, stdout, _ := run3("--root", root, "history", "predictions/p.md"); !strings.HasSuffix(stdout, "The file differs from v2; run tline check to save it.\n") {
		t.Errorf("pending change not reported:\n%s", stdout)
	}

	code, stdout, _ = run3("--root", root, "--json", "history", id)
	var out historyOutput
	if code != ExitOK || json.Unmarshal([]byte(stdout), &out) != nil || len(out.Versions) != 2 || !out.Pending ||
		out.Versions[1].SavedAt != "2026-10-14T02:00:00Z" || strings.Join(out.Versions[1].ChangedFields, ",") != "confidence" ||
		strings.Join(out.Versions[1].ChangedSections, ",") != "Notes" {
		t.Errorf("history --json = %s", stdout)
	}

	code, stdout, _ = run3("--root", root, "history", "p", "--version", "1")
	if v1 := readFile(t, filepath.Join(root, "history", "predictions", id, "v1.md")); code != ExitOK || stdout != v1 {
		t.Errorf("--version 1 = %q", stdout)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"history"}, "name one event"},
		{[]string{"history", "nope"}, `no event with history matches "nope"`},
		{[]string{"history", "p", "--version", "9"}, "has no version 9"},
	} {
		code, _, stderr := run3(append([]string{"--root", root}, tc.args...)...)
		if code != ExitUsage || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, %q", tc.args, code, stderr)
		}
	}
}

func TestHistoryCommandOnATypeWithoutHistory(t *testing.T) {
	fixedEffects(t)
	root := writeRoot(t, map[string]string{"tasks/a.md": draftTask})
	runJSON(t, "--root", root, "--json", "check")
	code, _, stderr := run3("--root", root, "history", "a")
	if code != ExitUsage || !strings.Contains(stderr, "type task does not keep history; add history: true") {
		t.Errorf("exit %d, %q", code, stderr)
	}
}
