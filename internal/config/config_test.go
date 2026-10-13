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

// templates is a stand-in for a plugin that provides goal and task types.
type templates struct{}

func (templates) Name() string { return "fake" }
func (templates) Types() []core.TypeDef {
	field := func(name string) core.FieldDef {
		return core.FieldDef{Name: name, Kind: core.KindString, Required: true}
	}
	status := &core.StatusDef{Values: []string{"open", "done"}, Closed: []string{"done"}}
	return []core.TypeDef{
		{Name: "goal", Timing: core.Future, Fields: []core.FieldDef{field("goal"), field("owner")}, Status: status},
		{Name: "task", Timing: core.Future, Fields: []core.FieldDef{field("task"), field("owner")}, Status: status},
	}
}

var plugins = []core.Plugin{templates{}}

func TestDecodeValid(t *testing.T) {
	cfg, err := Decode([]byte(valid), plugins)
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
	if goal.Plugin != "fake" || goal.Timing != core.Future || goal.Folder != "goals" {
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
		{"plugin type timing", "version: 1\ntypes:\n  goal: { folder: goals, entry_timing: past }\n", `unknown key "entry_timing"`},
		{"built-in field", "version: 1\ntypes:\n  task:\n    folder: tasks\n    fields: { owner: { type: string } }\n", `cannot redefine the field "owner" from plugin fake`},
		{"shared field", "version: 1\ntypes:\n  news:\n    folder: news\n    entry_timing: past\n    fields: { timeline_at: { type: time } }\n", `cannot redefine the shared field "timeline_at"`},
		{"same folder", "version: 1\ntypes:\n  a: { folder: x, entry_timing: past }\n  b: { folder: x, entry_timing: past }\n", "both use folder"},
		{"nested folders", "version: 1\ntypes:\n  a: { folder: x, entry_timing: past }\n  b: { folder: x/y, entry_timing: past }\n", "must not be nested"},
		{"folder outside root", "version: 1\ntypes:\n  a: { folder: ../x, entry_timing: past }\n", "subfolder of the root"},
		{"list without items", "version: 1\ntypes:\n  a:\n    folder: a\n    entry_timing: past\n    fields: { tags: { type: list } }\n", "a list needs items"},
		{"closed not in values", "version: 1\ntypes:\n  a:\n    folder: a\n    entry_timing: past\n    status: { values: [open], closed: [done] }\n", `closed status "done"`},
		{"invalid YAML", "version: [1\n", "yaml"},
		{"plugin type status", "version: 1\ntypes:\n  task: { folder: tasks, status: { values: [x] } }\n", `unknown key "status"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode([]byte(tt.yaml), plugins)
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

func TestPluginTypeNamesAreOrdinaryWithoutThePlugin(t *testing.T) {
	// Without a plugin providing it, "goal" is just a configured type and
	// needs its own entry_timing.
	_, err := Decode([]byte("version: 1\ntypes:\n  goal: { folder: goals }\n"), nil)
	if err == nil || !strings.Contains(err.Error(), "entry_timing is required") {
		t.Fatalf("got %v", err)
	}
}

type badTemplates struct{ name string }

func (b badTemplates) Name() string { return b.name }
func (badTemplates) Types() []core.TypeDef {
	return []core.TypeDef{{Name: "goal", Timing: core.Future}, {Name: "broken"}}
}

func TestPluginTemplateProblems(t *testing.T) {
	_, err := Decode([]byte("version: 1\ntypes:\n  goal: { folder: goals }\n"), []core.Plugin{templates{}, badTemplates{name: "other"}})
	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{
		`type "goal" is provided by both plugin fake and plugin other; use from: to choose one`,
		`plugin other provides type "broken" without a valid name and timing rule`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

// validating is a stand-in plugin with rules but no types.
type validating struct{}

func (validating) Name() string { return "lint" }
func (validating) Validate(core.PluginInput) ([]core.Diagnostic, error) {
	return nil, nil
}

func TestPluginsList(t *testing.T) {
	all := []core.Plugin{templates{}, validating{}}

	cfg, err := Decode([]byte("version: 1\ntypes:\n  goal: { folder: goals }\n"), all)
	if err != nil || strings.Join(cfg.Plugins, ",") != "fake,lint" {
		t.Fatalf("without a list every plugin is enabled: %v %v", cfg, err)
	}

	cfg, err = Decode([]byte("version: 1\nplugins: [lint]\ntypes:\n  goal: { folder: goals, entry_timing: past }\n"), all)
	if err != nil || strings.Join(cfg.Plugins, ",") != "lint" {
		t.Fatalf("with a list only those plugins are enabled: %v %v", cfg, err)
	}
	if g, _ := cfg.TypeByName("goal"); g.Plugin != "" || g.Timing != core.Past {
		t.Errorf("a disabled plugin's type name is free for a configured type: %+v", g)
	}

	for _, tc := range []struct{ yaml, want string }{
		{"plugins: [nope]", `unknown plugin "nope" (available: fake, lint)`},
		{"plugins: [fake, fake]", `plugin "fake" is listed twice`},
		{"plugins: [{ ticker: { run: ./check } }]", "external plugins are not supported yet"},
		{"plugins: fake", "plugins must be a list"},
	} {
		_, err := Decode([]byte("version: 1\n"+tc.yaml+"\ntypes: {}\n"), all)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", tc.yaml, err, tc.want)
		}
	}
}

func TestFromAndRenamedPluginTypes(t *testing.T) {
	cfg, err := Decode([]byte(`version: 1
types:
  goal: { folder: goals }
  objective: { from: fake/goal, folder: objectives, sections: [Risks] }
`), plugins)
	if err != nil {
		t.Fatal(err)
	}
	obj, _ := cfg.TypeByName("objective")
	if obj.Plugin != "fake" || obj.Base != "goal" || obj.Timing != core.Future || len(obj.Sections) != 1 {
		t.Errorf("objective = %+v", obj)
	}
	if g, _ := cfg.TypeByName("goal"); g.Base != "goal" {
		t.Errorf("short form goal = %+v", g)
	}

	for _, tc := range []struct{ yaml, want string }{
		{"plugins: []\ntypes:\n  goal: { from: fake/goal, folder: goals }", "plugin fake is not enabled; add it to plugins"},
		{"types:\n  goal: { from: fake/nothing, folder: goals }", `from "fake/nothing" does not name a plugin type`},
		{"types:\n  goal: { from: fake/goal, folder: goals, entry_timing: past }", `unknown key "entry_timing"`},
	} {
		_, err := Decode([]byte("version: 1\n"+tc.yaml+"\n"), plugins)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", tc.yaml, err, tc.want)
		}
	}
}

func TestFromResolvesAmbiguity(t *testing.T) {
	both := []core.Plugin{templates{}, badTemplates{name: "other"}}
	_, err := Decode([]byte("version: 1\nplugins: [fake, other]\ntypes:\n  goal: { from: other/goal, folder: goals }\n"), both)
	if err == nil || strings.Contains(err.Error(), "provided by both") {
		t.Fatalf("from: should resolve the ambiguity (the other error is the bad template): %v", err)
	}
}

func TestValidators(t *testing.T) {
	all := []core.Plugin{templates{}, validating{}}
	cfg, err := Decode([]byte("version: 1\ntypes:\n  news: { folder: news, entry_timing: past, validators: [lint] }\n"), all)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := cfg.TypeByName("news"); strings.Join(n.Validators, ",") != "lint" {
		t.Errorf("news = %+v", n)
	}
	for _, tc := range []struct{ yaml, want string }{
		{"types:\n  news: { folder: news, entry_timing: past, validators: [nope] }", `unknown validator plugin "nope"`},
		{"plugins: [fake]\ntypes:\n  news: { folder: news, entry_timing: past, validators: [lint] }", "plugin lint is not enabled"},
		{"types:\n  news: { folder: news, entry_timing: past, validators: [fake] }", "plugin fake has no validation rules"},
	} {
		_, err := Decode([]byte("version: 1\n"+tc.yaml+"\n"), all)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", tc.yaml, err, tc.want)
		}
	}
}
