---
type: task
timeline_at: 2026-11-07T23:59:59Z
created_by: claude
task: Judge recorded events by the timing rule they were entered under
owner: Badhree Babu
status: done
---

## How

- Use the event's own `entry_timing`, not the type's current rule, for recorded events.
- Reject partial stamps, malformed IDs, and `entered_at` without a `Z` suffix.
