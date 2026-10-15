// Package document reads event files: YAML frontmatter, level-2 body
// sections, their top-level list items, and the Markdown links in those
// items. It decodes into core.Document and never validates; core does that.
//
// Line numbers in the result refer to the file, starting at 1. Headings and
// list items inside fenced code blocks are ignored.
//
// Owning goal: goals/events-and-types.md.
package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jbadhree/timeline/core"
	"go.yaml.in/yaml/v3"
)

// Snapshot records the content hash of each scanned file, keyed by
// root-relative path, so a writer can tell whether a file changed after it
// was validated.
type Snapshot map[string][sha256.Size]byte

// Scan reads every Markdown file under the configured type folders of root.
// A missing folder is treated as empty; a folder that is a symbolic link or
// not a directory is an error. Symbolic links inside a folder are never
// followed: each is returned as a document that failed to parse, so the check
// reports it. Hidden files and directories are skipped. Results are sorted by
// path.
func Scan(ctx context.Context, root string, cfg *core.Config) ([]core.Document, Snapshot, error) {
	var docs []core.Document
	snap := Snapshot{}
	for _, t := range cfg.Types {
		dir := filepath.Join(root, filepath.FromSlash(t.Folder))
		info, err := os.Lstat(dir)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, nil, fmt.Errorf("scanning %s: %w", t.Folder, err)
		case info.Mode()&fs.ModeSymlink != 0:
			return nil, nil, fmt.Errorf("type folder %s is a symbolic link; Timeline does not follow links, so keep the folder itself in the root", t.Folder)
		case !info.IsDir():
			return nil, nil, fmt.Errorf("type folder %s is not a directory", t.Folder)
		}
		err = filepath.WalkDir(dir, func(p string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if strings.HasPrefix(entry.Name(), ".") && p != dir {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if entry.Type()&fs.ModeSymlink != 0 {
				docs = append(docs, core.Document{Path: rel, ParseErr: &core.Diagnostic{Path: rel,
					Message: "is a symbolic link; Timeline does not follow links, so keep event files and folders themselves in the type folder"}})
				return nil
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			docs = append(docs, Parse(rel, data))
			snap[rel] = sha256.Sum256(data)
			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("scanning %s: %w", t.Folder, err)
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, snap, nil
}

// Parse decodes one event file. Problems that prevent decoding are returned
// in Document.ParseErr rather than as an error, so they can be reported with
// every other diagnostic.
func Parse(path string, data []byte) core.Document {
	doc := core.Document{Path: path}
	data = bytes.TrimPrefix(data, bom)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		doc.ParseErr = &core.Diagnostic{Path: path, Line: 1, Message: "must start with YAML frontmatter: a line containing only ---"}
		return doc
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		doc.ParseErr = &core.Diagnostic{Path: path, Line: 1, Message: "frontmatter is not closed by a line containing only ---"}
		return doc
	}

	front := strings.Join(lines[1:end], "\n")
	fields, perr := parseFrontmatter(path, front)
	if perr != nil {
		doc.ParseErr = perr
		return doc
	}
	doc.Fields = fields
	doc.Sections = parseBody(lines[end+1:], end+2)
	return doc
}

// bom is the UTF-8 byte order mark some editors write at the start of a file.
var bom = []byte("\ufeff")

// frontmatterOffset converts a line within the frontmatter YAML to a file
// line: the YAML starts on line 2.
const frontmatterOffset = 1

func parseFrontmatter(path, src string) ([]core.Field, *core.Diagnostic) {
	if strings.TrimSpace(src) == "" {
		return nil, nil
	}
	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader([]byte(src)))
	if err := dec.Decode(&root); err != nil {
		return nil, &core.Diagnostic{Path: path, Line: 2, Message: "frontmatter is not valid YAML: " + err.Error()}
	}
	if len(root.Content) == 0 {
		return nil, nil
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil, &core.Diagnostic{Path: path, Line: m.Line + frontmatterOffset, Message: "frontmatter must be a mapping of field names to values"}
	}
	var fields []core.Field
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		fields = append(fields, core.Field{Name: k.Value, Value: rawValue(v), Line: k.Line + frontmatterOffset})
	}
	return fields, nil
}

func rawValue(n *yaml.Node) core.RawValue {
	rv := core.RawValue{Text: n.Value, Line: n.Line + frontmatterOffset}
	switch n.Kind {
	case yaml.SequenceNode:
		rv.Kind = core.ValueList
		for _, c := range n.Content {
			rv.Items = append(rv.Items, rawValue(c))
		}
	case yaml.MappingNode:
		rv.Kind = core.ValueMap
	case yaml.AliasNode:
		rv.Kind = core.ValueMap
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!null":
			rv.Kind = core.ValueNull
		case "!!int":
			rv.Kind = core.ValueInt
		case "!!float":
			rv.Kind = core.ValueFloat
		case "!!bool":
			rv.Kind = core.ValueBool
		case "!!timestamp":
			rv.Kind = core.ValueTimestamp
		default:
			rv.Kind = core.ValueString
		}
	}
	return rv
}

var (
	listItemRE = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+(.*)$`)

	// A link target is either <anything but >> or a run without spaces,
	// optionally followed by a quoted title.
	linkRE = regexp.MustCompile(`\[([^\]]*)\]\(\s*(?:<([^>]*)>|([^)\s]+))(?:\s+"[^"]*")?\s*\)`)
)

// parseBody splits body lines into level-2 sections. firstLine is the file
// line number of lines[0].
func parseBody(lines []string, firstLine int) []core.Section {
	var sections []core.Section
	var cur *core.Section
	var body []string
	var item *core.ListItem
	fence := ""

	flushItem := func() {
		if item != nil && cur != nil {
			item.Links = findLinks(item.Text, item.Line)
			cur.Items = append(cur.Items, *item)
		}
		item = nil
	}
	flushSection := func() {
		flushItem()
		if cur != nil {
			cur.Body = strings.Join(body, "\n")
			sections = append(sections, *cur)
		}
		cur, body = nil, nil
	}

	for i, line := range lines {
		lineNo := firstLine + i
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if isClosingFence(trimmed, fence) {
				fence = ""
			}
			body = append(body, line)
			continue
		}
		if f := openingFence(trimmed); f != "" {
			flushItem()
			fence = f
			body = append(body, line)
			continue
		}
		if strings.HasPrefix(line, "## ") || line == "##" || strings.HasPrefix(line, "# ") {
			flushSection()
			if strings.HasPrefix(line, "## ") {
				cur = &core.Section{Heading: headingText(strings.TrimPrefix(line, "## ")), Line: lineNo}
			}
			continue
		}
		body = append(body, line)
		if cur == nil {
			continue
		}
		cur.Links = append(cur.Links, findLinks(line, lineNo)...)
		switch {
		case listItemRE.MatchString(line):
			flushItem()
			item = &core.ListItem{Text: listItemRE.FindStringSubmatch(line)[1], Line: lineNo}
		case item != nil && trimmed != "" && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
			item.Text += "\n" + trimmed
		case trimmed == "":
			// A blank line keeps the item open for indented continuations.
		default:
			flushItem()
		}
	}
	flushSection()
	return sections
}

// openingFence returns the fence that a trimmed line opens: three or more
// backticks or tildes, which may be followed by an info string such as "go".
func openingFence(trimmed string) string {
	for _, c := range []byte{'`', '~'} {
		n := 0
		for n < len(trimmed) && trimmed[n] == c {
			n++
		}
		if n >= 3 {
			return trimmed[:n]
		}
	}
	return ""
}

// isClosingFence reports whether a trimmed line closes fence: only the same
// character, at least as many times, with no info string.
func isClosingFence(trimmed, fence string) bool {
	return len(trimmed) >= len(fence) && strings.Trim(trimmed, fence[:1]) == ""
}

// headingText returns a heading's text without an optional closing run of
// "#" characters, which counts only after a space: "## C#" is "C#".
func headingText(s string) string {
	s = strings.TrimSpace(s)
	if t := strings.TrimRight(s, "#"); t == "" || strings.HasSuffix(t, " ") {
		s = strings.TrimSpace(t)
	}
	return s
}

// codeSpanRE matches inline code, whose contents are never links.
var codeSpanRE = regexp.MustCompile("`+[^`]*`+")

func findLinks(text string, line int) []core.Link {
	var links []core.Link
	text = codeSpanRE.ReplaceAllString(text, "")
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		target := m[2]
		if target == "" {
			target = m[3]
		}
		links = append(links, core.Link{Text: m[1], Target: target, Line: line})
	}
	return links
}

// Stamp is what tline check writes into a draft when it records it.
type Stamp struct {
	ID          string
	EnteredAt   time.Time
	EntryTiming core.TimingRule
}

// InsertStamp returns data with the stamp's id, entered_at, and entry_timing
// inserted as the first frontmatter lines. Every other byte is unchanged, and
// the file's line ending (LF or CRLF) is kept.
func InsertStamp(data []byte, s Stamp) ([]byte, error) {
	nl := bytes.IndexByte(data, '\n')
	if nl < 0 || strings.TrimRight(string(bytes.TrimPrefix(data[:nl], bom)), " \t\r") != "---" {
		return nil, errors.New("file does not start with frontmatter")
	}
	eol := "\n"
	if nl > 0 && data[nl-1] == '\r' {
		eol = "\r\n"
	}
	stamp := "id: " + s.ID + eol +
		"entered_at: " + s.EnteredAt.UTC().Format(time.RFC3339) + eol +
		"entry_timing: " + string(s.EntryTiming) + eol
	out := make([]byte, 0, len(data)+len(stamp))
	out = append(out, data[:nl+1]...)
	out = append(out, stamp...)
	out = append(out, data[nl+1:]...)
	return out, nil
}

// Rewrite replaces the file at path with data, but only if its current
// content still hashes to want. It writes a temporary file in the same
// directory and renames it over the original, keeping the file mode, so
// readers see either the old or the new file, never a partial one.
func Rewrite(path string, want [sha256.Size]byte, data []byte) error {
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if sha256.Sum256(current) != want {
		return ErrChanged
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tline-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func(err error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return cleanup(err)
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// ErrChanged means a file changed after it was validated.
var ErrChanged = errors.New("file changed after it was validated")
