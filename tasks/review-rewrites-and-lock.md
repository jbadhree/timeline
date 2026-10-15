---
id: 01M4WFZBQRZQY7WJPS608T1HD2
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review safe rewrites and the writer lock
owner: Badhree Babu
status: done
---

## How

- Read `InsertStamp` and `Rewrite` in `internal/document/document.go`, and `internal/lock/lock.go`.
- Check that a stamped file keeps every other byte, its line endings, and its mode, and that a stale lock is explained.

## Findings

- **Fixed.** Stamping an event file that was a symbolic link replaced the link with a stamped copy and left the linked file a draft. Linked event files are now rejected before anything is written, as decided in [no symbolic links](../decisions/no-symbolic-links.md). Test: `TestCheckRejectsSymbolicLinks` (end to end).
- **Acceptable.** Stamps keep every other byte, CRLF line endings, and the file mode. The lock has no stale-lock detection, as designed; its message names the file to delete. An editor saving in the moment between the final re-read and the rename could still lose that save; closing it would need OS-specific locking, which the design deliberately avoids.
