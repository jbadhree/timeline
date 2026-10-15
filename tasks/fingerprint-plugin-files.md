---
id: 01M4WGQSZ0X1H50G0XZD2S513Z
entered_at: 2026-10-14T06:14:04Z
entry_timing: future
type: task
timeline_at: 2026-11-15T23:59:59Z
created_by: claude
task: Decide whether plugin approval covers the files a plugin program loads
owner: Badhree Babu
status: done
---

## Why

The code review found that the approval fingerprint covers each plugin program file inside the root, but not the files it reads or imports. A Python plugin approved as `tools/plugin.py` keeps running without asking again after someone changes the `tools/helpers.py` it imports.

## How

- List the options: fingerprint the program's whole folder, let `timeline.yaml` name extra files to fingerprint, or keep the current rule and document it, which `docs/plugins.md` now does.
- Record the choice as a decision, update the Plugins goal's design, and implement it with unit, integration, and end-to-end tests if it changes behavior.

## Outcome

Keep the current rule: [fingerprint only plugin programs for now](../decisions/plugin-fingerprint-scope.md). A plugin marketplace is listed as a later direction in the primary goal and will need its own approval mechanism.
