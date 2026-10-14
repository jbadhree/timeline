package core

import (
	"strings"
	"testing"
)

func historyType() *Config {
	cfg := testConfig()
	cfg.Types = append(cfg.Types, TypeDef{Name: "prediction", Folder: "predictions", Timing: Future, History: true,
		Frozen: []string{"tags", "## Reasoning"},
		Fields: []FieldDef{{Name: "tags", Kind: KindList, Items: KindString}, {Name: "note", Kind: KindString}}})
	return cfg
}

func predictionDoc(tags []string, reasoning, note string) Document {
	var items []RawValue
	for _, tg := range tags {
		items = append(items, RawValue{Kind: ValueString, Text: tg})
	}
	d := Document{Path: "predictions/p.md", Fields: []Field{
		str("id", "01M3VW7KR0FFWVGBE1D5GYN31R"), stamp("entered_at", "2026-10-01T00:00:00Z"), str("entry_timing", "future"),
		str("type", "prediction"), stamp("timeline_at", "2099-01-01T00:00:00Z"), str("created_by", "a"),
		{Name: "tags", Value: RawValue{Kind: ValueList, Items: items}},
	}}
	if note != "" {
		d.Fields = append(d.Fields, str("note", note))
	}
	if reasoning != "" {
		d.Sections = []Section{{Heading: "Reasoning", Body: reasoning}}
	}
	return d
}

func TestFrozenListAndSectionAgainstFirstVersion(t *testing.T) {
	first := predictionDoc([]string{"a", "b"}, "\nBecause.\n", "")
	check := func(current Document) []Diagnostic {
		t.Helper()
		return mustCheck(t, CheckInput{Config: historyType(), Docs: []Document{current}, Now: testNow,
			History: History{FirstVersions: map[string]*Document{"01M3VW7KR0FFWVGBE1D5GYN31R": &first}}}).Diagnostics
	}
	// Surrounding blank lines and unfrozen fields may change.
	if d := check(predictionDoc([]string{"a", "b"}, "Because.", "added later")); len(d) != 0 {
		t.Errorf("allowed changes rejected: %v", d)
	}
	d := check(predictionDoc([]string{"a"}, "", ""))
	all := ""
	for _, x := range d {
		all += x.String() + "\n"
	}
	if len(d) != 2 || !strings.Contains(all, `field "tags": is frozen; the first recorded version had [a, b]`) ||
		!strings.Contains(all, `section "Reasoning": is frozen; the first recorded version had "Because."`) {
		t.Errorf("got:\n%s", all)
	}
}

func TestNoFirstVersionMeansNoComparison(t *testing.T) {
	// History was just enabled: nothing to compare yet.
	res := mustCheck(t, CheckInput{Config: historyType(), Docs: []Document{predictionDoc(nil, "", "")}, Now: testNow})
	if len(res.Diagnostics) != 0 {
		t.Errorf("got %v", res.Diagnostics)
	}
}

func TestChangedParts(t *testing.T) {
	older := predictionDoc([]string{"a"}, "Because.", "first")
	newer := predictionDoc([]string{"a"}, "Because, revised.", "")
	newer.Sections = append(newer.Sections, Section{Heading: "Outcome", Body: "It happened."})
	fields, sections := ChangedParts(&older, &newer)
	if strings.Join(fields, ",") != "note" || strings.Join(sections, ",") != "Reasoning,Outcome" {
		t.Errorf("fields %v, sections %v", fields, sections)
	}
	if f, s := ChangedParts(&older, &older); len(f) != 0 || len(s) != 0 {
		t.Errorf("identical versions changed: %v %v", f, s)
	}
}
