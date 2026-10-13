---
id: 01M4SXMJ8GJAJPRC4FDSSRTVQJ
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-11-14T23:59:59Z
created_by: claude
task: Lock the root and re-read files before stamping
owner: Badhree Babu
status: done
---

## How

- Take `.tline/lock` so only one check stamps at a time, with a bounded wait.
- Re-read each draft just before writing it and stop if it changed since validation.
