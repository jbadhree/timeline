---
id: 01M4WFZBQR05BZXT6WSSXAFSMB
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Find design statements that the code contradicts
owner: Badhree Babu
status: done
---

## How

- Collect each contradiction the other review tasks found between a goal's Design and the code.
- Read the Design sections of the achieved goals once more for statements the code does not honor.
- Fix the code or correct the text, and record a decision when the behavior itself changes.

## Findings

- **Corrected.** The timing goal said "unchanged events are never revalidated", contradicting the recording goal and the code; it now says every check validates every event but judges timing by the event's own stamps.
- **Corrected.** The primary goal's command line section lacked `tline plugins` and exit 6.
- **Corrected.** The engineering foundation listed a `tests/integration/` folder that does not exist, omitted three packages, and said every package has a `doc.go`.
- **Corrected.** The goals goal said the views show drafts separately.
- **Added.** The events, timing, and history designs now state the behavior fixed in this review: linked folders and files, byte order marks, headings and fences, the timing rule after a type changes, new events at a removed event's path, case-insensitive reserved folders, and plugin review escaping. The behavior choices are recorded as decisions: [no symbolic links](../decisions/no-symbolic-links.md) and [fingerprint only plugin programs for now](../decisions/plugin-fingerprint-scope.md).
