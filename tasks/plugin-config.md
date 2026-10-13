---
id: 01M4SY1Q58VGN97QPEY9ZH91NT
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-11-28T23:59:59Z
created_by: claude
task: Enable plugins and their types in timeline.yaml
owner: Badhree Babu
status: open
---

## How

- Decode a `plugins` list and `from: plugin/type` on a type, with `folder` and extra fields and sections.
- Keep `goal: { folder: goals }` working as shorthand for `from: goals/goal`, so existing roots need no change.
- Report unknown plugins and types clearly.
