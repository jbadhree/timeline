---
id: 01M4WFZBQRCT63QWZCYBD7B3RD
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review the goal and task views
owner: Badhree Babu
status: done
---

## How

- Read `internal/plugins/goals/views.go`.
- Run every view on this repository with and without `--json`, and check filters, drafts, and incomplete results.

## Findings

- **Fixed (docs).** The design said the views "show drafts separately"; they flag drafts inline, which the tests and guides rely on, so the text was corrected.
- **Acceptable.** Every view runs on this repository with and without `--json`. `goal tree` without an argument omits goals reachable only through a cycle, but validation reports the cycle and the view exits 5.
