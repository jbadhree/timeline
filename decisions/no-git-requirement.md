---
type: decision
timeline_at: 2026-10-13T00:57:31Z
created_by: claude
decision: Timeline does not require or use Git
decided_by: Badhree Babu
status: current
---

## Context

None of Timeline's product behavior needs Git; only history and detecting hand-edited stamps did, and those can come from a later record store.

## Consequences

Timeline works on plain files in any folder. `tline check` stamps drafts and never commits. Hand-edited stamps cannot be detected yet.

## Supersedes

- [Record events in Git commits made by tline check](record-with-git-commits.md)
