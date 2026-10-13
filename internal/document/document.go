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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jbadhree/timeline/core"
	"go.yaml.in/yaml/v3"
)

// Scan reads every Markdown file under the configured type folders of root.
// A missing folder is treated as empty. Hidden files and directories are
// skipped. Results are sorted by path.
func Scan(ctx context.Context, root string, cfg *core.Config) ([]core.Document, error) {
	var docs []core.Document
	for _, t := range cfg.Types {
		dir := filepath.Join(root, filepath.FromSlash(t.Folder))
		err := filepath.WalkDir(dir, func(p string, entry fs.DirEntry, err error) error {
			if err != nil {
				if p == dir && os.IsNotExist(err) {
					return filepath.SkipDir
				}
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
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			docs = append(docs, Parse(filepath.ToSlash(rel), data))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", t.Folder, err)
		}
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, nil
}

// Parse decodes one event file. Problems that prevent decoding are returned
// in Document.ParseErr rather than as an error, so they can be reported with
// every other diagnostic.
func Parse(path string, data []byte) core.Document {
	doc := core.Document{Path: path}
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
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			body = append(body, line)
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			flushItem()
			fence = trimmed[:3]
			body = append(body, line)
			continue
		}
		if strings.HasPrefix(line, "## ") || line == "##" || strings.HasPrefix(line, "# ") {
			flushSection()
			if strings.HasPrefix(line, "## ") {
				cur = &core.Section{Heading: strings.TrimSpace(strings.TrimRight(strings.TrimPrefix(line, "## "), "# ")), Line: lineNo}
			}
			continue
		}
		body = append(body, line)
		if cur == nil {
			continue
		}
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

func findLinks(text string, line int) []core.Link {
	var links []core.Link
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		target := m[2]
		if target == "" {
			target = m[3]
		}
		links = append(links, core.Link{Text: m[1], Target: target, Line: line})
	}
	return links
}
