---
id: 01M4SY1Q58E2SA0BAF03WNT3HG
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
task: Let plugins register commands
owner: Badhree Babu
status: open
---

## How

- Route `tline <command>` to the plugin that registered it, after the core commands.
- Give the command the root's events, resolved links, the clock, and the `--json` flag.
- List plugin commands in `tline --help`.
