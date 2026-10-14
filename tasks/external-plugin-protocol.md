---
id: 01M4SY1Q58S56MC6SG61MZK3ZD
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
task: Run external validator plugins over a JSON protocol
owner: Badhree Babu
status: done
---

## How

- Start an external plugin program once per request with a timeout, write one JSON request (`describe`, `validate`, or `command`) to its stdin, and read one JSON response from its stdout.
- Encode the same facts built-in plugins receive, and decode type templates, diagnostics, and command output.
- Report a missing program, non-zero exit, timeout, invalid JSON, or unsupported protocol version as an operational error that names the plugin.
