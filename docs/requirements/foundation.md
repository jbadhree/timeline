# TL-FOUNDATION: Engineering foundation

Status: ready for implementation; tooling choices are recommendations pending selection. No code or automation is implemented yet.

## Purpose

The first application code must arrive with the environment, context, tests, types, lint, and SQL migration infrastructure needed to maintain it. This requirement governs the engineering foundation; product behavior remains in the [system design](../system-design.md) and [architecture](../architecture.md).

## Acceptance criteria

- TL-FOUNDATION/AC-1: A clean checkout can follow [development setup](../development.md) to install pinned tools, build, and run checks without personal credentials or a developer's global configuration. Native and dev-container setup are both documented and exercised in CI.
- TL-FOUNDATION/AC-2: CI and local tasks enforce formatting, compilation/type checking, curated lint, unit tests, real Git/SQLite integration tests, and executable E2E tests. A deliberately failing check returns a failure; missing suites are not silently reported as passing.
- TL-FOUNDATION/AC-3: A minimal executable E2E smoke test exists with the initial CLI shell. Every behavior subsequently added receives the required tests in the same change. No empty test suite counts as product verification.
- TL-FOUNDATION/AC-4: The first SQLite schema is a versioned, embedded migration. Empty-database setup, supported upgrades, interruption/failure, and newer-unknown-version refusal have integration coverage. Query generation, if selected, is reproducible and checked for drift.
- TL-FOUNDATION/AC-5: Each implemented package has context and dependency rules, and each implemented requirement has criterion-to-test traceability. The context entry point and links pass automated checks.
- TL-FOUNDATION/AC-6: Race checks run on a supported CI platform; cancellation and cross-process writer conflicts have separate tests. Native runtime tests run on every supported release OS before claiming support.
- TL-FOUNDATION/AC-7: The repository provides reproducible benchmark fixtures and a baseline report format before performance-sensitive implementation is accepted. Performance claims include environment, workload, and measurements.
- TL-FOUNDATION/AC-8: At least one integration of independent implementation worktrees is validated on the combined tree. Shared contracts and ownership are recorded; no unrelated user changes enter worker commits.

## Verification

Implementation must add real test paths and task output for each criterion here. All criteria are currently unimplemented. The [testing guide](../testing.md), [performance guide](../performance.md), and [tooling options](../tooling.md) define the required evidence and candidate tools.
