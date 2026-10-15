package document

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	docs, snap, err := Scan(context.Background(), root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].Path != "goals/b.md" || docs[1].Path != "goals/sub/a.md" {
		t.Errorf("docs = %+v", docs)
	}
	if len(snap) != 2 || snap["goals/b.md"] != sha256.Sum256([]byte("---\ntype: goal\n---\n")) {
		t.Errorf("snapshot = %v", snap)
	}
}

func TestHeadingsAndFences(t *testing.T) {
	src := "---\ntype: goal\n---\n" +
		"## C#\n## Closed ##\n" +
		"````\n```\n## Inside a four-backtick fence\n````\n" +
		"## After\n" +
		"```go\n```go\n## Inside, after a line that cannot close the fence\n```\n" +
		"~~~\n```\n## Inside a tilde fence\n~~~~\n" +
		"## Last\n"
	d := Parse("goals/g.md", []byte(src))
	var got []string
	for _, s := range d.Sections {
		got = append(got, s.Heading)
	}
	want := []string{"C#", "Closed", "After", "Last"}
	if len(got) != len(want) {
		t.Fatalf("headings = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("headings = %q, want %q", got, want)
			break
		}
	}
}

func TestByteOrderMark(t *testing.T) {
	d := Parse("goals/g.md", []byte("\ufeff---\ntype: goal\n---\n\n## Why\n\nBecause.\n"))
	if d.ParseErr != nil {
		t.Fatalf("a byte order mark should be ignored: %v", d.ParseErr)
	}
	if f, ok := d.Field("type"); !ok || f.Value.Text != "goal" || f.Line != 2 {
		t.Errorf("type = %+v", f)
	}
	if s, ok := d.Section("Why"); !ok || s.Line != 5 {
		t.Errorf("Why = %+v", s)
	}
}

func TestScanRejectsLinks(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	write := func(p, content string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := func(target, p string) {
		if err := os.Symlink(target, p); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &core.Config{Types: []core.TypeDef{{Name: "goal", Folder: "goals"}}}
	write(filepath.Join(elsewhere, "a.md"), "---\ntype: goal\n---\n")
	write(filepath.Join(root, "goals", "real.md"), "---\ntype: goal\n---\n")

	// Links inside a type folder, to a file or a folder, are reported, not read.
	link(filepath.Join(elsewhere, "a.md"), filepath.Join(root, "goals", "file.md"))
	link(elsewhere, filepath.Join(root, "goals", "folder"))
	docs, snap, err := Scan(context.Background(), root, cfg)
	if err != nil || len(docs) != 3 || len(snap) != 1 {
		t.Fatalf("docs = %+v, snap = %v, err = %v", docs, snap, err)
	}
	for _, d := range docs {
		linked := d.Path != "goals/real.md"
		if got := d.ParseErr != nil && strings.Contains(d.ParseErr.Message, "symbolic link"); got != linked {
			t.Errorf("%s: ParseErr = %v", d.Path, d.ParseErr)
		}
	}

	// A type folder that is a link, or a file, is an error, not an empty folder.
	if err := os.RemoveAll(filepath.Join(root, "goals")); err != nil {
		t.Fatal(err)
	}
	link(elsewhere, filepath.Join(root, "goals"))
	if _, _, err := Scan(context.Background(), root, cfg); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Errorf("want a symbolic link error, got %v", err)
	}
	if err := os.Remove(filepath.Join(root, "goals")); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(root, "goals"), "not a folder")
	if _, _, err := Scan(context.Background(), root, cfg); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("want a not-a-directory error, got %v", err)
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

func TestInsertStamp(t *testing.T) {
	s := Stamp{ID: "01M3VW7KR0FFWVGBE1D5GYN31R", EnteredAt: time.Date(2026, 10, 13, 6, 0, 0, 0, time.UTC), EntryTiming: core.Future}
	for name, tc := range map[string]struct{ in, want string }{
		"lf": {
			"---\ntype: task\n---\n\nBody stays   exactly.\n",
			"---\nid: 01M3VW7KR0FFWVGBE1D5GYN31R\nentered_at: 2026-10-13T06:00:00Z\nentry_timing: future\ntype: task\n---\n\nBody stays   exactly.\n",
		},
		"byte order mark kept": {
			"\ufeff---\ntype: task\n---\n",
			"\ufeff---\nid: 01M3VW7KR0FFWVGBE1D5GYN31R\nentered_at: 2026-10-13T06:00:00Z\nentry_timing: future\ntype: task\n---\n",
		},
		"crlf": {
			"---\r\ntype: task\r\n---\r\n",
			"---\r\nid: 01M3VW7KR0FFWVGBE1D5GYN31R\r\nentered_at: 2026-10-13T06:00:00Z\r\nentry_timing: future\r\ntype: task\r\n---\r\n",
		},
	} {
		got, err := InsertStamp([]byte(tc.in), s)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
	if _, err := InsertStamp([]byte("no frontmatter\n"), s); err == nil {
		t.Error("want an error without frontmatter")
	}
}

func TestRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.md")
	orig := []byte("---\ntype: task\n---\n")
	if err := os.WriteFile(path, orig, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Rewrite(path, sha256.Sum256(orig), []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	if string(got) != "new" || info.Mode().Perm() != 0o640 {
		t.Errorf("content %q mode %v", got, info.Mode().Perm())
	}
	// The file no longer matches the original hash: refuse to overwrite.
	if err := Rewrite(path, sha256.Sum256(orig), []byte("newer")); !errors.Is(err, ErrChanged) {
		t.Errorf("want ErrChanged, got %v", err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}
