package core

import (
	"path"
	"strings"
	"time"
)

// TimingRule is a type's entry timing rule.
type TimingRule string

// Entry timing rules.
const (
	Past     TimingRule = "past"
	Future   TimingRule = "future"
	RealTime TimingRule = "real_time"
)

// Valid reports whether r is one of the defined rules.
func (r TimingRule) Valid() bool {
	return r == Past || r == Future || r == RealTime
}

// FieldKind is the declared type of a frontmatter field.
type FieldKind string

// Field kinds.
const (
	KindString FieldKind = "string"
	KindNumber FieldKind = "number"
	KindBool   FieldKind = "bool"
	KindTime   FieldKind = "time"
	KindList   FieldKind = "list"
)

// Valid reports whether k is a defined field kind.
func (k FieldKind) Valid() bool {
	switch k {
	case KindString, KindNumber, KindBool, KindTime, KindList:
		return true
	}
	return false
}

// FieldDef declares one type-specific frontmatter field.
type FieldDef struct {
	Name     string
	Kind     FieldKind
	Items    FieldKind // element kind when Kind is KindList
	Required bool
	Values   []string // allowed values for a scalar or list items; empty means any
	NonEmpty bool     // string must not be blank (built-in answers)
}

// StatusDef lists a type's allowed status values and which are closed.
type StatusDef struct {
	Values []string
	Closed []string
}

// IsClosed reports whether s is a closed status.
func (s *StatusDef) IsClosed(v string) bool {
	return contains(s.Closed, v)
}

// Builtin identifies a built-in type.
type Builtin string

// Built-in types. NotBuiltin marks a configured type.
const (
	NotBuiltin Builtin = ""
	GoalType   Builtin = "goal"
	TaskType   Builtin = "task"
)

// TypeDef is one event type as configured in timeline.yaml.
type TypeDef struct {
	Name     string
	Folder   string // slash-separated, relative to the root, cleaned
	Builtin  Builtin
	Timing   TimingRule
	MaxDelay time.Duration // only for RealTime
	Fields   []FieldDef
	Sections []string // required ## headings
	Status   *StatusDef
}

// Field returns the named type-specific field definition.
func (t *TypeDef) Field(name string) (FieldDef, bool) {
	for _, f := range t.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldDef{}, false
}

// Config is a decoded, validated timeline.yaml.
type Config struct {
	Version int
	Name    string
	Types   []TypeDef
}

// TypeByName returns the named type.
func (c *Config) TypeByName(name string) (*TypeDef, bool) {
	for i := range c.Types {
		if c.Types[i].Name == name {
			return &c.Types[i], true
		}
	}
	return nil, false
}

// TypeForPath returns the type whose folder contains the root-relative,
// slash-separated path p.
func (c *Config) TypeForPath(p string) (*TypeDef, bool) {
	p = path.Clean(p)
	for i := range c.Types {
		if strings.HasPrefix(p, c.Types[i].Folder+"/") {
			return &c.Types[i], true
		}
	}
	return nil, false
}

// Shared frontmatter fields present on every event.
const (
	FieldID          = "id"
	FieldType        = "type"
	FieldTimelineAt  = "timeline_at"
	FieldEnteredAt   = "entered_at"
	FieldEntryTiming = "entry_timing"
	FieldCreatedBy   = "created_by"
	FieldStatus      = "status"
	FieldOwner       = "owner"
)

// SharedFields lists the frontmatter fields every event may carry. Types
// cannot redefine them.
var SharedFields = []string{FieldID, FieldType, FieldTimelineAt, FieldEnteredAt, FieldEntryTiming, FieldCreatedBy}

// IsSharedField reports whether name is a shared field.
func IsSharedField(name string) bool { return contains(SharedFields, name) }

// Built-in section headings.
const (
	SectionWhy           = "Why"
	SectionHow           = "How"
	SectionDoneWhen      = "Done when"
	SectionWhyItWillWork = "Why it will work"
)

// NewBuiltinType returns the definition of a built-in type stored in folder,
// with a timeline's own extra fields and sections appended. It does not check
// extras for conflicts; configuration decoding does that with
// BuiltinFieldNames.
func NewBuiltinType(b Builtin, folder string, extraFields []FieldDef, extraSections []string) TypeDef {
	t := TypeDef{Name: string(b), Folder: folder, Builtin: b, Timing: Future}
	switch b {
	case GoalType:
		t.Fields = []FieldDef{
			{Name: "goal", Kind: KindString, Required: true, NonEmpty: true},
			{Name: FieldOwner, Kind: KindString, Required: true, NonEmpty: true},
		}
		t.Status = &StatusDef{Values: []string{"open", "achieved", "missed", "dropped"}, Closed: []string{"achieved", "missed", "dropped"}}
	case TaskType:
		t.Fields = []FieldDef{
			{Name: "task", Kind: KindString, Required: true, NonEmpty: true},
			{Name: FieldOwner, Kind: KindString, Required: true, NonEmpty: true},
		}
		t.Status = &StatusDef{Values: []string{"open", "done", "dropped"}, Closed: []string{"done", "dropped"}}
	}
	t.Fields = append(t.Fields, extraFields...)
	t.Sections = append(t.Sections, extraSections...)
	return t
}

// BuiltinFieldNames returns the frontmatter fields a built-in type defines
// itself, which a configuration may not redefine.
func BuiltinFieldNames(b Builtin) []string {
	return []string{string(b), FieldOwner, FieldStatus}
}

// IsBuiltinName reports whether name is reserved for a built-in type.
func IsBuiltinName(name string) bool {
	return name == string(GoalType) || name == string(TaskType)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
