---
id: 01M4Z3GVG005ZVQ5APAY8Q2FPH
entered_at: 2026-10-15T06:20:48Z
entry_timing: future
type: goal
timeline_at: 2026-12-31T23:59:59Z
created_by: claude
goal: Keep test and tooling debt visible, and pay it down before each release
owner: Badhree Babu
status: open
---

## Why

The code review found problems that the tests did not catch: a `## Done when` item marked achieved without a test, end-to-end results served from a stale cache, and an install that left an older `tline` first on the user's PATH. Debt like this builds up between reviews unless something looks for it on a schedule. This goal runs until the first release, then is renewed with a new date.

## How

- [Report test coverage and untested Done when items before each release](../tasks/release-test-debt-report.md)
- [Warn when another tline on PATH hides the one install.sh installed](../tasks/install-warns-about-shadowed-tline.md)

## Done when

- Every release up to the first one has a test debt report, and each item in it is fixed, planned as a task under this goal, or accepted with a reason.
- Every task under this goal is done or dropped with a reason.

## Why it will work

Tying the report to releases gives it a natural rhythm without a scheduler, and recording each piece of debt as a task under this goal keeps it visible in `tline goal tree` until it is paid.
