package core

// ValueKind is the YAML kind of a raw frontmatter value, before it is checked
// against a declared field kind.
type ValueKind int

// Raw value kinds.
const (
	ValueNull ValueKind = iota
	ValueString
	ValueInt
	ValueFloat
	ValueBool
	ValueTimestamp
	ValueList
	ValueMap
)

// RawValue is one decoded frontmatter value with its source line.
type RawValue struct {
	Kind  ValueKind
	Text  string // scalar text exactly as written, without quotes
	Items []RawValue
	Line  int
}

// Field is one frontmatter key and its value.
type Field struct {
	Name  string
	Value RawValue
	Line  int
}

// Link is a Markdown link found in a list item.
type Link struct {
	Text   string
	Target string // as written, possibly with an #anchor
	Line   int
}

// ListItem is one top-level list item in a section, including continuation
// lines.
type ListItem struct {
	Text  string
	Line  int
	Links []Link
}

// Section is the body text under one level-2 heading.
type Section struct {
	Heading string
	Line    int
	Body    string // text between this heading and the next level-1 or level-2 heading
	Items   []ListItem
	Links   []Link // every link in the section outside fenced code, in order
}

// Document is one decoded event file.
type Document struct {
	Path     string // slash-separated, relative to the root
	Fields   []Field
	Sections []Section
	// ParseErr is set when the file could not be decoded; Fields and Sections
	// are then incomplete.
	ParseErr *Diagnostic
}

// Section returns the first section with the exact heading.
func (d *Document) Section(heading string) (*Section, bool) {
	for i := range d.Sections {
		if d.Sections[i].Heading == heading {
			return &d.Sections[i], true
		}
	}
	return nil, false
}

// Field returns the named frontmatter field.
func (d *Document) Field(name string) (*Field, bool) {
	for i := range d.Fields {
		if d.Fields[i].Name == name {
			return &d.Fields[i], true
		}
	}
	return nil, false
}
