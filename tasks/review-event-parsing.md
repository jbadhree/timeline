---
id: 01M4WFZBQRA7TTD53Y7E52TZDC
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review event file parsing
owner: Badhree Babu
status: done
---

## How

- Read the parsing and scanning parts of `internal/document/document.go`.
- Try a file with a UTF-8 byte order mark, CRLF line endings, headings ending in `#`, code fences of different lengths and with info strings, and a type folder that is a symbolic link.

## Findings

- **Fixed.** A file starting with a UTF-8 byte order mark was rejected as having no frontmatter; it is now read, and stamping keeps the mark.
- **Fixed.** `## C#` was read as the heading `C`: every trailing `#` was stripped, not only a closing run after a space.
- **Fixed.** Code fences closed on any line starting with three backticks or tildes, so a four-backtick fence closed early (exposing headings inside it) and a line such as ```` ```go ```` closed a fence it should not.
- **Fixed.** A type folder that was a symbolic link was scanned as an empty folder, so `tline check` reported "all valid" and checked nothing. Timeline now rejects symbolic links, as decided in [no symbolic links](../decisions/no-symbolic-links.md): a linked or non-directory type folder is an error, and a link inside a type folder is reported as a problem.
- Tests: `TestHeadingsAndFences`, `TestByteOrderMark`, `TestInsertStamp`, `TestScanRejectsLinks`, `TestCheckByteOrderMark`, and `TestCheckRejectsSymbolicLinks` (end to end).
