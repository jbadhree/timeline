---
id: 01M4SXMJ8GQNYKRS16KFP25DB3
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-12-19T23:59:59Z
created_by: claude
task: Implement tline goal list, show, and tree
owner: Badhree Babu
status: open
---

## How

- `goal list`: goals by target date with status, owner, and past-due flags.
- `goal show`: the seven answers, with every linked step's type, status, and past-due flag.
- `goal tree`: goals and tasks depth first in How order, with `--json`.
- Build it inside the goals plugin, as plugin commands rather than core code.
