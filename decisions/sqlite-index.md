---
id: 01M4SXMJ8G5J56B9ZH5BHRFVG6
entered_at: 2026-10-13T06:01:46Z
entry_timing: past
type: decision
timeline_at: 2026-10-08T06:20:19Z
created_by: claude
decision: Keep a rebuildable SQLite index of events
decided_by: Badhree Babu
status: superseded
---

## Context

The first design used a local SQLite index, rebuilt from Git, for fast queries.

## Consequences

The index brought schema versions, rebuilds, and drift between the index and the files.

## Superseded by

- [No database in the first release](no-database-for-v1.md)
