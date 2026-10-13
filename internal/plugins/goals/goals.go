// Package goals is the built-in goals plugin. It defines the goal and task
// types and the rules that give their links meaning: every goal step links to
// a goal or task, linked goals and tasks are children, children never form a
// cycle, a child may inherit its why, and a task's steps never link to goals
// or tasks.
//
// It sees only the facts the core provides (core.PluginInput) and adds
// diagnostics; it never reads or writes files.
//
// Owning goals: goals/goals-and-tasks.md and goals/plugins.md.
package goals

import (
	"fmt"
	"strings"

	"github.com/jbadhree/timeline/core"
)

// Type names provided by this plugin.
const (
	Goal = "goal"
	Task = "task"
)

// Section headings the plugin reads.
const (
	SectionWhy           = "Why"
	SectionHow           = "How"
	SectionDoneWhen      = "Done when"
	SectionWhyItWillWork = "Why it will work"
)

// Plugin is the goals plugin. The zero value is ready to use.
type Plugin struct{}

// New returns the goals plugin.
func New() Plugin { return Plugin{} }

// pluginName is the name timelines use in their plugins list and in
// "from: goals/goal".
const pluginName = "goals"

// Name implements core.Plugin.
func (Plugin) Name() string { return pluginName }

// Types implements core.TypeProvider: the goal and task templates. A root
// chooses their folders and may add fields and sections.
func (Plugin) Types() []core.TypeDef {
	return []core.TypeDef{
		{
			Name:   Goal,
			Timing: core.Future,
			Fields: []core.FieldDef{
				{Name: Goal, Kind: core.KindString, Required: true, NonEmpty: true},
				{Name: "owner", Kind: core.KindString, Required: true, NonEmpty: true},
			},
			Status: &core.StatusDef{Values: []string{"open", "achieved", "missed", "dropped"}, Closed: []string{"achieved", "missed", "dropped"}},
		},
		{
			Name:   Task,
			Timing: core.Future,
			Fields: []core.FieldDef{
				{Name: Task, Kind: core.KindString, Required: true, NonEmpty: true},
				{Name: "owner", Kind: core.KindString, Required: true, NonEmpty: true},
			},
			Status: &core.StatusDef{Values: []string{"open", "done", "dropped"}, Closed: []string{"done", "dropped"}},
		},
	}
}

type checker struct {
	*model
	diags []core.Diagnostic
}

func (c *checker) add(d core.Diagnostic) { c.diags = append(c.diags, d) }

// Validate implements core.Validator.
func (Plugin) Validate(in core.PluginInput) ([]core.Diagnostic, error) {
	c := &checker{model: newModel(in)}
	for _, n := range c.order {
		c.checkSections(n)
		c.checkSteps(n)
	}
	c.checkCycles()
	c.checkWhy()
	return c.diags, nil
}

// checkSections requires the answers that live in sections.
func (c *checker) checkSections(n *node) {
	d, p := n.ev.Doc, n.ev.Path
	switch n.kind() {
	case Goal:
		c.requireItems(p, d, SectionHow, "must list at least one step")
		c.requireItems(p, d, SectionDoneWhen, "must list at least one criterion")
		c.requireBody(p, d, SectionWhyItWillWork)
	case Task:
		c.requireItems(p, d, SectionHow, "must list at least one step")
	}
}

func (c *checker) requireItems(p string, d *core.Document, heading, msg string) {
	s, ok := d.Section(heading)
	if !ok {
		c.add(core.Diagnostic{Path: p, Section: heading, Message: "is required"})
		return
	}
	if len(s.Items) == 0 {
		c.add(core.Diagnostic{Path: p, Line: s.Line, Section: heading, Message: msg + " as a Markdown list item"})
	}
}

func (c *checker) requireBody(p string, d *core.Document, heading string) {
	s, ok := d.Section(heading)
	if !ok {
		c.add(core.Diagnostic{Path: p, Section: heading, Message: "is required"})
		return
	}
	if strings.TrimSpace(s.Body) == "" {
		c.add(core.Diagnostic{Path: p, Line: s.Line, Section: heading, Message: "must not be empty"})
	}
}

// checkSteps applies the How rules: every goal step links to a goal or task;
// a task's steps never link to a goal or task. The model has already made
// the linked goals and tasks children.
func (c *checker) checkSteps(n *node) {
	how, ok := n.ev.Doc.Section(SectionHow)
	if !ok {
		return
	}
	byItem := map[int][]core.ResolvedLink{}
	for _, l := range c.stepLinks(n) {
		byItem[l.Item] = append(byItem[l.Item], l)
	}
	for i, item := range how.Items {
		linked, broken := false, false
		for _, l := range byItem[i] {
			switch {
			case l.Broken:
				broken = true
			case c.isGoalOrTask(l.ToType) && n.kind() == Task:
				c.add(core.Diagnostic{Path: n.ev.Path, Line: l.Line, Section: SectionHow, Message: fmt.Sprintf("a task's steps must not link to a %s (%s); make this task a goal instead", l.ToType, l.To)})
			case c.isGoalOrTask(l.ToType):
				linked = true
			}
		}
		// A broken link is already reported by the core; do not also call
		// the step unlinked.
		if n.kind() == Goal && !linked && !broken {
			c.add(core.Diagnostic{Path: n.ev.Path, Line: item.Line, Section: SectionHow, Message: fmt.Sprintf("step %q must link to a goal or task", firstLine(item.Text))})
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 60 {
		s = s[:57] + "..."
	}
	return s
}

// checkCycles reports goals whose child links lead back to themselves.
func (c *checker) checkCycles() {
	const (
		unvisited = iota
		active
		done
	)
	state := map[*node]int{}
	var stack []*node
	reported := map[string]bool{}
	var visit func(n *node)
	visit = func(n *node) {
		state[n] = active
		stack = append(stack, n)
		for _, ch := range n.children {
			switch state[ch] {
			case unvisited:
				visit(ch)
			case active:
				var cycle []string
				for i := len(stack) - 1; i >= 0; i-- {
					cycle = append([]string{stack[i].ev.Path}, cycle...)
					if stack[i] == ch {
						break
					}
				}
				key := strings.Join(cycle, " -> ")
				if !reported[key] {
					reported[key] = true
					for _, p := range cycle {
						c.add(core.Diagnostic{Path: p, Section: SectionHow, Message: "goal steps form a cycle: " + key + " -> " + ch.ev.Path})
					}
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = done
	}
	for _, n := range c.order {
		if n.kind() == Goal && state[n] == unvisited {
			visit(n)
		}
	}
}

// checkWhy requires a Why section for goals and tasks that no goal links to.
func (c *checker) checkWhy() {
	for _, n := range c.order {
		s, ok := n.ev.Doc.Section(SectionWhy)
		switch {
		case ok && strings.TrimSpace(s.Body) == "":
			c.add(core.Diagnostic{Path: n.ev.Path, Line: s.Line, Section: SectionWhy, Message: "must not be empty"})
		case !ok && len(n.parents) == 0:
			c.add(core.Diagnostic{Path: n.ev.Path, Section: SectionWhy, Message: fmt.Sprintf("is required because no goal links to this %s from its How section", n.ev.Type.Name)})
		}
	}
}
