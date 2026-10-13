---
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
