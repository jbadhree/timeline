---
id: 01M4V98H587EQBZDK33DN564Y7
entered_at: 2026-10-13T18:44:09Z
entry_timing: past
type: decision
timeline_at: 2026-10-13T18:40:00Z
created_by: claude
decision: Whether and when to support external plugins written in any language
decided_by: Badhree Babu
status: current
---

## Context

Plugins are Go packages compiled into `tline`, such as the goals plugin. A team that needs its own rule, such as "every prediction's ticker must be a listed symbol", would have to change and rebuild `tline`. External plugins would be separate programs in any language that `tline` runs. They are not needed yet, so the question stays open.

## Options

- **External validators over JSON:** a timeline names a program, as in `plugins: [goals, ticker: { run: ./tools/check-ticker }]` with `validators: [ticker]` on a type. During `tline check`, after the core rules, `tline` starts it once with a versioned JSON request on stdin (the events of its types, their resolved links, and the current time) and reads diagnostics as JSON from stdout. It has a timeout; a crash, hang, or bad output fails the check with exit 1. It may only report on its own types and never writes files, and it runs only when the timeline's own `timeline.yaml` names it.
- **External types and commands** on the same protocol, such as `tline okr rollup` written in Python, once someone needs them.
- **WebAssembly plugins,** sandboxed and portable, if untrusted plugins ever matter.

The planned work is described in [the deferred task](../tasks/external-plugin-protocol.md).

## Decision

Decided on 2026-10-14: support external plugins as programs speaking a versioned JSON protocol, covering validators, type templates, and commands on the same protocol. tline starts the program once per request, writes one JSON request to its stdin, and reads one JSON response from its stdout. External plugins run only after the timeline's user approves them; see [approve external plugins per timeline](plugin-approval.md). WebAssembly plugins stay a later option.

## Consequences

External plugins implement the same core plugin interface as built-in ones and get the same guarantees: they see only facts, report only on their own types, never loosen a core rule, and never write event files. Each request starts a process, so plugins suit validation and occasional commands, not per-event calls. The protocol is documented in `docs/plugins.md` and must stay backward compatible: new fields may be added, existing ones keep their meaning.
