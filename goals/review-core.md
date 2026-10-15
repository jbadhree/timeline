---
id: 01M4WFZBQRXB9C4HDBF2DNDX9W
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: goal
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
goal: Review the core rules against their design
owner: Badhree Babu
status: achieved
---

## How

- [Review timing rules and recorded metadata](../tasks/review-timing-rules.md)
- [Review field, type, and link validation](../tasks/review-field-validation.md)
- [Review the plugin facts and history comparison in the core](../tasks/review-core-plugin-and-history.md)

## Done when

- `core/` is read in full, and each rule is compared with [two times and timing](two-times-and-timing.md), [events and types](events-and-types.md), and [event history](event-history.md).
- Every finding is fixed with a unit test or recorded with a reason.

## Why it will work

The core is pure and has no I/O, so every rule can be probed with a unit test built from literal documents.
