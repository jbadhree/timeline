---
id: 01M4VWWXJ02KHJQ75B5C9H871C
entered_at: 2026-10-14T00:27:20Z
entry_timing: future
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Save history copies in tline check
owner: Badhree Babu
status: open
---

## How

- Save v1 when a history-type draft is stamped, and the next version when a recorded event differs from its latest copy.
- Save v1 for recorded events without copies, as `history-enabled` or `late`, and log every copy in `versions.log`.
- Never write in a dry run or when any selected file is invalid; re-read files before copying.
