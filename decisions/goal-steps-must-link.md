---
type: decision
timeline_at: 2026-10-13T04:32:45Z
created_by: claude
decision: Every step in a goal's How must link to a goal or task
decided_by: Badhree Babu
status: current
---

## Context

Plain-text steps had no owner, date, or status, so they could never be done or past due.

## Consequences

`tline check` rejects unlinked goal steps, and all work under a goal is tracked down to tasks.
