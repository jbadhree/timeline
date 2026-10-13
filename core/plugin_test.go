package core

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// decisionsPlugin is a small plugin like one a timeline might write: a
// superseded decision must link to the decision that replaced it.
type decisionsPlugin struct {
	seen  *PluginInput // the input it was given, for inspection
	extra []Diagnostic // diagnostics to return as well
	err   error
}

func (decisionsPlugin) Name() string { return "decisions" }

func (decisionsPlugin) Types() []TypeDef {
	return []TypeDef{{Name: "decision", Timing: Past}}
}

func (p decisionsPlugin) Validate(in PluginInput) ([]Diagnostic, error) {
	if p.seen != nil {
		*p.seen = in
	}
	if p.err != nil {
		return nil, p.err
	}
	replaced := map[string]bool{}
	for _, l := range in.Links {
		if l.Section == "Superseded by" && l.ToType == "decision" {
			replaced[l.From] = true
		}
	}
	diags := append([]Diagnostic{}, p.extra...)
	for _, e := range in.Events {
		if f, ok := e.Doc.Field(FieldStatus); ok && f.Value.Text == "superseded" && !replaced[e.Path] {
			diags = append(diags, Diagnostic{Path: e.Path, Section: "Superseded by", Message: "a superseded decision must link to the decision that replaced it"})
		}
	}
	return diags, nil
}

func pluginConfig() *Config {
	cfg := testConfig()
	cfg.Types = append(cfg.Types, TypeDef{Name: "decision", Folder: "decisions", Timing: Past, Plugin: "decisions", Base: "decision",
		Status: &StatusDef{Values: []string{"current", "superseded"}, Closed: []string{"superseded"}}})
	return cfg
}

func decisionDoc(path, status string, sections ...Section) Document {
	return Document{Path: path, Fields: []Field{
		str("type", "decision"), stamp("timeline_at", "2026-10-01T00:00:00Z"), str("created_by", "a"), str("status", status),
	}, Sections: sections}
}

func TestPluginAddsDiagnostics(t *testing.T) {
	old := decisionDoc("decisions/old.md", "superseded")
	linked := decisionDoc("decisions/older.md", "superseded", Section{Heading: "Superseded by",
		Items: []ListItem{{Text: "[New](new.md)", Line: 12, Links: []Link{{Target: "new.md", Line: 12}}}},
		Links: []Link{{Target: "new.md", Line: 12}}})
	res := mustCheck(t, CheckInput{
		Config:  pluginConfig(),
		Docs:    []Document{old, linked, decisionDoc("decisions/new.md", "current")},
		Now:     testNow,
		Plugins: []Plugin{decisionsPlugin{}},
	})
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Path != "decisions/old.md" ||
		!strings.Contains(res.Diagnostics[0].Message, "must link to the decision that replaced it") {
		t.Fatalf("got %v", res.Diagnostics)
	}
	for _, e := range res.Events {
		if e.Valid != (e.Path != "decisions/old.md") {
			t.Errorf("%s valid = %v", e.Path, e.Valid)
		}
	}
}

func TestPluginSeesOnlyItsTypesAsFacts(t *testing.T) {
	var seen PluginInput
	why := Section{Heading: "Why", Body: "Because [of this](old.md).", Links: []Link{{Target: "old.md", Line: 9}}}
	how := Section{Heading: "How",
		Items: []ListItem{
			{Text: "First", Line: 12},
			{Text: "Second\nsee [goal](../goals/g.md) and [web](https://x.y)", Line: 13, Links: []Link{{Target: "../goals/g.md", Line: 13}, {Target: "https://x.y", Line: 13}}},
		},
		Links: []Link{{Target: "../goals/g.md", Line: 14}, {Target: "https://x.y", Line: 14}}, // continuation line
	}
	mustCheck(t, CheckInput{
		Config: pluginConfig(),
		Docs: []Document{
			decisionDoc("decisions/old.md", "current"),
			decisionDoc("decisions/new.md", "current", why, how),
			fullGoal("goals/g.md"), stepTask,
		},
		Now:     testNow,
		Plugins: []Plugin{decisionsPlugin{seen: &seen}},
	})
	if len(seen.Events) != 2 || seen.Events[0].Path != "decisions/new.md" || seen.Events[1].Path != "decisions/old.md" {
		t.Fatalf("events = %+v", seen.Events)
	}
	if !seen.Events[0].Draft || seen.Events[0].Type.Name != "decision" || !seen.Now.Equal(testNow) {
		t.Errorf("event facts = %+v, now %v", seen.Events[0], seen.Now)
	}
	want := []ResolvedLink{
		{From: "decisions/new.md", Section: "Why", Item: -1, Line: 9, Target: "old.md", To: "decisions/old.md", ToType: "decision"},
		{From: "decisions/new.md", Section: "How", Item: 1, Line: 14, Target: "../goals/g.md", To: "goals/g.md", ToType: "goal"},
		{From: "decisions/new.md", Section: "How", Item: 1, Line: 14, Target: "https://x.y"},
	}
	if len(seen.Links) != len(want) {
		t.Fatalf("links = %+v", seen.Links)
	}
	for i := range want {
		if seen.Links[i] != want[i] {
			t.Errorf("link %d = %+v, want %+v", i, seen.Links[i], want[i])
		}
	}
}

func TestPluginCannotJudgeOtherTypes(t *testing.T) {
	res := mustCheck(t, CheckInput{
		Config:  pluginConfig(),
		Docs:    []Document{decisionDoc("decisions/d.md", "current"), fullGoal("goals/g.md"), stepTask},
		Now:     testNow,
		Plugins: []Plugin{decisionsPlugin{extra: []Diagnostic{{Path: "goals/g.md", Message: "I dislike this goal"}}}},
	})
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Path != "goals/g.md" ||
		!strings.Contains(res.Diagnostics[0].Message, "plugin decisions reported a problem in a file that is not one of its types") {
		t.Fatalf("got %v", res.Diagnostics)
	}
}

func TestPluginFailures(t *testing.T) {
	in := CheckInput{Config: pluginConfig(), Docs: []Document{decisionDoc("decisions/d.md", "current")}, Now: testNow}

	boom := errors.New("boom")
	in.Plugins = []Plugin{decisionsPlugin{err: boom}}
	if _, err := Check(in); !errors.Is(err, boom) || !strings.Contains(err.Error(), "plugin decisions") {
		t.Errorf("failing plugin: %v", err)
	}

	in.Plugins = []Plugin{decisionsPlugin{}, decisionsPlugin{}}
	if _, err := Check(in); err == nil || !strings.Contains(err.Error(), "registered twice") {
		t.Errorf("duplicate plugin: %v", err)
	}
}

func TestPluginDiagnosticsRespectSelection(t *testing.T) {
	res := mustCheck(t, CheckInput{
		Config:   pluginConfig(),
		Docs:     []Document{decisionDoc("decisions/a.md", "superseded"), decisionDoc("decisions/b.md", "superseded")},
		Selected: []string{"decisions/b.md"},
		Now:      testNow,
		Plugins:  []Plugin{decisionsPlugin{}},
	})
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Path != "decisions/b.md" {
		t.Fatalf("got %v", res.Diagnostics)
	}
}

func TestRunCommandGivesFactsAndOwnDiagnostics(t *testing.T) {
	var got CommandInput
	cmd := Command{Name: "decision", Run: func(_ context.Context, in CommandInput) (CommandOutput, error) {
		got = in
		return CommandOutput{Text: "ok"}, nil
	}}
	badTask := taskDoc("tasks/bad.md")
	badTask.Fields = badTask.Fields[:3] // missing fields: not the plugin's problem
	out, err := RunCommand(context.Background(), CheckInput{
		Config:   pluginConfig(),
		Docs:     []Document{decisionDoc("decisions/a.md", "current"), decisionDoc("decisions/b.md", "maybe"), badTask},
		Selected: []string{"decisions/a.md"}, // ignored: a command sees the whole root
		Now:      testNow,
		Plugins:  []Plugin{decisionsPlugin{}},
	}, decisionsPlugin{}, cmd, []string{"current"}, true)
	if err != nil || out.Text != "ok" {
		t.Fatalf("out %+v, err %v", out, err)
	}
	if len(got.Events) != 2 || !got.JSON || len(got.Args) != 1 || !got.Now.Equal(testNow) {
		t.Errorf("input = %+v", got)
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].Path != "decisions/b.md" {
		t.Errorf("diagnostics = %+v", got.Diagnostics)
	}
}

func TestPastDue(t *testing.T) {
	task := testConfig().Types[1] // Future, statuses open/done
	news := testConfig().Types[2] // Past, no statuses
	obs := testConfig().Types[3]  // RealTime with statuses
	before, after := testNow.Add(-1), testNow.Add(1)
	withStatus := func(s string) *Document { return &Document{Fields: []Field{str("status", s)}} }
	for _, tc := range []struct {
		name string
		typ  TypeDef
		doc  *Document
		when time.Time
		want bool
	}{
		{"open future task whose date passed", task, withStatus("open"), before, true},
		{"open task not yet due", task, withStatus("open"), after, false},
		{"due exactly now", task, withStatus("open"), testNow, false},
		{"done task whose date passed", task, withStatus("done"), before, false},
		{"task without status", task, &Document{}, before, false},
		{"task with invalid time", task, withStatus("open"), time.Time{}, false},
		{"past type", news, withStatus("open"), before, false},
		{"real-time type with open status", obs, withStatus("open"), before, false},
	} {
		if got := PastDue(&tc.typ, tc.doc, tc.when, testNow); got != tc.want {
			t.Errorf("%s: PastDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestValidatorSeesTypesThatListIt(t *testing.T) {
	cfg := pluginConfig()
	for i := range cfg.Types {
		if cfg.Types[i].Name == "news" {
			cfg.Types[i].Validators = []string{"decisions"}
		}
	}
	var seen PluginInput
	news := Document{Path: "news/n.md", Fields: []Field{
		str("type", "news"), stamp("timeline_at", "2026-10-01T13:00:00Z"), str("created_by", "a"), str("ticker", "X"),
	}}
	mustCheck(t, CheckInput{
		Config:  cfg,
		Docs:    []Document{news, decisionDoc("decisions/d.md", "current"), stepTask},
		Now:     testNow,
		Plugins: []Plugin{decisionsPlugin{seen: &seen}},
	})
	var paths []string
	for _, e := range seen.Events {
		paths = append(paths, e.Path)
	}
	if strings.Join(paths, ",") != "decisions/d.md,news/n.md" {
		t.Errorf("plugin saw %v; want its own type and the type listing it as a validator", paths)
	}
}
