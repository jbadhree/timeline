package core

import (
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

// CheckInput is everything a dry-run check needs. Docs must include every
// event file under the configured folders, because some rules (duplicate IDs,
// links) depend on the whole root.
type CheckInput struct {
	Config *Config
	Docs   []Document
	// Selected limits which files are reported, as root-relative paths. Nil
	// means every file. Root-wide rules still consider every document.
	Selected []string
	// Now is the entry time a draft would receive if it were recorded now.
	Now time.Time
	// Plugins add rules after the core ones. See Plugin.
	Plugins []Plugin
}

// EventResult summarizes one checked event file.
type EventResult struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Draft bool   `json:"draft"`
	Valid bool   `json:"valid"`
}

// CheckResult is the outcome of a dry-run check.
type CheckResult struct {
	Events      []EventResult `json:"events"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
}

// Valid reports whether no diagnostics were found.
func (r *CheckResult) Valid() bool { return len(r.Diagnostics) == 0 }

// event is the validator's working state for one document.
type event struct {
	doc   *Document
	typ   *TypeDef
	id    string
	draft bool
}

type validator struct {
	in     CheckInput
	events []*event
	byPath map[string]*event
	diags  []Diagnostic
}

// Check validates every document and the rules that span the root, then runs
// each plugin's validator. It never assigns IDs or entry times and has no side
// effects. An error means a plugin failed, not that an event is invalid.
func Check(in CheckInput) (CheckResult, error) {
	v := &validator{in: in, byPath: map[string]*event{}}
	for i := range in.Docs {
		d := &in.Docs[i]
		t, ok := in.Config.TypeForPath(d.Path)
		if !ok {
			v.add(Diagnostic{Path: d.Path, Message: "file is not inside a configured type folder"})
			continue
		}
		e := &event{doc: d, typ: t}
		v.events = append(v.events, e)
		v.byPath[d.Path] = e
	}
	sort.Slice(v.events, func(i, j int) bool { return v.events[i].doc.Path < v.events[j].doc.Path })

	for _, e := range v.events {
		v.checkEvent(e)
	}
	v.checkDuplicateIDs()
	v.checkLinks()
	if err := v.runPlugins(); err != nil {
		return CheckResult{}, err
	}

	return v.result(), nil
}

func (v *validator) add(d Diagnostic) { v.diags = append(v.diags, d) }

func (v *validator) result() CheckResult {
	selected := func(p string) bool { return true }
	if v.in.Selected != nil {
		set := map[string]bool{}
		for _, p := range v.in.Selected {
			set[path.Clean(p)] = true
		}
		selected = func(p string) bool { return set[p] }
	}

	res := CheckResult{Events: []EventResult{}, Diagnostics: []Diagnostic{}}
	invalid := map[string]bool{}
	for _, d := range v.diags {
		if selected(d.Path) {
			res.Diagnostics = append(res.Diagnostics, d)
			invalid[d.Path] = true
		}
	}
	SortDiagnostics(res.Diagnostics)
	for _, e := range v.events {
		if !selected(e.doc.Path) {
			continue
		}
		res.Events = append(res.Events, EventResult{
			Path: e.doc.Path, Type: e.typ.Name, ID: e.id, Draft: e.draft, Valid: !invalid[e.doc.Path],
		})
	}
	return res
}

func (v *validator) checkEvent(e *event) {
	d := e.doc
	p := d.Path
	if d.ParseErr != nil {
		v.add(*d.ParseErr)
		e.draft = true
		return
	}

	seen := map[string]bool{}
	for _, f := range d.Fields {
		if seen[f.Name] {
			v.add(Diagnostic{Path: p, Line: f.Line, Field: f.Name, Message: "appears more than once"})
		}
		seen[f.Name] = true
	}

	// type
	if f, ok := d.Field(FieldType); !ok {
		v.add(Diagnostic{Path: p, Field: FieldType, Message: fmt.Sprintf("is required; files in %s/ have type %q", e.typ.Folder, e.typ.Name)})
	} else if f.Value.Kind != ValueString || f.Value.Text != e.typ.Name {
		v.add(Diagnostic{Path: p, Line: f.Line, Field: FieldType, Message: fmt.Sprintf("is %q, but files in %s/ have type %q", f.Value.Text, e.typ.Folder, e.typ.Name)})
	}

	eventTime, eventTimeOK := v.timeField(p, d, FieldTimelineAt, true)
	v.typedField(p, d, FieldDef{Name: FieldCreatedBy, Kind: KindString, Required: true, NonEmpty: true})

	enteredAt, recorded := v.recordedMetadata(e)

	// Type-specific fields, status, and unknown fields.
	for _, fd := range e.typ.Fields {
		v.typedField(p, d, fd)
	}
	if e.typ.Status != nil {
		v.typedField(p, d, FieldDef{Name: FieldStatus, Kind: KindString, Required: true, Values: e.typ.Status.Values})
	}
	for _, f := range d.Fields {
		if IsSharedField(f.Name) {
			continue
		}
		if _, ok := e.typ.Field(f.Name); ok {
			continue
		}
		if f.Name == FieldStatus && e.typ.Status != nil {
			continue
		}
		v.add(Diagnostic{Path: p, Line: f.Line, Field: f.Name, Message: fmt.Sprintf("is not a field of type %q", e.typ.Name)})
	}

	// Timing: a draft is checked against the time it would be recorded now; a
	// recorded event against its own entry time.
	if eventTimeOK {
		entry := v.in.Now
		if recorded {
			entry = enteredAt // zero if entered_at was missing or invalid, already reported
		}
		if !entry.IsZero() {
			if err := CheckTiming(e.typ.Timing, eventTime, entry, e.typ.MaxDelay); err != nil {
				f, _ := d.Field(FieldTimelineAt)
				msg := err.Error()
				if !recorded {
					msg += " (checked against the current time, when this draft would be recorded)"
				}
				v.add(Diagnostic{Path: p, Line: f.Line, Field: FieldTimelineAt, Message: msg})
			}
		}
	}

	for _, s := range e.typ.Sections {
		if _, ok := d.Section(s); !ok {
			v.add(Diagnostic{Path: p, Section: s, Message: "is required"})
		}
	}

}

// recordedMetadata checks id, entered_at, and entry_timing. They are written
// together by tline check, so a file has all three or none. It returns the
// entry time and whether the file claims to be recorded.
func (v *validator) recordedMetadata(e *event) (time.Time, bool) {
	d, p := e.doc, e.doc.Path
	idF, hasID := d.Field(FieldID)
	atF, hasAt := d.Field(FieldEnteredAt)
	tmF, hasTm := d.Field(FieldEntryTiming)
	if !hasID && !hasAt && !hasTm {
		e.draft = true
		return time.Time{}, false
	}
	if !hasID || !hasAt || !hasTm {
		var missing []string
		if !hasID {
			missing = append(missing, FieldID)
		}
		if !hasAt {
			missing = append(missing, FieldEnteredAt)
		}
		if !hasTm {
			missing = append(missing, FieldEntryTiming)
		}
		v.add(Diagnostic{Path: p, Message: fmt.Sprintf("has some recorded metadata but is missing %s; tline check writes id, entered_at, and entry_timing together, so a draft must omit all three", strings.Join(missing, ", "))})
	}

	if hasID {
		if idF.Value.Kind != ValueString || !ValidULID(idF.Value.Text) {
			v.add(Diagnostic{Path: p, Line: idF.Line, Field: FieldID, Message: "must be a 26-character ULID"})
		} else {
			e.id = idF.Value.Text
		}
	}
	var at time.Time
	if hasAt {
		t, err := parseTimeValue(atF.Value)
		switch {
		case err != nil:
			v.add(Diagnostic{Path: p, Line: atF.Line, Field: FieldEnteredAt, Message: err.Error()})
		case !strings.HasSuffix(atF.Value.Text, "Z"):
			v.add(Diagnostic{Path: p, Line: atF.Line, Field: FieldEnteredAt, Message: "must be written in UTC with a Z suffix"})
		default:
			at = t
		}
	}
	if hasTm {
		r := TimingRule(tmF.Value.Text)
		switch {
		case tmF.Value.Kind != ValueString || !r.Valid():
			v.add(Diagnostic{Path: p, Line: tmF.Line, Field: FieldEntryTiming, Message: "must be past, future, or real_time"})
		case r != e.typ.Timing:
			v.add(Diagnostic{Path: p, Line: tmF.Line, Field: FieldEntryTiming, Message: fmt.Sprintf("is %q, but type %q uses %q", r, e.typ.Name, e.typ.Timing)})
		}
	}
	return at, true
}

func (v *validator) timeField(p string, d *Document, name string, required bool) (time.Time, bool) {
	f, ok := d.Field(name)
	if !ok || f.Value.Kind == ValueNull {
		if required {
			v.add(Diagnostic{Path: p, Field: name, Message: "is required"})
		}
		return time.Time{}, false
	}
	t, err := parseTimeValue(f.Value)
	if err != nil {
		v.add(Diagnostic{Path: p, Line: f.Line, Field: name, Message: err.Error()})
		return time.Time{}, false
	}
	return t, true
}

func parseTimeValue(rv RawValue) (time.Time, error) {
	if rv.Kind != ValueString && rv.Kind != ValueTimestamp {
		return time.Time{}, fmt.Errorf("must be an RFC 3339 timestamp, got %s", kindName(rv.Kind))
	}
	return ParseTimestamp(rv.Text)
}

// typedField checks one declared field: presence, kind, and allowed values.
func (v *validator) typedField(p string, d *Document, fd FieldDef) {
	f, ok := d.Field(fd.Name)
	if !ok || f.Value.Kind == ValueNull {
		if fd.Required {
			v.add(Diagnostic{Path: p, Field: fd.Name, Message: "is required"})
		}
		return
	}
	if fd.Kind == KindList {
		if f.Value.Kind != ValueList {
			v.add(Diagnostic{Path: p, Line: f.Line, Field: fd.Name, Message: fmt.Sprintf("must be a list, got %s", kindName(f.Value.Kind))})
			return
		}
		for _, item := range f.Value.Items {
			if msg := checkScalar(fd.Items, item, fd.Values, false); msg != "" {
				v.add(Diagnostic{Path: p, Line: item.Line, Field: fd.Name, Message: "item " + msg})
			}
		}
		return
	}
	if msg := checkScalar(fd.Kind, f.Value, fd.Values, fd.NonEmpty); msg != "" {
		v.add(Diagnostic{Path: p, Line: f.Line, Field: fd.Name, Message: msg})
	}
}

// checkScalar returns a problem description, or "" if rv fits kind k.
func checkScalar(k FieldKind, rv RawValue, values []string, nonEmpty bool) string {
	switch k {
	case KindString:
		if rv.Kind != ValueString {
			return fmt.Sprintf("must be a string, got %s (quote it if it should be text)", kindName(rv.Kind))
		}
		if nonEmpty && strings.TrimSpace(rv.Text) == "" {
			return "must not be empty"
		}
	case KindNumber:
		if rv.Kind != ValueInt && rv.Kind != ValueFloat {
			return fmt.Sprintf("must be a number, got %s", kindName(rv.Kind))
		}
	case KindBool:
		if rv.Kind != ValueBool {
			return fmt.Sprintf("must be true or false, got %s", kindName(rv.Kind))
		}
	case KindTime:
		if _, err := parseTimeValue(rv); err != nil {
			return err.Error()
		}
	}
	if len(values) > 0 && !contains(values, rv.Text) {
		return fmt.Sprintf("is %q; allowed values are %s", rv.Text, strings.Join(values, ", "))
	}
	return ""
}

func kindName(k ValueKind) string {
	switch k {
	case ValueNull:
		return "an empty value"
	case ValueString:
		return "a string"
	case ValueInt, ValueFloat:
		return "a number"
	case ValueBool:
		return "true or false"
	case ValueTimestamp:
		return "a timestamp"
	case ValueList:
		return "a list"
	case ValueMap:
		return "a mapping"
	}
	return "an unknown value"
}

func (v *validator) checkDuplicateIDs() {
	byID := map[string][]*event{}
	for _, e := range v.events {
		if e.id != "" {
			byID[e.id] = append(byID[e.id], e)
		}
	}
	for id, es := range byID {
		if len(es) < 2 {
			continue
		}
		for _, e := range es {
			var others []string
			for _, o := range es {
				if o != e {
					others = append(others, o.doc.Path)
				}
			}
			f, _ := e.doc.Field(FieldID)
			v.add(Diagnostic{Path: e.doc.Path, Line: f.Line, Field: FieldID, Message: fmt.Sprintf("%s is also used by %s", id, strings.Join(others, ", "))})
		}
	}
}

// linkResult says what a link points to.
type linkResult int

const (
	linkOther   linkResult = iota // URL, anchor, or file outside the configured folders
	linkMissing                   // into a configured folder, but no such event
	linkEvent                     // an event file in the root
)

// resolveLink returns the event a relative link points to. It reports
// nothing; checkLinks reports missing targets once for every section.
func (v *validator) resolveLink(e *event, l Link) (*event, linkResult) {
	t := l.Target
	if i := strings.IndexByte(t, '#'); i >= 0 {
		t = t[:i]
	}
	if t == "" || strings.Contains(t, "://") || strings.HasPrefix(t, "mailto:") || strings.HasPrefix(t, "/") {
		return nil, linkOther
	}
	if u, err := url.PathUnescape(t); err == nil {
		t = u // [x](list%20files.md)
	}
	p := path.Clean(path.Join(path.Dir(e.doc.Path), t))
	if p == ".." || strings.HasPrefix(p, "../") {
		return nil, linkOther
	}
	if _, inFolder := v.in.Config.TypeForPath(p); !inFolder || !strings.HasSuffix(p, ".md") {
		return nil, linkOther
	}
	target, found := v.byPath[p]
	if !found {
		return nil, linkMissing
	}
	return target, linkEvent
}

// checkLinks reports every link, in any section of any event, that points
// into a configured folder but matches no event file. Timeline gives links no
// meaning here; it only checks that they lead somewhere.
func (v *validator) checkLinks() {
	for _, e := range v.events {
		for _, s := range e.doc.Sections {
			for _, l := range s.Links {
				if _, r := v.resolveLink(e, l); r == linkMissing {
					t := l.Target
					if i := strings.IndexByte(t, '#'); i >= 0 {
						t = t[:i]
					}
					if u, err := url.PathUnescape(t); err == nil {
						t = u
					}
					v.add(Diagnostic{Path: e.doc.Path, Line: l.Line, Section: s.Heading, Message: fmt.Sprintf("link %q does not resolve to an event (looked for %s)", l.Target, path.Clean(path.Join(path.Dir(e.doc.Path), t)))})
				}
			}
		}
	}
}
