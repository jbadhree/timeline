---
id: 01M4WFZBQRJQ1K2TYD225MZNA9
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review command routing, help, and exit codes
owner: Badhree Babu
status: done
---

## How

- Read `cmd/tline/main.go`, `internal/cli/cli.go`, `internal/cli/timeline.go`, and `internal/cli/plugincmd.go`.
- Check that plugin commands see the same problems `tline check` reports, and that every exit code matches the primary goal's command line table.

## Findings

- **Fixed.** Plugin commands such as `tline task list` were checked without history, so a changed frozen field or stamp, which `tline check` rejects, left the view complete (exit 0) and showing the rewritten claim. Commands now get the same history facts as `tline check`. Tests: `TestPluginCommandSeesHistoryProblems` (CLI) and `TestViewsReportChangedFrozenFields` (end to end).
- **Fixed (docs).** Exit code 6 and `tline plugins` were missing from the primary goal's command line section.
