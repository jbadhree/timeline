package core

import (
	"fmt"
	"sort"
	"time"
)

// Plugin adds types, validation rules, or commands on top of the core. A
// plugin implements Plugin and any of TypeProvider, Validator, and
// CommandProvider. Plugins only read the facts the core gives them and only
// add diagnostics: they never write files and cannot loosen a core rule.
//
// Owning goal: goals/plugins.md.
type Plugin interface {
	// Name identifies the plugin in configuration and diagnostics. It uses
	// lowercase letters, digits, and underscores.
	Name() string
}

// TypeProvider is a plugin that defines event types, such as goal and task.
// The definitions are templates: a root chooses each type's folder and may
// add fields and sections.
type TypeProvider interface {
	Plugin
	Types() []TypeDef
}

// Validator is a plugin with rules beyond the core ones. Validate receives
// the events of the plugin's types and must return diagnostics only for them.
// An error means the plugin itself failed, not that an event is invalid.
type Validator interface {
	Plugin
	Validate(in PluginInput) ([]Diagnostic, error)
}

// CommandProvider is a plugin that adds tline subcommands, such as
// "goal tree". Routing commands to plugins is done by the CLI.
type CommandProvider interface {
	Plugin
	Commands() []Command
}

// Command is a subcommand a plugin adds.
type Command struct {
	Name    string // as typed after "tline", such as "goal"
	Summary string // one line for tline --help
}

// PluginInput is what a validator sees: read-only facts about the events of
// its types. Plugins must not modify it.
type PluginInput struct {
	Config *Config
	Events []PluginEvent
	// Links are the links in those events' sections, resolved to their
	// targets, in document order.
	Links []ResolvedLink
	// Now is the time the check runs, the entry time a draft would get.
	Now time.Time
}

// PluginEvent is one event as plugins see it.
type PluginEvent struct {
	Path  string
	Type  *TypeDef
	ID    string // empty for a draft
	Draft bool
	Doc   *Document
}

// ResolvedLink is a fact about one link: where it is and what it points to.
// It carries no meaning; plugins decide what a link means.
type ResolvedLink struct {
	From    string // path of the event containing the link
	Section string // heading of the section containing the link
	Item    int    // index of the list item containing the link, or -1
	Line    int
	Target  string // as written
	// To is the path of the event the link points to, and ToType its type.
	// Both are empty for URLs, anchors, files outside the configured
	// folders, and links that do not resolve.
	To     string
	ToType string
	// Broken is true when the link points into a configured folder but no
	// event file is there. The core has already reported it.
	Broken bool
}

// pluginTypes returns the names of the types a plugin provides.
func pluginTypes(p Plugin) map[string]bool {
	names := map[string]bool{}
	if tp, ok := p.(TypeProvider); ok {
		for _, t := range tp.Types() {
			names[t.Name] = true
		}
	}
	return names
}

// runPlugins runs each validator over the events of its types and adds its
// diagnostics. A diagnostic for a file outside the plugin's types is replaced
// by one that says so, so a plugin cannot judge another plugin's events.
func (v *validator) runPlugins() error {
	seen := map[string]bool{}
	for _, p := range v.in.Plugins {
		if seen[p.Name()] {
			return fmt.Errorf("plugin %q is registered twice", p.Name())
		}
		seen[p.Name()] = true
		val, ok := p.(Validator)
		if !ok {
			continue
		}
		in, owned := v.pluginInput(pluginTypes(p))
		diags, err := val.Validate(in)
		if err != nil {
			return fmt.Errorf("plugin %s: %w", p.Name(), err)
		}
		for _, d := range diags {
			if !owned[d.Path] {
				v.add(Diagnostic{Path: d.Path, Line: d.Line, Message: fmt.Sprintf("plugin %s reported a problem in a file that is not one of its types: %s", p.Name(), d.Message)})
				continue
			}
			v.add(d)
		}
	}
	return nil
}

// pluginInput collects the events of the given types and their links.
func (v *validator) pluginInput(types map[string]bool) (PluginInput, map[string]bool) {
	in := PluginInput{Config: v.in.Config, Now: v.in.Now, Events: []PluginEvent{}, Links: []ResolvedLink{}}
	owned := map[string]bool{}
	for _, e := range v.events {
		if !types[e.typ.Name] {
			continue
		}
		owned[e.doc.Path] = true
		in.Events = append(in.Events, PluginEvent{Path: e.doc.Path, Type: e.typ, ID: e.id, Draft: e.draft, Doc: e.doc})
		in.Links = append(in.Links, v.linksOf(e)...)
	}
	return in, owned
}

// linksOf resolves every link in an event's sections.
func (v *validator) linksOf(e *event) []ResolvedLink {
	var out []ResolvedLink
	for _, s := range e.doc.Sections {
		for _, l := range s.Links {
			rl := ResolvedLink{From: e.doc.Path, Section: s.Heading, Item: itemIndex(s, l), Line: l.Line, Target: l.Target}
			switch t, r := v.resolveLink(e, l); r {
			case linkEvent:
				rl.To, rl.ToType = t.doc.Path, t.typ.Name
			case linkMissing:
				rl.Broken = true
			}
			out = append(out, rl)
		}
	}
	return out
}

// itemIndex returns the index of the list item containing link l, or -1 if
// the link is in prose. The candidate is the last item starting at or before
// the link's line, confirmed by the item having a link with the same target.
func itemIndex(s Section, l Link) int {
	i := sort.Search(len(s.Items), func(i int) bool { return s.Items[i].Line > l.Line }) - 1
	if i < 0 {
		return -1
	}
	for _, il := range s.Items[i].Links {
		if il.Target == l.Target {
			return i
		}
	}
	return -1
}
