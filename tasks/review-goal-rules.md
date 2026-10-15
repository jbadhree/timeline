---
id: 01M4WFZBQR3GQVSP3RX93KV6XF
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review goal and task rules
owner: Badhree Babu
status: done
---

## How

- Read `internal/plugins/goals/goals.go` and `model.go`.
- Check each rule in the goals-and-tasks Design, cycles, and how step text is shortened in messages.

## Findings

- **Fixed.** Long step text in "must link to a goal or task" messages could be cut inside a multi-byte character, printing `\xc3` in the message. Test: `TestLongStepTextIsShortenedWithoutSplittingCharacters`.
- **Acceptable.** Required answers, inherited why, link rules, and cycle detection match the design.
