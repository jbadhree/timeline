package goals

import (
	"sort"
	"strings"

	"github.com/jbadhree/timeline/core"
)

// node is one goal or task and its place in the tree.
type node struct {
	ev       core.PluginEvent
	children []*node // in How order; a goal's linked goals and tasks
	parents  []*node // goals that link to this node from their How
}

// model is the tree of goals and tasks built from the core's facts. The
// validator and the views share it, so both agree on what a child is.
type model struct {
	nodes map[string]*node // by path
	order []*node          // by path
	links map[string][]core.ResolvedLink
	// kinds maps each configured type name to the plugin type it was built
	// from ("goal" or "task"), so renamed types keep their rules.
	kinds map[string]string
}

// newModel builds the tree. A link in a goal's How list item to a goal or
// task makes the target a child. Events that failed to parse are left out.
func newModel(in core.PluginInput) *model {
	m := &model{nodes: map[string]*node{}, links: map[string][]core.ResolvedLink{}, kinds: map[string]string{}}
	for _, t := range in.Config.Types {
		if t.Plugin == pluginName {
			m.kinds[t.Name] = t.Base
		}
	}
	for _, ev := range in.Events {
		if ev.Doc.ParseErr != nil {
			continue
		}
		n := &node{ev: ev}
		m.nodes[ev.Path] = n
		m.order = append(m.order, n)
	}
	sort.Slice(m.order, func(i, j int) bool { return m.order[i].ev.Path < m.order[j].ev.Path })
	for _, l := range in.Links {
		m.links[l.From] = append(m.links[l.From], l)
	}
	for _, n := range m.order {
		if n.kind() != Goal {
			continue
		}
		for _, l := range m.stepLinks(n) {
			if !m.isGoalOrTask(l.ToType) {
				continue
			}
			if child, ok := m.nodes[l.To]; ok {
				n.children = append(n.children, child)
				child.parents = append(child.parents, n)
			}
		}
	}
	return m
}

// stepLinks returns the links in list items of n's How section, in order.
func (m *model) stepLinks(n *node) []core.ResolvedLink {
	var out []core.ResolvedLink
	for _, l := range m.links[n.ev.Path] {
		if l.Section == SectionHow && l.Item >= 0 {
			out = append(out, l)
		}
	}
	return out
}

// kind returns "goal" or "task": the plugin type the node's type was built
// from, whatever the timeline calls it.
func (n *node) kind() string {
	if n.ev.Type.Base != "" {
		return n.ev.Type.Base
	}
	return n.ev.Type.Name
}

// isGoalOrTask reports whether a configured type name is a goal or task type.
func (m *model) isGoalOrTask(typeName string) bool {
	k := m.kinds[typeName]
	return k == Goal || k == Task
}

// title returns the goal or task statement, from the field named after the
// plugin type ("goal" or "task").
func (n *node) title() string {
	if f, ok := n.ev.Doc.Field(n.kind()); ok && f.Value.Kind == core.ValueString {
		return f.Value.Text
	}
	return "(no " + n.kind() + " statement)"
}

// field returns a string field's value, or "".
func (n *node) field(name string) string {
	if f, ok := n.ev.Doc.Field(name); ok && f.Value.Kind == core.ValueString {
		return f.Value.Text
	}
	return ""
}

// section returns a section's body, trimmed, and whether it exists.
func (n *node) section(heading string) (string, bool) {
	s, ok := n.ev.Doc.Section(heading)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(s.Body), true
}

// find resolves a command argument to a goal or task of type typ (or any,
// if typ is empty): an ID, a root-relative path, or a file name with or
// without ".md". It returns every match.
func (m *model) find(arg, typ string) []*node {
	var out []*node
	for _, n := range m.order {
		if typ != "" && n.kind() != typ {
			continue
		}
		base := n.ev.Path[strings.LastIndexByte(n.ev.Path, '/')+1:]
		if arg == n.ev.ID || arg == n.ev.Path || arg == base || arg+".md" == base {
			out = append(out, n)
		}
	}
	return out
}
