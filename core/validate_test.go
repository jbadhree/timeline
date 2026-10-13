package core

import (
	"strings"
	"testing"
	"time"
)

// The tests build documents directly; internal/document tests cover parsing.
// The goal and task types here are plain configured types: their rules live
// in the goals plugin, whose tests are in internal/plugins/goals.

var testNow = time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)

func testConfig() *Config {
	return &Config{Version: 1, Name: "test", Types: []TypeDef{
		{Name: "goal", Folder: "goals", Timing: Future, Fields: []FieldDef{
			{Name: "goal", Kind: KindString, Required: true, NonEmpty: true},
			{Name: "owner", Kind: KindString, Required: true, NonEmpty: true},
		}, Status: &StatusDef{Values: []string{"open", "achieved"}, Closed: []string{"achieved"}}},
		{Name: "task", Folder: "tasks", Timing: Future, Fields: []FieldDef{
			{Name: "task", Kind: KindString, Required: true, NonEmpty: true},
			{Name: "owner", Kind: KindString, Required: true, NonEmpty: true},
		}, Status: &StatusDef{Values: []string{"open", "done"}, Closed: []string{"done"}}},
		{Name: "news", Folder: "news", Timing: Past, Fields: []FieldDef{
			{Name: "ticker", Kind: KindString, Required: true},
			{Name: "score", Kind: KindNumber},
			{Name: "tags", Kind: KindList, Items: KindString, Values: []string{"a", "b"}},
		}},
		{Name: "observation", Folder: "observations", Timing: RealTime, MaxDelay: 15 * time.Minute,
			Status: &StatusDef{Values: []string{"open", "closed"}, Closed: []string{"closed"}}, Sections: []string{"Source"}},
	}}
}

func str(name, v string) Field { return Field{Name: name, Value: RawValue{Kind: ValueString, Text: v}} }

func stamp(name, v string) Field {
	return Field{Name: name, Value: RawValue{Kind: ValueTimestamp, Text: v}}
}

func items(heading string, texts ...string) Section {
	s := Section{Heading: heading, Body: strings.Join(texts, "\n")}
	for _, t := range texts {
		it := ListItem{Text: t}
		if i := strings.Index(t, "]("); i >= 0 {
			end := strings.Index(t[i:], ")")
			it.Links = []Link{{Target: t[i+2 : i+end]}}
		}
		s.Items = append(s.Items, it)
		s.Links = append(s.Links, it.Links...)
	}
	return s
}

func prose(heading, body string, links ...string) Section {
	s := Section{Heading: heading, Body: body}
	for _, l := range links {
		s.Links = append(s.Links, Link{Target: l})
	}
	return s
}

func goalDoc(path string, sections ...Section) Document {
	return Document{Path: path, Fields: []Field{
		str("type", "goal"), stamp("timeline_at", "2026-12-31T23:59:59Z"), str("created_by", "agent"),
		str("goal", "Ship it"), str("owner", "Ada"), str("status", "open"),
	}, Sections: sections}
}

// stepTask is the task fullGoal links to by default.
var stepTask = taskDoc("tasks/step.md", items("How", "Do the work."))

func fullGoal(path string, how ...string) Document {
	if len(how) == 0 {
		how = []string{"[Do the work](../tasks/step.md)"}
	}
	return goalDoc(path,
		prose("Why", "Because."),
		items("How", how...),
		items("Done when", "It shipped."),
		prose("Why it will work", "It is simple."),
	)
}

func taskDoc(path string, sections ...Section) Document {
	return Document{Path: path, Fields: []Field{
		str("type", "task"), stamp("timeline_at", "2026-12-01T00:00:00Z"), str("created_by", "agent"),
		str("task", "List files"), str("owner", "Ada"), str("status", "open"),
	}, Sections: sections}
}

func check(t *testing.T, docs ...Document) CheckResult {
	t.Helper()
	return mustCheck(t, CheckInput{Config: testConfig(), Docs: docs, Now: testNow})
}

func mustCheck(t *testing.T, in CheckInput) CheckResult {
	t.Helper()
	res, err := Check(in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// wantProblems asserts that each substring appears in some diagnostic for
// path, and that no other paths have diagnostics.
func wantProblems(t *testing.T, res CheckResult, path string, substrs ...string) {
	t.Helper()
	var msgs []string
	for _, d := range res.Diagnostics {
		if d.Path != path {
			t.Errorf("unexpected diagnostic for another file: %s", d)
			continue
		}
		msgs = append(msgs, d.String())
	}
	all := strings.Join(msgs, "\n")
	for _, s := range substrs {
		if !strings.Contains(all, s) {
			t.Errorf("no diagnostic for %s contains %q; got:\n%s", path, s, all)
		}
	}
	if len(substrs) == 0 && len(msgs) > 0 {
		t.Errorf("want no diagnostics, got:\n%s", all)
	}
}

func TestGoalLinks(t *testing.T) {
	g := fullGoal("goals/g.md", "[Missing](missing.md)")
	g.Sections[2] = items("Done when", "[Outside](../README.md)", "[Web](https://example.com)", "[Anchor](#why)", "[Gone](../tasks/gone.md)")
	res := check(t, g)
	wantProblems(t, res, "goals/g.md",
		`section "How": link "missing.md" does not resolve to an event (looked for goals/missing.md)`,
		`section "Done when": link "../tasks/gone.md" does not resolve to an event (looked for tasks/gone.md)`,
	)
	// A missing link is reported once, not also as an unlinked step; links in
	// Done when to files outside the folders, URLs, and anchors are fine.
	if len(res.Diagnostics) != 2 {
		t.Errorf("got %v", res.Diagnostics)
	}
}

func TestDraftTimingUsesNow(t *testing.T) {
	g := fullGoal("goals/g.md")
	g.Fields[1] = stamp("timeline_at", "2026-10-12T12:00:00Z") // equal to now: not in the future
	news := Document{Path: "news/n.md", Fields: []Field{
		str("type", "news"), stamp("timeline_at", "2026-10-13T00:00:00Z"), str("created_by", "a"), str("ticker", "X"),
	}}
	res := check(t, g, news, stepTask)
	if len(res.Diagnostics) != 2 {
		t.Fatalf("got %v", res.Diagnostics)
	}
	for _, d := range res.Diagnostics {
		if d.Field != "timeline_at" || !strings.Contains(d.Message, "checked against the current time") {
			t.Errorf("unexpected: %s", d)
		}
	}
}

func TestRecordedMetadata(t *testing.T) {
	news := func(extra ...Field) Document {
		return Document{Path: "news/n.md", Fields: append([]Field{
			str("type", "news"), stamp("timeline_at", "2026-10-01T13:00:00Z"), str("created_by", "a"), str("ticker", "X"),
		}, extra...)}
	}
	id := str("id", "01M3VW7KR0FFWVGBE1D5GYN31R")

	ok := check(t, news(id, stamp("entered_at", "2026-10-01T14:00:00Z"), str("entry_timing", "past")))
	wantProblems(t, ok, "")
	if ok.Events[0].Draft || ok.Events[0].ID == "" {
		t.Errorf("stamped event not reported as recorded: %+v", ok)
	}

	wantProblems(t, check(t, news(id)), "news/n.md", "missing entered_at, entry_timing")
	wantProblems(t, check(t, news(id, stamp("entered_at", "2026-10-01T14:00:00+00:00"), str("entry_timing", "past"))),
		"news/n.md", "UTC with a Z suffix")
	wantProblems(t, check(t, news(id, stamp("entered_at", "2026-10-01T14:00:00Z"), str("entry_timing", "future"))),
		"news/n.md", `type "news" uses "past"`)
	// Timing is judged against the recorded entry time, not now.
	wantProblems(t, check(t, news(id, stamp("entered_at", "2026-10-01T12:00:00Z"), str("entry_timing", "past"))),
		"news/n.md", "must happen before it is entered")
	wantProblems(t, check(t, news(str("id", "not-a-ulid"), stamp("entered_at", "2026-10-01T14:00:00Z"), str("entry_timing", "past"))),
		"news/n.md", "26-character ULID")
}

func TestDuplicateIDs(t *testing.T) {
	doc := func(path string) Document {
		return Document{Path: path, Fields: []Field{
			str("id", "01M3VW7KR0FFWVGBE1D5GYN31R"), stamp("entered_at", "2026-10-01T14:00:00Z"), str("entry_timing", "past"),
			str("type", "news"), stamp("timeline_at", "2026-10-01T13:00:00Z"), str("created_by", "a"), str("ticker", "X"),
		}}
	}
	res := check(t, doc("news/a.md"), doc("news/b.md"))
	if len(res.Diagnostics) != 2 || !strings.Contains(res.Diagnostics[0].Message, "also used by news/b.md") {
		t.Errorf("got %v", res.Diagnostics)
	}
}

func TestFieldTypes(t *testing.T) {
	d := Document{Path: "news/n.md", Fields: []Field{
		str("type", "news"), stamp("timeline_at", "2026-10-01T13:00:00Z"), str("created_by", "a"),
		{Name: "ticker", Value: RawValue{Kind: ValueInt, Text: "42"}},
		str("score", "high"),
		{Name: "tags", Value: RawValue{Kind: ValueList, Items: []RawValue{{Kind: ValueString, Text: "a"}, {Kind: ValueString, Text: "z"}}}},
		str("typo", "x"),
		str("created_by", "b"),
	}}
	wantProblems(t, check(t, d), "news/n.md",
		`field "ticker": must be a string, got a number`,
		`field "score": must be a number`,
		`field "tags": item is "z"; allowed values are a, b`,
		`field "typo": is not a field of type "news"`,
		`field "created_by": appears more than once`,
	)
}

func TestWrongTypeForFolderAndRequiredSections(t *testing.T) {
	d := Document{Path: "observations/o.md", Fields: []Field{
		str("type", "news"), stamp("timeline_at", "2026-10-12T11:50:00Z"), str("created_by", "a"), str("status", "pending"),
	}}
	wantProblems(t, check(t, d), "observations/o.md",
		`is "news", but files in observations/ have type "observation"`,
		`field "status": is "pending"; allowed values are open, closed`,
		`section "Source": is required`,
	)
}

func TestSelectedLimitsReportingButNotRootWideRules(t *testing.T) {
	broken := taskDoc("tasks/broken.md")
	broken.Fields = broken.Fields[:4] // no owner, no status
	dup := func(path string) Document {
		d := taskDoc(path)
		d.Fields = append([]Field{str("id", "01M3VW7KR0FFWVGBE1D5GYN31R"), stamp("entered_at", "2026-10-01T14:00:00Z"), str("entry_timing", "future")}, d.Fields...)
		return d
	}
	res := mustCheck(t, CheckInput{
		Config:   testConfig(),
		Docs:     []Document{taskDoc("tasks/t.md"), broken, dup("tasks/x.md"), dup("tasks/y.md")},
		Selected: []string{"tasks/t.md", "tasks/x.md"},
		Now:      testNow,
	})
	// tasks/broken.md has problems but is not selected; the duplicate ID in
	// tasks/x.md is found through the unselected tasks/y.md.
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Path != "tasks/x.md" || !strings.Contains(res.Diagnostics[0].Message, "also used by tasks/y.md") {
		t.Fatalf("got %+v", res.Diagnostics)
	}
	if len(res.Events) != 2 {
		t.Errorf("events = %+v", res.Events)
	}
}

func TestSelectedNothingElse(t *testing.T) {
	res := mustCheck(t, CheckInput{
		Config:   testConfig(),
		Docs:     []Document{taskDoc("tasks/t.md"), taskDoc("tasks/u.md")},
		Selected: []string{"tasks/t.md"},
		Now:      testNow,
	})
	if len(res.Diagnostics) != 0 || len(res.Events) != 1 || res.Events[0].Path != "tasks/t.md" {
		t.Errorf("got %+v", res)
	}
}

func TestParseErrorIsReported(t *testing.T) {
	bad := Document{Path: "goals/bad.md", ParseErr: &Diagnostic{Path: "goals/bad.md", Line: 1, Message: "must start with YAML frontmatter"}}
	wantProblems(t, check(t, bad), "goals/bad.md", "must start with YAML frontmatter")
}

func TestLinksInAnySectionMustResolve(t *testing.T) {
	g := fullGoal("goals/g.md")
	g.Sections[0] = prose("Why", "Because of a decision.", "../goals/decided.md", "../goals/gone.md", "https://example.com")
	res := check(t, g, stepTask, fullGoal("goals/decided.md"))
	wantProblems(t, res, "goals/g.md", `section "Why": link "../goals/gone.md" does not resolve`)
	if len(res.Diagnostics) != 1 {
		t.Errorf("got %v", res.Diagnostics)
	}
}
