---
id: 01M4Z3GVG0M116NKC2X1P3CMTQ
entered_at: 2026-10-15T06:20:48Z
entry_timing: future
type: task
timeline_at: 2026-12-24T23:59:59Z
created_by: claude
task: Report test coverage and untested Done when items before each release
owner: Badhree Babu
status: open
---

## How

- Before tagging a release, run `go test -coverprofile` across the module and list functions below 70% statement coverage, remembering that code exercised only through the built binary shows as uncovered.
- For each goal changed since the last release, match every `## Done when` item to the tests that demonstrate it, and list the items without one.
- Record each gap as a task under the test debt goal, fix it, or note why it is accepted, and link the report from the release notes.
