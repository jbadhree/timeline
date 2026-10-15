---
id: 01M4Z2Q94GVH1QD27GZN0RDWCH
entered_at: 2026-10-15T06:06:50Z
entry_timing: past
type: decision
timeline_at: 2026-10-14T00:00:00Z
created_by: claude
decision: Timeline does not follow symbolic links in type folders
decided_by: Badhree Babu
status: current
---

## Context

The code review found that a type folder that was a symbolic link was scanned as empty, and that stamping replaced a linked event file with a copy. Following links would fix both, but a timeline whose events live behind links is hard to track: the files a person sees in the root are not where the events are, and the same file can appear in two timelines.

## Consequences

Every event file and type folder must be in the root itself. A type folder that is a symbolic link, or that is not a directory, stops `tline` with exit 1 and names the folder. A symbolic link inside a type folder, to a file or a folder, is reported by `tline check` as a problem in that path (exit 3), so nothing is written until it is replaced by the file or folder itself. Views mark their results incomplete while such a link exists.

## Supersedes

- [A type folder that is a symbolic link to a directory is followed](follow-linked-type-folders.md)
