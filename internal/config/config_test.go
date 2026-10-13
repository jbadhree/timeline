package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbadhree/timeline/core"
)

const valid = `version: 1
name: markets
types:
  news:
    folder: news
    entry_timing: past
    fields:
      ticker: { type: string, required: true }
      beliefs: { type: list, items: string }
  observation:
    folder: data/observations
    entry_timing: real_time
    max_delay: 15m
    status: { values: [open, closed], closed: [closed] }
    sections: [Source]
  goal:
    folder: goals
    fields:
      budget: { type: number }
    sections: [Risks]
  task:
    folder: tasks
`

func TestDecodeValid(t *testing.T) {
	cfg, err := Decode([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "markets" || len(cfg.Types) != 4 {
		t.Fatalf("cfg = %+v", cfg)
	}
	obs, _ := cfg.TypeByName("observation")
	if obs.MaxDelay != 15*time.Minute || obs.Status == nil || !obs.Status.IsClosed("closed") {
		t.Errorf("observation = %+v", obs)
	}
	goal, _ := cfg.TypeByName("goal")
	if goal.Builtin != core.GoalType || goal.Timing != core.Future {
		t.Errorf("goal = %+v", goal)
	}
	if _, ok := goal.Field("budget"); !ok {
		t.Error("goal lost its configured extra field")
	}
	if _, ok := goal.Field("goal"); !ok {
		t.Error("goal lost its built-in field")
	}
	if got, ok := cfg.TypeForPath("data/observations/x.md"); !ok || got.Name != "observation" {
		t.Errorf("TypeForPath = %v, %v", got, ok)
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		name, yaml, want string
	}{
		{"wrong version", "version: 2\ntypes: {}\n", "version must be 1"},
		{"missing version", "types: {}\n", "version is required"},
		{"unknown top-level key", "version: 1\ntypes: {}\nextra: 1\n", `unknown key "extra"`},
		{"missing folder", "version: 1\ntypes:\n  news: { entry_timing: past }\n", "folder is required"},
		{"missing entry_timing", "version: 1\ntypes:\n  news: { folder: news }\n", "entry_timing is required"},
		{"bad entry_timing", "version: 1\ntypes:\n  news: { folder: news, entry_timing: later }\n", "past, future, or real_time"},
		{"real_time without delay", "version: 1\ntypes:\n  obs: { folder: obs, entry_timing: real_time }\n", "max_delay is required"},
		{"delay on past", "version: 1\ntypes:\n  news: { folder: news, entry_timing: past, max_delay: 5m }\n", "only applies to real_time"},
		{"built-in timing", "version: 1\ntypes:\n  goal: { folder: goals, entry_timing: past }\n", `unknown key "entry_timing"`},
		{"built-in field", "version: 1\ntypes:\n  task:\n    folder: tasks\n    fields: { owner: { type: string } }\n", `cannot redefine the built-in field "owner"`},
		{"shared field", "version: 1\ntypes:\n  news:\n    folder: news\n    entry_timing: past\n    fields: { timeline_at: { type: time } }\n", `cannot redefine the shared field "timeline_at"`},
		{"same folder", "version: 1\ntypes:\n  a: { folder: x, entry_timing: past }\n  b: { folder: x, entry_timing: past }\n", "both use folder"},
		{"nested folders", "version: 1\ntypes:\n  a: { folder: x, entry_timing: past }\n  b: { folder: x/y, entry_timing: past }\n", "must not be nested"},
		{"folder outside root", "version: 1\ntypes:\n  a: { folder: ../x, entry_timing: past }\n", "subfolder of the root"},
		{"list without items", "version: 1\ntypes:\n  a:\n    folder: a\n    entry_timing: past\n    fields: { tags: { type: list } }\n", "a list needs items"},
		{"closed not in values", "version: 1\ntypes:\n  a:\n    folder: a\n    entry_timing: past\n    status: { values: [open], closed: [done] }\n", `closed status "done"`},
		{"invalid YAML", "version: [1\n", "yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.yaml))
			var cerr *Error
			if !errors.As(err, &cerr) {
				t.Fatalf("want *Error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q lacks %q", err, tt.want)
			}
		})
	}
}

func TestFindRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte("version: 1\ntypes: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	if got, err := FindRoot("", nested); err != nil || got != root {
		t.Errorf("nearest root = %q, %v; want %q", got, err, root)
	}
	if got, err := FindRoot("../..", nested); err != nil || got != root {
		t.Errorf("explicit relative root = %q, %v", got, err)
	}
	if _, err := FindRoot(nested, root); !errors.Is(err, ErrNoRoot) {
		t.Errorf("explicit root without timeline.yaml: %v", err)
	}
	if _, err := FindRoot("", t.TempDir()); !errors.Is(err, ErrNoRoot) {
		t.Errorf("no root anywhere: %v", err)
	}
}
