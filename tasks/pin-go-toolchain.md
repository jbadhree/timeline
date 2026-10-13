---
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Pin the Go toolchain and developer tools
owner: Badhree Babu
status: open
---

## How

- Set the minimum Go version in `go.mod` and the development toolchain with a `toolchain` line.
- Add golangci-lint at a fixed version with a short, explicit list of linters.
- Make CI install exactly those versions, never `latest`.
