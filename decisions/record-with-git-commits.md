---
id: 01M4SXMJ8GMQB5Z8XMA8HHH1T9
entered_at: 2026-10-13T06:01:46Z
entry_timing: past
type: decision
timeline_at: 2026-10-08T06:20:19Z
created_by: claude
decision: Record events in Git commits made by tline check
decided_by: Badhree Babu
status: superseded
---

## Context

The first design made Git history the source of truth for entry times and history, so a hand-typed `entered_at` could be detected.

## Consequences

Recording needed temporary Git indexes, crash recovery between writing and committing, and special handling for edits committed by hand.

## Superseded by

- [Timeline does not require or use Git](no-git-requirement.md)
