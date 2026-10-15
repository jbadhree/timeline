---
id: 01M4WFZBQRKCHD1AR1END48AAS
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review tline check
owner: Badhree Babu
status: done
---

## How

- Read `internal/cli/check.go`.
- Check path selection (files, folders, the root, files outside folders, non-Markdown files), the recheck before stamping, and the output after a conflict.

## Findings

- **Fixed.** Naming a file that is not an event file, such as `tasks/notes.txt` or a hidden `.md` file, selected nothing and reported "all valid" with exit 0. It is now a usage error (exit 2). Tests: `TestSelectPaths` and new rows in `TestCheckUsageAndConfigErrors` (end to end).
- **Fixed (test gap).** Re-validating against the entry time read after validation was untested: the existing test's draft already failed the first validation. `TestCheckFutureDraftThatPassesDuringValidation` moves the clock between the two reads, and fails if the recheck is removed.
- **Acceptable.** Output after a conflict, JSON fields, and exit codes match the design.
