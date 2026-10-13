---
id: 01M4V98H587EQBZDK33DN564Y7
entered_at: 2026-10-13T18:44:09Z
entry_timing: past
type: decision
timeline_at: 2026-10-13T18:40:00Z
created_by: claude
decision: Whether and when to support external plugins written in any language
decided_by: Badhree Babu
status: open
---

## Context

Plugins are Go packages compiled into `tline`, such as the goals plugin. A team that needs its own rule, such as "every prediction's ticker must be a listed symbol", would have to change and rebuild `tline`. External plugins would be separate programs in any language that `tline` runs. They are not needed yet, so the question stays open.

## Options

- **External validators over JSON:** a timeline names a program, as in `plugins: [goals, ticker: { run: ./tools/check-ticker }]` with `validators: [ticker]` on a type. During `tline check`, after the core rules, `tline` starts it once with a versioned JSON request on stdin (the events of its types, their resolved links, and the current time) and reads diagnostics as JSON from stdout. It has a timeout; a crash, hang, or bad output fails the check with exit 1. It may only report on its own types and never writes files, and it runs only when the timeline's own `timeline.yaml` names it.
- **External types and commands** on the same protocol, such as `tline okr rollup` written in Python, once someone needs them.
- **WebAssembly plugins,** sandboxed and portable, if untrusted plugins ever matter.

The planned work is described in [the deferred task](../tasks/external-plugin-protocol.md).

## Consequences

Until this is decided, only built-in plugins exist, and `timeline.yaml` rejects external plugin entries with a clear message. The plugin interface in the core is already the shape external validators would use, so deciding later costs nothing now.
