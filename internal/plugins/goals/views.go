package goals

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jbadhree/timeline/core"
)

const goalUsage = `Usage:
  tline goal list [--owner NAME] [--status STATUS] [--past-due]
  tline goal show <goal>
  tline goal tree [goal]

list   goals by target date, with status, owner, and past-due flags
show   one goal as its seven answers, with the status of every step
tree   goals and tasks beneath a goal, or beneath every top-level goal

A goal is named by its ID, its path, or its file name, such as "plugins".
Add --json for machine-readable output.
`

const taskUsage = `Usage:
  tline task list [--owner NAME] [--status STATUS] [--past-due]
  tline task show <task>

list   tasks by due date, with status, owner, parent goals, and past-due flags
show   one task as its five answers, with its why inherited from parent goals

A task is named by its ID, its path, or its file name, such as "stamp-drafts".
Add --json for machine-readable output.
`

// Commands implements core.CommandProvider.
func (Plugin) Commands() []core.Command {
	return []core.Command{
		{Name: "goal", Summary: "list, show, and tree views of goals", Usage: goalUsage, Run: runGoal},
		{Name: "task", Summary: "list and show views of tasks", Usage: taskUsage, Run: runTask},
	}
}

func runGoal(_ context.Context, in core.CommandInput) (core.CommandOutput, error) {
	m := newModel(in.PluginInput)
	sub, args := subcommand(in.Args)
	switch sub {
	case "list":
		return listView(m, in, Goal, args)
	case "show":
		n, err := one(m, Goal, args)
		if err != nil {
			return core.CommandOutput{}, err
		}
		return withCompleteness(showGoal(m, n), in), nil
	case "tree":
		if len(args) > 1 {
			return core.CommandOutput{}, core.UsageError{Msg: "tree takes at most one goal"}
		}
		var roots []*node
		if len(args) == 1 {
			n, err := one(m, Goal, args)
			if err != nil {
				return core.CommandOutput{}, err
			}
			roots = []*node{n}
		} else {
			for _, n := range byDue(m.order) {
				if n.ev.Type.Name == Goal && len(n.parents) == 0 {
					roots = append(roots, n)
				}
			}
		}
		return withCompleteness(treeView(roots), in), nil
	}
	return core.CommandOutput{}, core.UsageError{Msg: "expected list, show, or tree; run 'tline goal --help'"}
}

func runTask(_ context.Context, in core.CommandInput) (core.CommandOutput, error) {
	m := newModel(in.PluginInput)
	sub, args := subcommand(in.Args)
	switch sub {
	case "list":
		return listView(m, in, Task, args)
	case "show":
		n, err := one(m, Task, args)
		if err != nil {
			return core.CommandOutput{}, err
		}
		return withCompleteness(showTask(n), in), nil
	}
	return core.CommandOutput{}, core.UsageError{Msg: "expected list or show; run 'tline task --help'"}
}

func subcommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

// withCompleteness marks output incomplete when any goal or task is invalid:
// a view built from invalid events cannot be trusted to be whole.
func withCompleteness(out core.CommandOutput, in core.CommandInput) core.CommandOutput {
	out.Incomplete = len(in.Diagnostics) > 0
	return out
}

// one resolves a single goal or task argument.
func one(m *model, typ string, args []string) (*node, error) {
	if len(args) != 1 {
		return nil, core.UsageError{Msg: fmt.Sprintf("name one %s by ID, path, or file name", typ)}
	}
	found := m.find(args[0], typ)
	switch len(found) {
	case 0:
		return nil, core.UsageError{Msg: fmt.Sprintf("no %s matches %q", typ, args[0])}
	case 1:
		return found[0], nil
	}
	var paths []string
	for _, n := range found {
		paths = append(paths, n.ev.Path)
	}
	return nil, core.UsageError{Msg: fmt.Sprintf("%q matches several %ss: %s", args[0], typ, strings.Join(paths, ", "))}
}

func byDue(nodes []*node) []*node {
	out := append([]*node{}, nodes...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].ev.Time, out[j].ev.Time
		if !a.Equal(b) {
			return a.Before(b)
		}
		return out[i].ev.Path < out[j].ev.Path
	})
	return out
}

func date(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	return t.UTC().Format("2006-01-02")
}

// flags describes the past-due and draft state of a node for terminal output.
func flags(n *node) string {
	var f []string
	if n.ev.PastDue {
		f = append(f, "PAST DUE")
	}
	if n.ev.Draft {
		f = append(f, "draft")
	}
	return strings.Join(f, ", ")
}

// item is the JSON form of a goal or task in lists and trees.
type item struct {
	Path     string   `json:"path"`
	ID       string   `json:"id,omitempty"`
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Owner    string   `json:"owner"`
	Due      string   `json:"due"`
	PastDue  bool     `json:"past_due"`
	Draft    bool     `json:"draft"`
	Parents  []string `json:"parents,omitempty"`
	Children []item   `json:"children,omitempty"`
}

func toItem(n *node) item {
	it := item{
		Path: n.ev.Path, ID: n.ev.ID, Type: n.ev.Type.Name, Title: n.title(),
		Status: n.field(core.FieldStatus), Owner: n.field("owner"),
		PastDue: n.ev.PastDue, Draft: n.ev.Draft,
	}
	if !n.ev.Time.IsZero() {
		it.Due = n.ev.Time.UTC().Format(time.RFC3339)
	}
	for _, p := range n.parents {
		it.Parents = append(it.Parents, p.ev.Path)
	}
	return it
}

func listView(m *model, in core.CommandInput, typ string, args []string) (core.CommandOutput, error) {
	fs := flag.NewFlagSet(typ+" list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	owner := fs.String("owner", "", "")
	status := fs.String("status", "", "")
	pastDue := fs.Bool("past-due", false, "")
	if err := fs.Parse(args); err != nil {
		return core.CommandOutput{}, core.UsageError{Msg: err.Error()}
	}
	if fs.NArg() > 0 {
		return core.CommandOutput{}, core.UsageError{Msg: fmt.Sprintf("unexpected argument %q", fs.Arg(0))}
	}

	var rows []*node
	for _, n := range byDue(m.order) {
		if n.ev.Type.Name != typ ||
			(*owner != "" && n.field("owner") != *owner) ||
			(*status != "" && n.field(core.FieldStatus) != *status) ||
			(*pastDue && !n.ev.PastDue) {
			continue
		}
		rows = append(rows, n)
	}

	items := []item{}
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	header := "DUE\tSTATUS\tFLAGS\tOWNER\t" + strings.ToUpper(typ) + "\tPATH"
	if typ == Task {
		header = "DUE\tSTATUS\tFLAGS\tOWNER\tTASK\tGOAL\tPATH"
	}
	// The tabwriter writes to a strings.Builder, which cannot fail.
	_, _ = fmt.Fprintln(w, header)
	for _, n := range rows {
		items = append(items, toItem(n))
		cols := []string{date(n.ev.Time), n.field(core.FieldStatus), flags(n), n.field("owner"), n.title()}
		if typ == Task {
			var parents []string
			for _, p := range n.parents {
				parents = append(parents, p.ev.Path)
			}
			cols = append(cols, strings.Join(parents, ", "))
		}
		cols = append(cols, n.ev.Path)
		_, _ = fmt.Fprintln(w, strings.Join(cols, "\t"))
	}
	_ = w.Flush()
	text := b.String()
	if len(rows) == 0 {
		text = fmt.Sprintf("No %ss match.\n", typ)
	}
	return withCompleteness(core.CommandOutput{Text: text, JSON: map[string]any{typ + "s": items}}, in), nil
}

// step is one entry of a goal's How, as shown by goal show.
type step struct {
	Text    string `json:"text"`
	Targets []item `json:"targets"`
}

func showGoal(m *model, n *node) core.CommandOutput {
	var b strings.Builder
	header(&b, n)
	why(&b, n)

	var steps []step
	b.WriteString("\nHow\n")
	if how, ok := n.ev.Doc.Section(SectionHow); ok {
		byItem := map[int][]core.ResolvedLink{}
		for _, l := range m.stepLinks(n) {
			byItem[l.Item] = append(byItem[l.Item], l)
		}
		for i, it := range how.Items {
			st := step{Text: it.Text, Targets: []item{}}
			for _, l := range byItem[i] {
				if t, ok := m.nodes[l.To]; ok && isGoalOrTask(l.ToType) {
					st.Targets = append(st.Targets, toItem(t))
					fmt.Fprintf(&b, "  - [%s] %s: %s  (%s)%s\n", t.field(core.FieldStatus), t.ev.Type.Name, t.title(), date(t.ev.Time), suffix(flags(t)))
				}
			}
			if len(st.Targets) == 0 {
				fmt.Fprintf(&b, "  - %s  (not linked to a goal or task)\n", firstLine(it.Text))
			}
			steps = append(steps, st)
		}
	}
	done, _ := n.section(SectionDoneWhen)
	works, _ := n.section(SectionWhyItWillWork)
	b.WriteString("\nDone when\n" + indent(done) + "\nWhy it will work\n" + indent(works))

	whyText, inherited := whyAnswer(n)
	return core.CommandOutput{Text: b.String(), JSON: map[string]any{
		"goal": toItem(n), "why": whyText, "why_inherited_from": inherited,
		"how": steps, "done_when": done, "why_it_will_work": works,
	}}
}

func showTask(n *node) core.CommandOutput {
	var b strings.Builder
	header(&b, n)
	why(&b, n)
	how, _ := n.section(SectionHow)
	b.WriteString("\nHow\n" + indent(how))
	whyText, inherited := whyAnswer(n)
	return core.CommandOutput{Text: b.String(), JSON: map[string]any{
		"task": toItem(n), "why": whyText, "why_inherited_from": inherited, "how": how,
	}}
}

// header writes what, by when, owner, status, and where the file is.
func header(b *strings.Builder, n *node) {
	label := strings.ToUpper(n.ev.Type.Name[:1]) + n.ev.Type.Name[1:]
	fmt.Fprintf(b, "%s: %s\n", label, n.title())
	fmt.Fprintf(b, "By:     %s%s\n", date(n.ev.Time), suffix(flags(n)))
	fmt.Fprintf(b, "Owner:  %s\n", n.field("owner"))
	fmt.Fprintf(b, "Status: %s\n", n.field(core.FieldStatus))
	if n.ev.ID != "" {
		fmt.Fprintf(b, "File:   %s (id %s)\n", n.ev.Path, n.ev.ID)
	} else {
		fmt.Fprintf(b, "File:   %s\n", n.ev.Path)
	}
}

func why(b *strings.Builder, n *node) {
	text, inherited := whyAnswer(n)
	b.WriteString("\nWhy\n")
	if len(inherited) > 0 {
		b.WriteString("  Inherited from:\n")
	}
	b.WriteString(indent(text))
}

// whyAnswer returns the node's own Why, or its parents' goal statements and
// their paths when it inherits its why.
func whyAnswer(n *node) (string, []string) {
	if text, ok := n.section(SectionWhy); ok {
		return text, nil
	}
	var lines, paths []string
	for _, p := range n.parents {
		lines = append(lines, "- "+p.title()+" ("+p.ev.Path+")")
		paths = append(paths, p.ev.Path)
	}
	return strings.Join(lines, "\n"), paths
}

func treeView(roots []*node) core.CommandOutput {
	var b strings.Builder
	var items []item
	var walk func(n *node, prefix string, last bool, onPath map[*node]bool, depth int) item
	walk = func(n *node, prefix string, last bool, onPath map[*node]bool, depth int) item {
		branch, next := "", ""
		if depth > 0 {
			branch, next = "├─ ", "│  "
			if last {
				branch, next = "└─ ", "   "
			}
		}
		fmt.Fprintf(&b, "%s%s[%s] %s  (%s)%s\n", prefix, branch, n.field(core.FieldStatus), n.title(), date(n.ev.Time), suffix(flags(n)))
		it := toItem(n)
		it.Parents = nil
		if onPath[n] {
			return it // a cycle; validation reports it
		}
		onPath[n] = true
		for i, c := range n.children {
			it.Children = append(it.Children, walk(c, prefix+next, i == len(n.children)-1, onPath, depth+1))
		}
		delete(onPath, n)
		return it
	}
	for i, r := range roots {
		if i > 0 {
			b.WriteString("\n")
		}
		items = append(items, walk(r, "", true, map[*node]bool{}, 0))
	}
	if len(roots) == 0 {
		b.WriteString("No goals.\n")
		items = []item{}
	}
	return core.CommandOutput{Text: b.String(), JSON: map[string]any{"goals": items}}
}

func suffix(s string) string {
	if s == "" {
		return ""
	}
	return "  " + s
}

func indent(text string) string {
	if strings.TrimSpace(text) == "" {
		return "  (none)\n"
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}
