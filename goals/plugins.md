---
id: 01M4SY1Q58X975M8Q178X18PSD
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: goal
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
goal: Let plugins define types, validation rules, and commands, and ship goals and tasks as the first one
owner: Badhree Babu
status: achieved
---

## Why

Every domain gives its events and links its own meaning: goals have children, decisions supersede each other, incidents have follow-ups. Timeline should hold the facts that are true everywhere and let plugins add meaning, so new kinds of events need no change to Timeline. Decided in [link meanings belong to plugins](../decisions/relations-in-plugins.md) and [goal and task ship as a built-in plugin](../decisions/goals-and-tasks-as-plugin.md).

## How

- [Define the plugin interface in the core](../tasks/plugin-interface.md)
- [Move goal and task rules into a built-in goals plugin](../tasks/goals-plugin.md)
- [Enable plugins and their types in timeline.yaml](../tasks/plugin-config.md)
- [Let plugins register commands](../tasks/plugin-commands.md)

## Done when

- `core/` contains no goal- or task-specific code: no children, cycles, inherited why, or linked-step rules.
- Every existing goal and task test passes against the goals plugin, and this repository's goals and tasks check exactly as before.
- `tline goal` and `tline task` exist only because the goals plugin registers them.

## Why it will work

The goal and task rules already sit in a few functions over parsed documents and resolved links, so moving them behind an interface changes where they live, not what they do. Making Timeline's own goals the first plugin proves the interface on real rules before anyone else depends on it, and doing it before the goal views are written means those views are built once, in their final home.

## Design

### Facts in the core, meaning in plugins

| The core provides, for every domain | Plugins decide, per domain |
| --- | --- |
| Event files, fields, statuses, sections, list items | Which links count as relations, such as child, supersedes, or blocks |
| Two times, timing rules, stamps, IDs, past due | Whether a relation must be a tree, acyclic, or anything goes |
| Resolved links: which section and item links to which event and type | Rules across events, such as inherited why or every goal step linking to a goal or task |
| Diagnostics, sorting, JSON output | Commands and views, such as the goal tree |

The core still checks that links into configured folders resolve, because that is a fact. Plugins can only add diagnostics: they never write files and never loosen a core rule, so the two-times guarantees hold whatever is installed.

### A plugin

A plugin has a name and provides any of:

- **types:** definitions with fields, statuses, required sections, and timing rule, like a `timeline.yaml` type;
- **validate:** given the events of its types and their resolved links, return diagnostics;
- **commands:** named subcommands, such as `goal tree`, given the same events and returning terminal or JSON output.

In Go, these are small interfaces in `core/plugin.go`: every plugin implements `Plugin` (a name) and any of `TypeProvider`, `Validator`, and `CommandProvider`. A validator receives a `PluginInput`: the config, the events of the plugin's own types, every link in those events as a `ResolvedLink` (section, list item, line, target as written, and the event and type it resolves to), and the current time. It returns diagnostics, or an error if the plugin itself failed, which stops the check with exit 1. Plugin validation runs after the core rules, its diagnostics are merged, sorted, and limited to the selected files like any other, and a diagnostic for a file outside the plugin's types is reported as the plugin's mistake instead.

A command (`core.Command`) has a name, a summary for `tline --help`, usage text, and a `Run` function. `tline <name> ...` runs it after the core commands: the CLI checks the root, then passes the command the same facts a validator gets, its arguments, whether `--json` was asked for (before or after the command), and the diagnostics in the plugin's own events. The command returns text, a JSON value, and whether its result is incomplete (exit 5). A `core.UsageError` exits 2 and any other error exits 1. A plugin command that clashes with a core command or another plugin's command is an error, and `tline --help` lists plugin commands with the plugin that provides each.

Built-in plugins are Go packages compiled into `tline` that implement the core interface; the goals plugin in `internal/plugins/goals` is the first. A type in `timeline.yaml` named after a plugin's type, such as `goal: { folder: goals }`, is built from that plugin's template: the root picks its folder and may add fields and sections. Two plugins providing the same type name, or a template without a valid name and timing rule, is a configuration error. External plugins are executables that speak a versioned JSON protocol: `tline` starts each once per command, sends the events and links on stdin with the current time, and reads diagnostics or output from stdout, with a timeout. They run only when the root's own `timeline.yaml` names them.

### Configuration

```yaml
plugins: [goals]                    # the plugins this timeline uses
types:
  goal: { from: goals/goal, folder: goals }
  objective: { from: goals/goal, folder: objectives }   # a second, renamed goal type
  task: { from: goals/task, folder: tasks }
  prediction:
    folder: predictions
    entry_timing: future
    validators: [ticker]            # another plugin's rules on a configured type
```

- `plugins` lists the plugins a timeline uses; only those run, and a listed plugin that `tline` does not have is an error. Without the list, every built-in plugin is used, so older roots keep working. A plugin that is not listed leaves its type names free for ordinary configured types, and its commands refuse to run in that timeline.
- `from: plugin/type` builds a type from a plugin's template. The type may have any name, and several types may share one template; the plugin's rules follow the template (`TypeDef.Base`), not the name. Without `from`, a type named like a type of a used plugin is built from it, as in `goal: { folder: goals }`; if two used plugins provide that name, `from` must choose.
- `validators` adds the rules of other used plugins to a type; those plugins then receive its events.
- External plugins, written as `- name: { run: ./path }`, are rejected: whether and when to support them is an [open decision](../decisions/external-plugins.md).

A plugin depending on another plugin's types reads only that plugin's facts, never its meaning. Versioned plugin names and the protocol version keep this stable. WebAssembly plugins are a possible later option if untrusted plugins ever matter.
