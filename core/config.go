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

// TypeDef is one event type as configured in timeline.yaml.
type TypeDef struct {
	Name     string
	Folder   string // slash-separated, relative to the root, cleaned
	Plugin   string // name of the plugin that provides this type; empty if configured
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
)

// SharedFields lists the frontmatter fields every event may carry. Types
// cannot redefine them.
var SharedFields = []string{FieldID, FieldType, FieldTimelineAt, FieldEnteredAt, FieldEntryTiming, FieldCreatedBy}

// IsSharedField reports whether name is a shared field.
func IsSharedField(name string) bool { return contains(SharedFields, name) }

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
