---
type: decision
timeline_at: 2026-10-13T00:19:05Z
created_by: claude
decision: No database in the first release
decided_by: Badhree Babu
status: current
---

## Context

Every query can be answered by reading the files, and a database was only a copy that could drift.

## Consequences

Commands read the files directly. A SQLite record store, with history, is a later direction.

## Supersedes

- [Keep a rebuildable SQLite index of events](sqlite-index.md)
