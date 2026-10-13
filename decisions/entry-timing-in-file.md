---
id: 01M4SXMJ8GVSSDAK2XSX7JAKNY
entered_at: 2026-10-13T06:01:46Z
entry_timing: past
type: decision
timeline_at: 2026-10-13T00:19:05Z
created_by: claude
decision: Write entry_timing into each recorded event
decided_by: Badhree Babu
status: current
---

## Context

A reader of one file could not see whether an event was entered before, after, or as it happened without the type definition.

## Consequences

`tline check` stamps `entry_timing` with `id` and `entered_at`, and recorded events are judged by their own stamp.
