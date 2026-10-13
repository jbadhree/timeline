---
id: 01M4SXMJ8GD6ZQGN12ZZ6KXWF7
entered_at: 2026-10-13T06:01:46Z
entry_timing: past
type: decision
timeline_at: 2026-10-13T00:19:05Z
created_by: claude
decision: Make goal and task built-in types with their own commands
decided_by: Badhree Babu
status: superseded
---

## Context

Much of what goes on a timeline is intended work, and a fixed shape lets the CLI check and present it.

## Consequences

`goal` and `task` are reserved type names. A later plugin system may move them into a built-in plugin without changing their files.

## Superseded by

- [Goal and task ship as a built-in plugin, not as core types](goals-and-tasks-as-plugin.md)
