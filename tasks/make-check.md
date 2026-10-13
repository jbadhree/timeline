---
id: 01M4SXMJ8GZHPCSQ8T0QAX6HP0
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-10-31T23:59:59Z
created_by: claude
task: Add make check for formatting, build, lint, and tests
owner: Badhree Babu
status: done
---

## How

- Add `Makefile` targets `fmt-check`, `build` (without CGO), `lint`, `test`, and `race`.
- Install golangci-lint at the pinned version into `bin/` on first use.
