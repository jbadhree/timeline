---
id: 01M4WFZBQR54053KG2MD0RWDVA
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review how external plugins are run
owner: Badhree Babu
status: done
---

## How

- Read `internal/plugins/external/external.go` and `protocol.go`.
- Check timeouts, output limits, protocol and name checks, and what a plugin's type templates may declare.

## Findings

- **Fixed.** A plugin's type template could name a closed status that is not one of its status values, which `timeline.yaml` rejects for configured types. Tests: `TestDecodeTemplate` and a `badstatus` case in `TestExternalPluginFailures` (end to end).
- **Acceptable.** Timeouts kill the program and stop waiting for its children, output is limited to 64 MB, and protocol and name mismatches fail with exit 1. Approved plugins inherit the user's environment, as any program they run would.
