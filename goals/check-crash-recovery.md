---
type: goal
timeline_at: 2026-11-14T23:59:59Z
created_by: claude
goal: Recover cleanly when a check stops between writing IDs and committing
owner: Badhree Babu
status: open
---

## Why

`tline check` writes `id`, `entered_at`, and `entry_timing` into files before it creates the Timeline commit. If it stops in between, the files claim metadata that history does not support, and the next check would reject them as fabricated.

## How

- Keep a record under `.tline/` of the assignments a check has written, while it holds the writer lock.
- On the next check, use that record to finish the commit if the files still match, or report exactly which files need attention if they do not.
- Document the protocol in the `## Design` section of the parent recording goal.

## Done when

- A test that stops a check after writing files and before committing is followed by a check that records those events with the assigned IDs and entry times.
- If a file was edited in between, the next check reports it and never commits content it did not validate.

## Why it will work

Only one check holds the writer lock at a time, so the record cannot be overwritten by a concurrent check, and it lives outside Git so it never becomes part of history.
