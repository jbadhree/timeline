---
id: 01M4Z3GVG04DHZY36DPD6YTGS3
entered_at: 2026-10-15T06:20:48Z
entry_timing: future
type: task
timeline_at: 2026-12-15T23:59:59Z
created_by: claude
task: Warn when another tline on PATH hides the one install.sh installed
owner: Badhree Babu
status: open
---

## Why

After installing v0.5.0 into `~/.local/bin`, `tline --version` still printed v0.4.0, because an older copy in `~/go/bin` came first on the user's PATH. The script reported success and gave no hint.

## How

- After installing, compare `command -v tline` with the installed path; if they differ, print which copy runs instead and how to remove it or put the install folder first on PATH.
- Keep the script from editing shell files; decide separately whether to offer adding the folder to PATH, with a way to turn it off such as `TLINE_NO_MODIFY_PATH=1`.
- Cover it in the install-script CI job with a fake older `tline` placed earlier on PATH, and keep `shellcheck` passing.
