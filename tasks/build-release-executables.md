---
id: 01M4SXMJ8GE2W6SPNRT23AM618
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-12-24T23:59:59Z
created_by: claude
task: Build tline executables for macOS and Linux with checksums
owner: Badhree Babu
status: open
---

## How

- Cross-compile `tline` without CGO for macOS and Linux on amd64 and arm64.
- Embed the version and commit, and publish SHA-256 checksums beside the binaries.
- Run the end-to-end tests natively on each OS before calling it supported.
