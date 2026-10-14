---
id: 01M4WBAK00GCYX66XRCS5XJY6Z
entered_at: 2026-10-14T04:39:28Z
entry_timing: future
type: task
timeline_at: 2026-10-21T23:59:59Z
created_by: claude
task: Configure external plugins in timeline.yaml
owner: Badhree Babu
status: done
---

## How

- Accept `- name: { run: path-or-program, timeout: 10s }` in `plugins`, describe each plugin once per run, and build `from:` types and `validators:` from it.
- Report a clash with a built-in plugin, a core command, or another plugin's command or type as a configuration error.
