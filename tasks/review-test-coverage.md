---
id: 01M4WFZBQRHEN8781C2M11SDQ6
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-24T23:59:59Z
created_by: claude
task: Review test coverage against each Done when
owner: Badhree Babu
status: done
---

## How

- Run `go test -cover` across the packages and list weakly covered functions.
- For each achieved goal, match its `## Done when` items to tests, and list any item without one.

## Findings

- **Fixed.** The end-to-end tests build `tline` in a subprocess, which `go test` cannot see, so after a change to the code they could be reported as passing from the test cache: a broken binary passed `make check`. The tests now open every source file while running, which makes those files part of the cache key. Verified: an edit to `internal/document` reruns them, and an unchanged tree still uses the cache.
- **Fixed.** The timing goal's criterion that changing a type's rule does not change how recorded events are judged had no test and was false; see the timing review. The recheck before stamping had no test; see the check review.
- **Acceptable.** In-process statement coverage is 74%; `internal/plugins/external` shows 0% there because its tests run the built binary end to end. "A deliberately failing test fails CI" is checked by hand.
