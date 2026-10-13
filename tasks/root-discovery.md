---
id: 01M4SXMJ8GQJ2F8F8P904XPMG6
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-11-07T23:59:59Z
created_by: claude
task: Find the Timeline root from the nearest timeline.yaml or --root
owner: Badhree Babu
status: done
---

## How

- Search the working directory, then its parents, for the nearest `timeline.yaml`.
- With `--root`, require `timeline.yaml` in that directory and never fall back to another root.
