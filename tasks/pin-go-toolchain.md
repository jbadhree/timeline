---
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Pin the Go toolchain and developer tools
owner: Badhree Babu
status: done
---

## How

- Set the minimum Go version in `go.mod` and the development toolchain with a `toolchain` line.
- Add golangci-lint at a fixed version with a short, explicit list of linters.
- Make CI install exactly those versions, never `latest`.

Done 2026-10-12: Go 1.26.0 minimum with `toolchain go1.27.2`; golangci-lint v2.14.0 pinned in the `Makefile` with errcheck, govet, ineffassign, staticcheck, unused, and gofmt; CI pins actions/checkout v7.0.1 and actions/setup-go v7.0.0.
