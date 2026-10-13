---
id: 01M4SY1Q58DQN2BXNJPMJ7S7CS
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-11-28T23:59:59Z
created_by: claude
task: Move goal and task rules into a built-in goals plugin
owner: Badhree Babu
status: done
---

## How

- Create `internal/plugins/goals` with the goal and task type definitions and their rules: answers, linked steps, children, cycles, and inherited why.
- Remove that code from `core/validate.go`, keeping core link resolution.
- Move the existing goal and task tests with it, unchanged, and run `make plan` on this repository.
