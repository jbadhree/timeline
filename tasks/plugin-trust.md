---
id: 01M4WBAK00S3H8XDW8VF4CHXQK
entered_at: 2026-10-14T04:39:28Z
entry_timing: future
type: task
timeline_at: 2026-10-21T23:59:59Z
created_by: claude
task: Approve external plugins per timeline
owner: Badhree Babu
status: done
---

## How

- Add `tline plugins`, `tline plugins trust`, and `tline plugins untrust`, storing approvals outside the repository.
- Stop with exit 6 and an explanation when a timeline's plugins are not approved or have changed; allow `TLINE_TRUST_PLUGINS=1` for CI.
