package core

import (
	"fmt"
	"sort"
)

// Diagnostic is one validation problem in a file.
type Diagnostic struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Field   string `json:"field,omitempty"`
	Section string `json:"section,omitempty"`
	Message string `json:"message"`
}

func (d Diagnostic) String() string {
	loc := d.Path
	if d.Line > 0 {
		loc = fmt.Sprintf("%s:%d", loc, d.Line)
	}
	switch {
	case d.Field != "":
		return fmt.Sprintf("%s: field %q: %s", loc, d.Field, d.Message)
	case d.Section != "":
		return fmt.Sprintf("%s: section %q: %s", loc, d.Section, d.Message)
	}
	return fmt.Sprintf("%s: %s", loc, d.Message)
}

// SortDiagnostics orders diagnostics by path, line, field, section, and
// message.
func SortDiagnostics(ds []Diagnostic) {
	sort.Slice(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.Section != b.Section {
			return a.Section < b.Section
		}
		return a.Message < b.Message
	})
}
