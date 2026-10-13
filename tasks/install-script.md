---
id: 01M4VDZ6EGE9VV741S2W2HEWJK
entered_at: 2026-10-13T20:06:26Z
entry_timing: future
type: task
timeline_at: 2026-11-10T23:59:59Z
created_by: claude
task: Serve an install script at tline.sh
owner: Badhree Babu
status: open
---

## How

- Write a script that detects OS and CPU, downloads the matching release archive, verifies `SHA256SUMS`, and installs `tline`.
- Host it at `https://tline.sh` so `curl -fsSL https://tline.sh | sh` works, and publish it from the release workflow.
