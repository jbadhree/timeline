---
id: 01M4WFZBQRR3R81WHXQS3M7HX7
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: goal
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
goal: Check that the documentation and goal designs match the code
owner: Badhree Babu
status: achieved
---

## How

- [Find design statements that the code contradicts](../tasks/review-design-drift.md)
- [Check the user and agent guides against the built tline](../tasks/review-user-docs.md)

## Done when

- Every contradiction found between a goal's `## Design`, the guides, and the code is resolved: the code is fixed, the text is corrected, or a decision is recorded.
- Local links in the changed documentation resolve.

## Why it will work

The designs live next to the plan and were written before the code; reading them again after the code exists is the quickest way to find where they drifted apart.
