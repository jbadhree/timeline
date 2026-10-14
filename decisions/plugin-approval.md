---
id: 01M4WBAK00SVMEXR6WJR0G0X7G
entered_at: 2026-10-14T04:39:28Z
entry_timing: past
type: decision
timeline_at: 2026-10-14T04:35:00Z
created_by: claude
decision: Run a timeline's external plugins only after its user approves them
decided_by: Badhree Babu
status: current
---

## Context

An external plugin is a program named in `timeline.yaml`, so running `tline` in a timeline someone else wrote would run their programs with the user's permissions. People and agents open timelines they did not write, and `tline` has so far been safe to run in any folder.

## Consequences

`tline` runs a timeline's external plugins only after the user approves them with `tline plugins trust`, which records the timeline's folder and a fingerprint of its plugin settings and in-repository plugin programs in the user's own configuration, never in the repository. Until then, or after the settings or programs change, commands that need the plugins stop with exit 6 and explain how to review and approve them. Approval asks for confirmation in a terminal and needs `--yes` otherwise, and the agent instructions tell agents to ask their user instead of approving plugins themselves. `TLINE_TRUST_PLUGINS=1` skips approval for environments such as CI that only run code their owner already chose.
