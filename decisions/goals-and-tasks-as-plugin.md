---
id: 01M4SY1Q58Z29KRJX75EMM0EKH
entered_at: 2026-10-13T06:08:57Z
entry_timing: past
type: decision
timeline_at: 2026-10-13T06:07:00Z
created_by: claude
decision: Goal and task ship as a built-in plugin, not as core types
decided_by: Badhree Babu
status: current
---

## Context

Goal and task were built into the core, with their link rules (children, cycles, inherited why, linked steps) in core code. [Link meanings belong to plugins](relations-in-plugins.md), and other domains need their own types and rules without changing Timeline.

## Consequences

The core keeps facts that hold in every domain: files, fields, statuses, two times, stamps, and resolved links. Goal and task move into a goals plugin compiled into `tline` that uses the same plugin interface as external plugins. Their file format and type names do not change, and `tline goal` and `tline task` become commands of that plugin. The goal and task views are written once, as plugin commands, after the plugin interface exists.

## Supersedes

- [Make goal and task built-in types with their own commands](goals-and-tasks-built-in.md)
