---
id: 01M4SY1Q58S56MC6SG61MZK3ZD
entered_at: 2026-10-13T06:08:57Z
entry_timing: future
type: task
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
task: Run external validator plugins over a JSON protocol
owner: Badhree Babu
status: open
---

## How

- Define a versioned JSON request (events, links, now) and response (diagnostics).
- Start each external plugin once per check with a timeout, and report a plugin that fails or times out as an operational error.
- Add an example shell or Python plugin and an end-to-end test that uses it.
