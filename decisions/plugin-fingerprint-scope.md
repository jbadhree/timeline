---
id: 01M4Z2Q94G0M7NMCGX1F7MHGDT
entered_at: 2026-10-15T06:06:50Z
entry_timing: past
type: decision
timeline_at: 2026-10-14T00:00:00Z
created_by: claude
decision: Plugin approval fingerprints each plugin program file, not the files it loads, until plugins are shared beyond their own timelines
decided_by: Badhree Babu
status: current
---

## Context

The code review found that an approved plugin keeps running after a file it loads changes, such as a Python module beside the program. Today a timeline's external plugins are written by the people who use that timeline, so the gap is acceptable.

## Consequences

Approval stays as it is, and `docs/plugins.md` tells users to review changes to the files a plugin loads. A plugin marketplace, where people run plugins written by others, will need a stronger approval mechanism, such as fingerprinting everything a plugin ships or approving signed plugin releases; it is listed as a later direction in the primary goal.
