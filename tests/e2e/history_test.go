package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const historyTimeline = `version: 1
types:
  prediction:
    folder: predictions
    entry_timing: future
    history: true
    frozen: [pred]
    fields:
      pred: { type: string, required: true }
      confidence: { type: number }
`

const historyDraft = "---\ntype: prediction\ntimeline_at: 2099-12-31T00:00:00Z\ncreated_by: e2e\npred: It rises\nconfidence: 0.6\n---\n"

func TestHistoryEndToEnd(t *testing.T) {
	root := writeFiles(t, map[string]string{"timeline.yaml": historyTimeline, "predictions/p.md": historyDraft})
	file := filepath.Join(root, "predictions", "p.md")
	replace := func(old, new string) {
		t.Helper()
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(strings.Replace(string(b), old, new, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if r := tline(t, root, "check", "--dry-run"); r.code != 0 {
		t.Fatalf("dry run: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(root, "history")); !os.IsNotExist(err) {
		t.Fatal("a dry run wrote history")
	}

	r := tline(t, root, "check")
	if r.code != 0 || !strings.Contains(r.stdout, "Saved 1 history copy:\n  predictions/p.md  v1  recorded") {
		t.Fatalf("stamp: %+v", r)
	}
	if r = tline(t, root, "check"); r.code != 0 || strings.Contains(r.stdout, "history copy") {
		t.Fatalf("unchanged: %+v", r)
	}

	replace("confidence: 0.6", "confidence: 0.9")
	if r = tline(t, root, "check"); r.code != 0 || !strings.Contains(r.stdout, "predictions/p.md  v2  changed") {
		t.Fatalf("edit: %+v", r)
	}

	replace("pred: It rises", "pred: It falls")
	r = tline(t, root, "check")
	if r.code != 3 || !strings.Contains(r.stdout, `field "pred": is frozen; the first recorded version had "It rises"`) {
		t.Fatalf("frozen: %+v", r)
	}
	replace("pred: It falls", "pred: It rises")

	r = tline(t, root, "history", "p")
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	if r.code != 0 || len(lines) != 3 || !strings.HasPrefix(lines[0], "predictions/p.md (prediction, id ") ||
		!strings.HasSuffix(lines[1], "recorded") || !strings.HasSuffix(lines[2], "changed: confidence") {
		t.Fatalf("history: %+v", r)
	}

	r = tline(t, root, "history", "predictions/p.md", "--json")
	var out struct {
		ID       string `json:"id"`
		Versions []struct {
			Version int    `json:"version"`
			Reason  string `json:"reason"`
		} `json:"versions"`
	}
	if r.code != 0 || json.Unmarshal([]byte(r.stdout), &out) != nil || len(out.Versions) != 2 || out.Versions[1].Reason != "changed" {
		t.Fatalf("history --json: %+v", r)
	}

	r = tline(t, root, "history", out.ID, "--version", "1")
	v1, _ := os.ReadFile(filepath.Join(root, "history", "predictions", out.ID, "v1.md"))
	if r.code != 0 || r.stdout != string(v1) || !strings.Contains(r.stdout, "confidence: 0.6") {
		t.Fatalf("history --version 1: %+v", r)
	}
}

func TestMarketsExampleHasFrozenPredictions(t *testing.T) {
	root := copyFixture(t, filepath.Join("..", "..", "..", "examples", "markets"))
	r := tline(t, root, "history", "2026-10-01-nvda-all-time-high")
	if r.code != 0 || !strings.Contains(r.stdout, "v1") || !strings.Contains(r.stdout, "copied when history was enabled") {
		t.Fatalf("history: %+v", r)
	}
	file := filepath.Join(root, "predictions", "2026-10-01-nvda-all-time-high.md")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(strings.Replace(string(b), "ticker: NVDA", "ticker: AMD", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if r = tline(t, root, "check", "--dry-run"); r.code != 3 || !strings.Contains(r.stdout, `field "ticker": is frozen`) {
		t.Fatalf("frozen ticker: %+v", r)
	}
}

func TestNewEventAtARemovedPathAfterAStoppedCheck(t *testing.T) {
	root := writeFiles(t, map[string]string{"timeline.yaml": historyTimeline, "predictions/p.md": historyDraft})
	file := filepath.Join(root, "predictions", "p.md")
	if r := tline(t, root, "check"); r.code != 0 {
		t.Fatalf("recording: %+v", r)
	}
	// Make the first event's copy old, as if it was recorded long ago.
	dirs, err := filepath.Glob(filepath.Join(root, "history", "predictions", "*"))
	if err != nil || len(dirs) != 1 {
		t.Fatalf("history dirs = %v, %v", dirs, err)
	}
	logPath := filepath.Join(dirs[0], "versions.log")
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(log))
	if err := os.WriteFile(logPath, []byte(strings.Replace(string(log), fields[1], "2000-01-01T00:00:00Z", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	v1Path := filepath.Join(dirs[0], "v1.md")
	v1, err := os.ReadFile(v1Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(v1Path, []byte(strings.Replace(string(v1), "entered_at: "+fields[1], "entered_at: 2000-01-01T00:00:00Z", 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replace it with a new event, then lose the new event's v1, as a check
	// stopped between stamping and saving would.
	if err := os.WriteFile(file, []byte(historyDraft), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := tline(t, root, "check"); r.code != 0 {
		t.Fatalf("recording the new event: %+v", r)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimPrefix(strings.Split(string(data), "\n")[1], "id: ")
	if err := os.RemoveAll(filepath.Join(root, "history", "predictions", id)); err != nil {
		t.Fatal(err)
	}
	if r := tline(t, root, "check"); r.code != 0 || !strings.Contains(r.stdout, "v1  late") {
		t.Errorf("the next check should save the missing copy: %+v", r)
	}
}
