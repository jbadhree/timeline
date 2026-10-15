---
id: 01M4WFZBQR77VHNP9D66C29B3H
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: goal
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
goal: Review external plugin approval and execution for safety
owner: Badhree Babu
status: achieved
---

## How

- [Review the approval store and fingerprint](../tasks/review-trust-store.md)
- [Review how external plugins are run](../tasks/review-external-runner.md)

## Done when

- `internal/trust`, `internal/plugins/external`, and `internal/cli/pluginscmd.go` are read in full, assuming the timeline was written by someone else.
- No external plugin program can run before the user approves it, and what the user reviews before approving is what will run.
- Every finding is fixed with a test or recorded with a reason.

## Why it will work

External plugins are the one place tline runs code it did not ship. Reviewing them as an attacker who controls `timeline.yaml` and the files in the root, but not the user's configuration directory, gives a clear threat model to check against [plugin approval](../decisions/plugin-approval.md).
