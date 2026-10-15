---
id: 01M4WFZBQR5QTKVCG1PS609TDQ
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review field, type, and link validation
owner: Badhree Babu
status: done
---

## How

- Read the field, status, unknown-field, duplicate-ID, and link parts of `core/validate.go`, and `core/config.go`.
- Try every field kind with wrong values, lists with wrong items, and links with anchors, escapes, URLs, and paths outside the root.

## Findings

- **No defects found.** Field kinds, allowed values, list items, unknown and duplicate fields, duplicate IDs, and link resolution behave as designed.
- **Acceptable.** Images (`![alt](x.png)`) are matched as links, which is harmless because only `.md` targets inside type folders can be events. Reference-style links and links split across lines are not recognized; a goal step written that way is reported as unlinked, so the problem is visible rather than silent.
