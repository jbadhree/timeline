---
id: 01M4SY1Q58JGG9DGVR2A9SB54F
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-11-21T23:59:59Z
created_by: claude
task: Define the plugin interface in the core
owner: Badhree Babu
status: open
---

## How

- Add a `Plugin` interface in `core/` with optional types, validate, and commands.
- Pass plugins the events of their types, their resolved links, and the current time; accept only diagnostics or output back.
- Run plugin validation after the core rules and merge and sort all diagnostics.
