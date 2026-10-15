---
id: 01M4WFZBQR40214DPGFF9DAXBF
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-22T23:59:59Z
created_by: claude
task: Review the approval store and fingerprint
owner: Badhree Babu
status: done
---

## How

- Read `internal/trust/trust.go` and `internal/cli/pluginscmd.go`.
- Assume a hostile `timeline.yaml`: check that nothing runs before approval, that the fingerprint covers what will run, and that the review screen shows exactly what will run, with no way for the file to hide text from it.

## Findings

- **Fixed (security).** The approval review printed each plugin's `run` setting raw. A hostile `timeline.yaml` could use terminal escapes, such as `"./tools/other\e[2K\r    runs:    ./tools/lint"`, to erase the real program from the screen and show a harmless one, so the user would approve a program they never saw. Settings with control or other invisible characters, including right-to-left overrides, are now shown quoted with them escaped, in `tline plugins`, `plugins trust`, and the not-approved message. Tests: `TestShownEscapesInvisibleCharacters` and `TestPluginReviewShowsHiddenCharacters` (end to end).
- **Accepted.** The fingerprint covers the plugin program file, not files it loads, so an approved Python plugin keeps running after a module beside it changes. Documented in `docs/plugins.md` and accepted for now in [fingerprint only plugin programs for now](../decisions/plugin-fingerprint-scope.md); a plugin marketplace is a later direction that will need more.
- **Acceptable.** Nothing runs before approval: configuration errors stop before any plugin is started, and `tline --help` checks approval first. Approvals live outside the timeline and are keyed by its absolute path, so reaching it by another path only asks again.
