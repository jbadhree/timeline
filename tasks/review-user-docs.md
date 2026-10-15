---
id: 01M4WFZBQR0YS4TT02TTMVZF7R
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Check the user and agent guides against the built tline
owner: Badhree Babu
status: done
---

## How

- Read `README.md` and `docs/`.
- Run the commands they show against the built `tline` and the examples, and fix any that do not work as written.

## Findings

- **Corrected.** `README.md` called `tline` "planned" and said `tline history` belonged to a later SQLite store, though both exist.
- **Added.** `docs/plugins.md` now says what the approval fingerprint does not cover, and that the review escapes hidden characters.
- **Checked.** Every command `docs/agents.md` lists runs as documented against this repository, and `TestQuickstartGuide` keeps `docs/quickstart.md` honest.
