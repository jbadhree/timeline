---
id: 01M4SY1Q58X975M8Q178X18PSD
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: goal
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
goal: Let plugins define types, validation rules, and commands, and ship goals and tasks as the first one
owner: Badhree Babu
status: open
---

## Why

Every domain gives its events and links its own meaning: goals have children, decisions supersede each other, incidents have follow-ups. Timeline should hold the facts that are true everywhere and let plugins add meaning, so new kinds of events need no change to Timeline. Decided in [link meanings belong to plugins](../decisions/relations-in-plugins.md) and [goal and task ship as a built-in plugin](../decisions/goals-and-tasks-as-plugin.md).

## How

- [Define the plugin interface in the core](../tasks/plugin-interface.md)
- [Move goal and task rules into a built-in goals plugin](../tasks/goals-plugin.md)
- [Enable plugins and their types in timeline.yaml](../tasks/plugin-config.md)
- [Let plugins register commands](../tasks/plugin-commands.md)
- [Run external validator plugins over a JSON protocol](../tasks/external-plugin-protocol.md)

## Done when

- `core/` contains no goal- or task-specific code: no children, cycles, inherited why, or linked-step rules.
- Every existing goal and task test passes against the goals plugin, and this repository's goals and tasks check exactly as before.
- `tline goal` and `tline task` exist only because the goals plugin registers them.
- An example external plugin, written as a short script, adds a diagnostic to a configured type through the JSON protocol.

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

Built-in plugins are Go packages compiled into `tline` that implement the core interface. External plugins are executables that speak a versioned JSON protocol: `tline` starts each once per command, sends the events and links on stdin with the current time, and reads diagnostics or output from stdout, with a timeout. They run only when the root's own `timeline.yaml` names them.

### Configuration

```yaml
plugins:
  - goals                         # built in
  - ticker: { run: ./tools/check-ticker }   # external
types:
  goal: { from: goals/goal, folder: goals }
  task: { from: goals/task, folder: tasks }
  prediction:
    folder: predictions
    entry_timing: future
    validators: [ticker]
```

A plugin depending on another plugin's types reads only that plugin's facts, never its meaning. Versioned plugin names and the protocol version keep this stable. WebAssembly plugins are a possible later option if untrusted plugins ever matter.
