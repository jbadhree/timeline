---
id: 01M4WGQSZ0VZ7J74VAWTMFYQHR
entered_at: 2026-10-14T06:14:04Z
entry_timing: past
type: decision
timeline_at: 2026-10-14T06:00:00Z
created_by: claude
decision: A type folder that is a symbolic link to a directory is followed
decided_by: Claude, in the code review, for Badhree Babu to confirm
status: superseded
---

## Context

A type folder that was a symbolic link was scanned as an empty folder, so `tline check` reported a timeline with events in it as valid and checked nothing. Linking a folder from elsewhere, such as a shared notes folder, is a reasonable setup, and the alternative of rejecting links is just as simple to implement.

## Consequences

`tline` reads, stamps, and keeps history for events behind a linked type folder as if the folder were inside the root; event paths stay root-relative, through the link. Links to folders inside a type folder are still not followed, so a folder cannot loop or pull in a whole tree by accident. An event file that is a link is read through it, and stamping writes to the linked file and keeps the link, rather than replacing the link with a stamped copy. A type folder that is a broken link, or a file, is an error rather than an empty folder.

## Superseded by

- [Timeline does not follow symbolic links](no-symbolic-links.md)
