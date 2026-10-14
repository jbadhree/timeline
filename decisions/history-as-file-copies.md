---
id: 01M4VWWXJ0K4HE5N3PYZ89XBEN
entered_at: 2026-10-14T00:27:20Z
entry_timing: past
type: decision
timeline_at: 2026-10-14T00:20:00Z
created_by: claude
decision: Keep event history as plain file copies, per type
decided_by: Badhree Babu
status: current
---

## Context

Readers need to see earlier versions of recorded events, and a prediction's claim must not be rewritten after its outcome is known. History was planned for a later record store, but that is far off, and Timeline must keep working with no database and no version control ([no database in the first release](no-database-for-v1.md), [Timeline does not require Git](no-git-requirement.md)).

## Consequences

A type opts in with `history: true`. The event file is always the latest version, and `tline check` saves exact copies of earlier versions under `history/<type folder>/<event id>/v1.md`, `v2.md`, and so on, with a small `versions.log` beside them. Users keep `history/` with their timeline and never edit it. A type can freeze fields and sections, so a change from the first recorded version is rejected, and history types reject any change to their stamps. Both decisions above stay in force: copies are plain files. The hosted record store can later import these copies.
