---
type: decision
timeline_at: 2026-10-13T00:19:05Z
created_by: claude
decision: Make goal and task built-in types with their own commands
decided_by: Badhree Babu
status: current
---

## Context

Much of what goes on a timeline is intended work, and a fixed shape lets the CLI check and present it.

## Consequences

`goal` and `task` are reserved type names. A later plugin system may move them into a built-in plugin without changing their files.
