---
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
