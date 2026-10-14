package external

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jbadhree/timeline/core"
)

// Protocol is the protocol version this tline speaks.
const Protocol = 1

// Request kinds.
const (
	reqDescribe = "describe"
	reqValidate = "validate"
	reqCommand  = "command"
)

// describeRequest asks a plugin what it provides.
type describeRequest struct {
	Protocol int    `json:"protocol"`
	Request  string `json:"request"`
}

// facts are what validate and command requests carry: the same facts as
// core.PluginInput.
type facts struct {
	Now    string      `json:"now"`
	Root   string      `json:"root"`
	Config configJSON  `json:"config"`
	Events []eventJSON `json:"events"`
	Links  []linkJSON  `json:"links"`
}

type validateRequest struct {
	Protocol int    `json:"protocol"`
	Request  string `json:"request"`
	facts
}

type commandRequest struct {
	Protocol int    `json:"protocol"`
	Request  string `json:"request"`
	facts
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	JSON        bool              `json:"json"`
	Diagnostics []core.Diagnostic `json:"diagnostics"`
}

// describeResponse is what a plugin provides.
type describeResponse struct {
	Name      string         `json:"name"`
	Protocols []int          `json:"protocols"`
	Types     []templateJSON `json:"types"`
	Validates bool           `json:"validates"`
	Commands  []commandJSON  `json:"commands"`
}

type commandJSON struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Usage   string `json:"usage"`
}

type validateResponse struct {
	Diagnostics []core.Diagnostic `json:"diagnostics"`
}

type commandResponse struct {
	Text       string          `json:"text"`
	JSON       json.RawMessage `json:"json"`
	Incomplete bool            `json:"incomplete"`
	UsageError string          `json:"usage_error"`
}

// templateJSON is a type template in the shape of a timeline.yaml type.
type templateJSON struct {
	Name        string               `json:"name"`
	EntryTiming string               `json:"entry_timing"`
	MaxDelay    string               `json:"max_delay,omitempty"`
	Fields      map[string]fieldJSON `json:"fields,omitempty"`
	Sections    []string             `json:"sections,omitempty"`
	Status      *statusJSON          `json:"status,omitempty"`
}

type fieldJSON struct {
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Values   []string `json:"values,omitempty"`
	Items    string   `json:"items,omitempty"`
}

type statusJSON struct {
	Values []string `json:"values"`
	Closed []string `json:"closed"`
}

// configJSON is the timeline's configuration as plugins see it.
type configJSON struct {
	Name    string     `json:"name"`
	Plugins []string   `json:"plugins"`
	Types   []typeJSON `json:"types"`
}

type typeJSON struct {
	templateJSON
	Folder     string   `json:"folder"`
	Plugin     string   `json:"plugin,omitempty"`
	Base       string   `json:"base,omitempty"`
	Validators []string `json:"validators,omitempty"`
	History    bool     `json:"history,omitempty"`
	Frozen     []string `json:"frozen,omitempty"`
}

type eventJSON struct {
	Path       string         `json:"path"`
	Type       string         `json:"type"`
	ID         string         `json:"id,omitempty"`
	Draft      bool           `json:"draft"`
	Time       string         `json:"time,omitempty"`
	PastDue    bool           `json:"past_due"`
	Fields     map[string]any `json:"fields"`
	FieldLines map[string]int `json:"field_lines"`
	Sections   []sectionJSON  `json:"sections"`
}

type sectionJSON struct {
	Heading string     `json:"heading"`
	Line    int        `json:"line"`
	Body    string     `json:"body"`
	Items   []itemJSON `json:"items"`
}

type itemJSON struct {
	Text string `json:"text"`
	Line int    `json:"line"`
}

type linkJSON struct {
	From    string `json:"from"`
	Section string `json:"section"`
	Item    int    `json:"item"`
	Line    int    `json:"line"`
	Target  string `json:"target"`
	To      string `json:"to,omitempty"`
	ToType  string `json:"to_type,omitempty"`
	Broken  bool   `json:"broken"`
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// encodeFacts converts core.PluginInput to its JSON form.
func encodeFacts(root string, in core.PluginInput) facts {
	f := facts{Now: rfc3339(in.Now), Root: root, Config: encodeConfig(in.Config), Events: []eventJSON{}, Links: []linkJSON{}}
	for _, e := range in.Events {
		ev := eventJSON{
			Path: e.Path, Type: e.Type.Name, ID: e.ID, Draft: e.Draft, Time: rfc3339(e.Time), PastDue: e.PastDue,
			Fields: map[string]any{}, FieldLines: map[string]int{}, Sections: []sectionJSON{},
		}
		for _, fl := range e.Doc.Fields {
			ev.Fields[fl.Name] = encodeValue(fl.Value)
			ev.FieldLines[fl.Name] = fl.Line
		}
		for _, s := range e.Doc.Sections {
			sj := sectionJSON{Heading: s.Heading, Line: s.Line, Body: s.Body, Items: []itemJSON{}}
			for _, it := range s.Items {
				sj.Items = append(sj.Items, itemJSON{Text: it.Text, Line: it.Line})
			}
			ev.Sections = append(ev.Sections, sj)
		}
		f.Events = append(f.Events, ev)
	}
	for _, l := range in.Links {
		f.Links = append(f.Links, linkJSON{From: l.From, Section: l.Section, Item: l.Item, Line: l.Line, Target: l.Target, To: l.To, ToType: l.ToType, Broken: l.Broken})
	}
	return f
}

// encodeValue converts a frontmatter value to JSON: strings, numbers,
// booleans, null, and lists keep their YAML type; timestamps are strings.
func encodeValue(rv core.RawValue) any {
	switch rv.Kind {
	case core.ValueNull:
		return nil
	case core.ValueInt:
		if n, err := strconv.ParseInt(rv.Text, 0, 64); err == nil {
			return n
		}
		return rv.Text
	case core.ValueFloat:
		if n, err := strconv.ParseFloat(rv.Text, 64); err == nil {
			return n
		}
		return rv.Text
	case core.ValueBool:
		return rv.Text == "true"
	case core.ValueList:
		items := []any{}
		for _, it := range rv.Items {
			items = append(items, encodeValue(it))
		}
		return items
	case core.ValueMap:
		return map[string]any{}
	}
	return rv.Text
}

func encodeConfig(cfg *core.Config) configJSON {
	c := configJSON{Name: cfg.Name, Plugins: append([]string{}, cfg.Plugins...), Types: []typeJSON{}}
	for _, t := range cfg.Types {
		tj := typeJSON{
			templateJSON: encodeTemplate(t),
			Folder:       t.Folder, Plugin: t.Plugin, Base: t.Base, Validators: t.Validators, History: t.History, Frozen: t.Frozen,
		}
		c.Types = append(c.Types, tj)
	}
	return c
}

func encodeTemplate(t core.TypeDef) templateJSON {
	tj := templateJSON{Name: t.Name, EntryTiming: string(t.Timing), Sections: t.Sections}
	if t.MaxDelay > 0 {
		tj.MaxDelay = t.MaxDelay.String()
	}
	if len(t.Fields) > 0 {
		tj.Fields = map[string]fieldJSON{}
		for _, f := range t.Fields {
			tj.Fields[f.Name] = fieldJSON{Type: string(f.Kind), Required: f.Required, Values: f.Values, Items: string(f.Items)}
		}
	}
	if t.Status != nil {
		tj.Status = &statusJSON{Values: t.Status.Values, Closed: t.Status.Closed}
	}
	return tj
}

// decodeTemplate converts a plugin's type template to a core.TypeDef. Fields
// are ordered by name, since JSON objects have no order.
func decodeTemplate(tj templateJSON) (core.TypeDef, error) {
	t := core.TypeDef{Name: tj.Name, Timing: core.TimingRule(tj.EntryTiming), Sections: tj.Sections}
	if tj.MaxDelay != "" {
		d, err := time.ParseDuration(tj.MaxDelay)
		if err != nil || d <= 0 {
			return t, fmt.Errorf("type %q: max_delay %q is not a positive duration", tj.Name, tj.MaxDelay)
		}
		t.MaxDelay = d
	}
	if t.Timing == core.RealTime && t.MaxDelay == 0 {
		return t, fmt.Errorf("type %q: a real_time type needs max_delay", tj.Name)
	}
	names := make([]string, 0, len(tj.Fields))
	for n := range tj.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fj := tj.Fields[n]
		fd := core.FieldDef{Name: n, Kind: core.FieldKind(fj.Type), Required: fj.Required, Values: fj.Values, Items: core.FieldKind(fj.Items)}
		switch {
		case core.IsSharedField(n) || n == core.FieldStatus:
			return t, fmt.Errorf("type %q: field %q is a shared field", tj.Name, n)
		case !fd.Kind.Valid():
			return t, fmt.Errorf("type %q: field %q has unknown type %q", tj.Name, n, fj.Type)
		case fd.Kind == core.KindList && (fd.Items == "" || fd.Items == core.KindList || !fd.Items.Valid()):
			return t, fmt.Errorf("type %q: list field %q needs items of type string, number, bool, or time", tj.Name, n)
		}
		t.Fields = append(t.Fields, fd)
	}
	if tj.Status != nil {
		if len(tj.Status.Values) == 0 {
			return t, fmt.Errorf("type %q: status needs values", tj.Name)
		}
		t.Status = &core.StatusDef{Values: tj.Status.Values, Closed: tj.Status.Closed}
	}
	return t, nil
}
