---
id: 01M4WFZBQRDCPSY93NT04N8TWN
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review timeline.yaml decoding and root discovery
owner: Badhree Babu
status: done
---

## How

- Read `internal/config/config.go`.
- Try reserved, nested, and duplicate folders, renamed plugin types, unknown keys, external plugin entries, and wrong value types.

## Findings

- **Fixed.** Reserved folder names (`history`, `.tline`, `.git`) and overlapping type folders were compared case-sensitively. On a case-insensitive disk, as on most Macs, a type folder named `History` is the history folder: its copies were read back as events, with duplicate IDs. Tests: `TestDecodeErrors` and the reserved-folder cases in `internal/config`, and `TestFolderNamedLikeHistoryInAnotherCaseIsReserved` (end to end).
- **Acceptable.** `tline plugins` lists an external entry named like a built-in plugin, but decoding rejects it, so nothing runs. Root discovery, strict keys, renamed plugin types, and value types behave as designed.
