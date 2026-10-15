---
id: 01M4WFZBQRXZT9DF50CQHD7T6Z
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: goal
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
goal: Review the tline commands
owner: Badhree Babu
status: achieved
---

## How

- [Review tline check](../tasks/review-check-command.md)
- [Review tline history](../tasks/review-history-command.md)
- [Review command routing, help, and exit codes](../tasks/review-command-routing.md)

## Done when

- `internal/cli` and `cmd/tline` are read in full.
- Each command's flags, exit codes, JSON, and stdout and stderr match the [primary goal's command line](primary-goal-build-timeline.md#command-line) and the owning goal's design.
- Every finding is fixed with a test or recorded with a reason.

## Why it will work

The CLI is thin by design, so reviewing it is mostly checking that it passes the right facts to the core and maps results to the documented exit codes.
