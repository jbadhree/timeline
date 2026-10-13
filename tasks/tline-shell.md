---
id: 01M4SXMJ8G9QWZBHGMYJD16HSJ
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-10-31T23:59:59Z
created_by: claude
task: Add the tline shell with --help, --version, --root, and --json
owner: Badhree Babu
status: done
---

## How

- Parse global flags with the standard `flag` package and print usage on `--help`.
- Print the version and commit for `--version`, as JSON with `--json`.
- Exit with code 2 for unknown flags and commands, leaving stdout empty.
