---
id: 01M4SXMJ8GRE7VFCX2DYYM01SR
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
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
