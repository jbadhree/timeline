---
id: 01M4SXMJ8GCEM75RS46BPN7XS4
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: task
timeline_at: 2026-10-31T23:59:59Z
created_by: claude
task: Run make check and make race in CI on macOS and Linux
owner: Badhree Babu
status: done
---

## How

- Add `.github/workflows/ci.yml` for pushes to `main` and pull requests.
- Pin `actions/checkout` and `actions/setup-go` to exact versions.
