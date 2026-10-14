package core

import (
	"fmt"
	"strings"
)

// History facts a check needs, read from the history folder by the caller.
// See goals/event-history.md.
type History struct {
	// FirstVersions holds each history-type event's first recorded version
	// (v1), by event ID.
	FirstVersions map[string]*Document
	// LastPaths maps an event path to the ID whose latest copy was saved from
	// that path.
	LastPaths map[string]string
}

// checkHistory compares recorded events of history types with their first
// version: stamps and frozen fields and sections must not change.
func (v *validator) checkHistory() {
	h := v.in.History
	current := map[string]bool{}
	for _, e := range v.events {
		if e.id != "" {
			current[e.id] = true
		}
	}
	for _, e := range v.events {
		if !e.typ.History || e.draft || e.id == "" || e.doc.ParseErr != nil {
			continue
		}
		first := h.FirstVersions[e.id]
		if first == nil {
			if other := h.LastPaths[e.doc.Path]; other != "" && other != e.id && !current[other] {
				f, _ := e.doc.Field(FieldID)
				v.add(Diagnostic{Path: e.doc.Path, Line: f.Line, Field: FieldID, Message: fmt.Sprintf("is %s, but this file was recorded as %s; recorded stamps must not change", e.id, other)})
			}
			continue
		}
		for _, name := range []string{FieldID, FieldEnteredAt, FieldEntryTiming} {
			v.compareField(e, first, name, "changed since the event was recorded; stamps must not change")
		}
		for _, name := range e.typ.Frozen {
			if heading, ok := strings.CutPrefix(name, "## "); ok {
				v.compareSection(e, first, heading)
			} else {
				v.compareField(e, first, name, "is frozen")
			}
		}
	}
}

func (v *validator) compareField(e *event, first *Document, name, problem string) {
	was, hadIt := first.Field(name)
	now, hasIt := e.doc.Field(name)
	if hadIt == hasIt && (!hadIt || sameValue(was.Value, now.Value)) {
		return
	}
	original := "had no " + name
	if hadIt {
		original = "had " + showValue(was.Value)
	}
	line := 0
	if hasIt {
		line = now.Line
	}
	v.add(Diagnostic{Path: e.doc.Path, Line: line, Field: name, Message: fmt.Sprintf("%s; the first recorded version %s", problem, original)})
}

func (v *validator) compareSection(e *event, first *Document, heading string) {
	was, hadIt := first.Section(heading)
	now, hasIt := e.doc.Section(heading)
	if hadIt == hasIt && (!hadIt || strings.TrimSpace(was.Body) == strings.TrimSpace(now.Body)) {
		return
	}
	original := "had no such section"
	if hadIt {
		original = fmt.Sprintf("had %q", excerpt(was.Body))
	}
	line := 0
	if hasIt {
		line = now.Line
	}
	v.add(Diagnostic{Path: e.doc.Path, Line: line, Section: heading, Message: "is frozen; the first recorded version " + original})
}

// sameValue reports whether two frontmatter values are the same YAML value.
func sameValue(a, b RawValue) bool {
	if a.Kind != b.Kind || a.Text != b.Text || len(a.Items) != len(b.Items) {
		return false
	}
	for i := range a.Items {
		if !sameValue(a.Items[i], b.Items[i]) {
			return false
		}
	}
	return true
}

func showValue(rv RawValue) string {
	switch rv.Kind {
	case ValueList:
		var items []string
		for _, it := range rv.Items {
			items = append(items, it.Text)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case ValueNull:
		return "an empty value"
	}
	return fmt.Sprintf("%q", rv.Text)
}

func excerpt(body string) string {
	s := strings.Join(strings.Fields(body), " ")
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}

// ChangedParts lists the frontmatter fields and the sections whose content
// differs between two versions of an event, in the newer version's order,
// followed by those only in the older one.
func ChangedParts(older, newer *Document) (fields, sections []string) {
	seen := map[string]bool{}
	for _, f := range newer.Fields {
		seen[f.Name] = true
		if was, ok := older.Field(f.Name); !ok || !sameValue(was.Value, f.Value) {
			fields = append(fields, f.Name)
		}
	}
	for _, f := range older.Fields {
		if !seen[f.Name] {
			fields = append(fields, f.Name)
		}
	}
	seen = map[string]bool{}
	for _, s := range newer.Sections {
		seen[s.Heading] = true
		if was, ok := older.Section(s.Heading); !ok || strings.TrimSpace(was.Body) != strings.TrimSpace(s.Body) {
			sections = append(sections, s.Heading)
		}
	}
	for _, s := range older.Sections {
		if !seen[s.Heading] {
			sections = append(sections, s.Heading)
		}
	}
	return fields, sections
}
