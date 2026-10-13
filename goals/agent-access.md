---
id: 01M4VDZ6EGF2391C1BD3FQ5ENZ
entered_at: 2026-10-13T20:06:26Z
entry_timing: future
type: goal
timeline_at: 2026-11-30T23:59:59Z
created_by: claude
goal: Make it easy for any agent and person to start using Timeline
owner: Badhree Babu
status: open
---

## How

- [Build a tline MCP server](../tasks/mcp-server.md)
- [Serve an install script at tline.sh](../tasks/install-script.md)
- [Decide whether tline creates timelines and event files](../tasks/init-and-new-decision.md)

## Done when

- An agent in an MCP client can check and record events and read goals and tasks without shell commands.
- `curl -fsSL https://tline.sh | sh` installs the latest release on macOS and Linux.

## Why it will work

MCP is how agents use tools today, and a one-line install removes the main barrier for people; both grow free usage, which is where paying teams come from.
