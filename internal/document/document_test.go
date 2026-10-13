package document

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jbadhree/timeline/core"
)

const goal = `---
type: goal
timeline_at: 2026-12-31T23:59:59Z
owner: "Ada"
score: 3
tags:
  - a
  - b
empty:
---

# Title

## Why

Because.

## How

- [Record events](record-events.md)
- Plain step that
  continues here with [a link](../tasks/t.md "title").
1. Numbered step

` + "```" + `
## Not a heading
- not an item
` + "```" + `

## Done when
`

func TestParseGoal(t *testing.T) {
	d := Parse("goals/g.md", []byte(goal))
	if d.ParseErr != nil {
		t.Fatal(d.ParseErr)
	}
	kinds := map[string]core.ValueKind{
		"type": core.ValueString, "timeline_at": core.ValueTimestamp, "owner": core.ValueString,
		"score": core.ValueInt, "tags": core.ValueList, "empty": core.ValueNull,
	}
	for name, want := range kinds {
		f, ok := d.Field(name)
		if !ok || f.Value.Kind != want {
			t.Errorf("field %s = %+v, want kind %v", name, f, want)
		}
	}
	if f, _ := d.Field("timeline_at"); f.Line != 3 {
		t.Errorf("timeline_at line = %d, want 3", f.Line)
	}

	var headings []string
	for _, s := range d.Sections {
		headings = append(headings, s.Heading)
	}
	if len(headings) != 3 || headings[0] != "Why" || headings[1] != "How" || headings[2] != "Done when" {
		t.Fatalf("headings = %q", headings)
	}
	how, _ := d.Section("How")
	if how.Line != 18 {
		t.Errorf("How line = %d, want 18", how.Line)
	}
	if len(how.Items) != 3 {
		t.Fatalf("How items = %+v", how.Items)
	}
	if l := how.Items[0].Links; len(l) != 1 || l[0].Target != "record-events.md" || l[0].Line != 20 {
		t.Errorf("first item links = %+v", l)
	}
	if l := how.Items[1].Links; len(l) != 1 || l[0].Target != "../tasks/t.md" {
		t.Errorf("continuation link = %+v", l)
	}
	if done, _ := d.Section("Done when"); len(done.Items) != 0 {
		t.Errorf("Done when items = %+v", done.Items)
	}
}

func TestParseErrors(t *testing.T) {
	for name, src := range map[string]string{
		"no frontmatter": "# Just a note\n",
		"unclosed":       "---\ntype: goal\n",
		"invalid yaml":   "---\ntype: [goal\n---\n",
		"not a mapping":  "---\n- a\n- b\n---\n",
	} {
		if d := Parse("x.md", []byte(src)); d.ParseErr == nil {
			t.Errorf("%s: want a parse error", name)
		}
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("goals/b.md", "---\ntype: goal\n---\n")
	write("goals/sub/a.md", "---\ntype: goal\n---\n")
	write("goals/notes.txt", "ignored")
	write("goals/.hidden.md", "ignored")
	write("other/c.md", "---\n---\n")

	cfg := &core.Config{Types: []core.TypeDef{
		{Name: "goal", Folder: "goals"},
		{Name: "task", Folder: "tasks"}, // missing folder is empty
	}}
	docs, err := Scan(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Path != "goals/b.md" || docs[1].Path != "goals/sub/a.md" {
		t.Errorf("docs = %+v", docs)
	}
}

func TestLinkForms(t *testing.T) {
	for text, want := range map[string]string{
		`[a](plain.md)`:                    "plain.md",
		`[a](list%20files.md)`:             "list%20files.md",
		`[a](<list files.md>)`:             "list files.md",
		`[a](goal.md "Title")`:             "goal.md",
		`[a](goal.md#how)`:                 "goal.md#how",
		`[a](https://example.com/x)`:       "https://example.com/x",
		"`[code](no.md)` then [a](yes.md)": "yes.md",
	} {
		links := findLinks(text, 1)
		if len(links) != 1 || links[0].Target != want {
			t.Errorf("findLinks(%q) = %+v, want target %q", text, links, want)
		}
	}
}

func TestSectionLinksIncludeProse(t *testing.T) {
	d := Parse("goals/g.md", []byte("---\ntype: goal\n---\n\n## Why\n\nBecause [we decided](../decisions/d.md) and [also](x.md).\n\n```\n[not a link](y.md)\n```\n"))
	why, ok := d.Section("Why")
	if !ok || len(why.Links) != 2 || why.Links[0].Target != "../decisions/d.md" || why.Links[0].Line != 7 {
		t.Errorf("Why links = %+v", why)
	}
}
