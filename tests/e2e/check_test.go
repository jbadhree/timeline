package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture events use dates in 2099: a draft is checked against the current
// time, so nearer dates would start failing once they pass.

type checkJSON struct {
	DryRun   bool `json:"dry_run"`
	Recorded bool `json:"recorded"`
	Valid    bool `json:"valid"`
	Events   []struct {
		Path  string `json:"path"`
		Type  string `json:"type"`
		Draft bool   `json:"draft"`
		Valid bool   `json:"valid"`
	} `json:"events"`
	Diagnostics []struct {
		Path    string `json:"path"`
		Field   string `json:"field"`
		Section string `json:"section"`
		Message string `json:"message"`
	} `json:"diagnostics"`
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func decodeCheck(t *testing.T, r result) checkJSON {
	t.Helper()
	var out checkJSON
	if err := json.Unmarshal([]byte(r.stdout), &out); err != nil {
		t.Fatalf("stdout is not check JSON: %v\n%s", err, r.stdout)
	}
	return out
}

func TestCheckDryRunValidRoot(t *testing.T) {
	root := fixture(t, "valid")
	// Run from a nested folder to exercise upward root discovery.
	r := tline(t, filepath.Join(root, "tasks"), "check", "--dry-run")
	if r.code != 0 || r.stderr != "" {
		t.Fatalf("got %+v", r)
	}
	if !strings.Contains(r.stdout, "checked 3 events (2 drafts, 1 recorded): all valid. Nothing was recorded.") {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestCheckDryRunInvalidRootJSON(t *testing.T) {
	r := tline(t, t.TempDir(), "--root", fixture(t, "invalid"), "check", "--dry-run", "--json")
	if r.code != 3 {
		t.Fatalf("exit %d, want 3 (%+v)", r.code, r)
	}
	out := decodeCheck(t, r)
	if out.Valid || !out.DryRun || out.Recorded {
		t.Errorf("flags = %+v", out)
	}
	want := []string{
		`owner|is required`,
		`status|is "someday"`,
		`Done when|is required`,
		`Why it will work|is required`,
		`How|link "nowhere.md" does not resolve`,
		`Why|is required because no goal links`,
	}
	var got []string
	for _, d := range out.Diagnostics {
		if d.Path != "goals/incomplete.md" {
			t.Errorf("unexpected diagnostic path %q: %s", d.Path, d.Message)
		}
		got = append(got, d.Field+d.Section+"|"+d.Message)
	}
	all := strings.Join(got, "\n")
	for _, w := range want {
		parts := strings.SplitN(w, "|", 2)
		found := false
		for _, g := range got {
			if strings.HasPrefix(g, parts[0]+"|") && strings.Contains(g, parts[1]) {
				found = true
			}
		}
		if !found {
			t.Errorf("no diagnostic %q in:\n%s", w, all)
		}
	}
	for _, e := range out.Events {
		if wantValid := e.Path == "tasks/ok.md"; e.Valid != wantValid {
			t.Errorf("event %s valid = %v", e.Path, e.Valid)
		}
	}
}

func TestCheckSelectedPathWithSpaces(t *testing.T) {
	root := fixture(t, "valid")
	r := tline(t, root, "check", "--json", "tasks/list files.md", "--dry-run")
	if r.code != 0 {
		t.Fatalf("got %+v", r)
	}
	out := decodeCheck(t, r)
	if len(out.Events) != 1 || out.Events[0].Path != "tasks/list files.md" {
		t.Errorf("events = %+v", out.Events)
	}
}

func TestCheckMarketsExample(t *testing.T) {
	r := tline(t, t.TempDir(), "--root", filepath.Join("..", "..", "examples", "markets"), "check", "--dry-run", "--json")
	// --root is relative to the working directory, which is a temp dir here.
	if r.code != 2 {
		t.Fatalf("relative --root from elsewhere should not resolve: %+v", r)
	}
	abs, err := filepath.Abs(filepath.Join("..", "..", "examples", "markets"))
	if err != nil {
		t.Fatal(err)
	}
	r = tline(t, t.TempDir(), "--root", abs, "check", "--dry-run", "--json")
	if r.code != 0 {
		t.Fatalf("got %+v", r)
	}
	out := decodeCheck(t, r)
	if len(out.Events) != 2 || out.Events[0].Draft || out.Events[1].Draft {
		t.Errorf("out = %+v", out)
	}
}

func TestCheckUsageAndConfigErrors(t *testing.T) {
	valid := fixture(t, "valid")

	badConfig := t.TempDir()
	if err := os.WriteFile(filepath.Join(badConfig, "timeline.yaml"), []byte("version: 1\ntypes:\n  news: { folder: news }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		dir      string
		args     []string
		code     int
		inStderr string
		inStdout string
	}{
		{"stamping not implemented", valid, []string{"check"}, 2, "stamping drafts is not implemented yet", ""},
		{"no root", t.TempDir(), []string{"check", "--dry-run"}, 2, "no timeline.yaml found", ""},
		{"path outside folders", valid, []string{"check", "--dry-run", "timeline.yaml"}, 2, "not inside a configured type folder", ""},
		{"path outside root", valid, []string{"check", "--dry-run", ".."}, 2, "outside the Timeline root", ""},
		{"missing path", valid, []string{"check", "--dry-run", "tasks/nope.md"}, 2, "tasks/nope.md", ""},
		{"invalid config", badConfig, []string{"check", "--dry-run"}, 3, "", "entry_timing is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tline(t, tt.dir, tt.args...)
			if r.code != tt.code {
				t.Errorf("exit %d, want %d (%+v)", r.code, tt.code, r)
			}
			if !strings.Contains(r.stderr, tt.inStderr) || !strings.Contains(r.stdout, tt.inStdout) {
				t.Errorf("got %+v", r)
			}
		})
	}
}
