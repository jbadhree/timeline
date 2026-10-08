# Testing and quality gates

These are required implementation practices. The repository currently has no executable or test suite. Product scenarios come from the [requirements index](requirements/README.md), not from whatever the code currently happens to do.

## What the test levels mean for a CLI

| Level | Boundary | Examples | Placement once implemented |
| --- | --- | --- | --- |
| Unit | Pure logic with controlled inputs; no real Git processes or database | Timing equality, field validation, decoding, deterministic sorting, original registration invariants | Beside code in `*_test.go` |
| Integration | Real adapters collaborating through core APIs | File-backed SQLite with production migrations, real temporary Git repo, partial failure and recovery | Adapter tests and `tests/integration/` |
| End to end | The built `tline` binary, invoked as a user or agent invokes it | Flags, root discovery, exact exits, JSON, stdout/stderr, disk and Git effects across commands | `tests/e2e/` with readable fixtures |

Use standard [Go testing](https://pkg.go.dev/testing) initially. An in-process call to a CLI handler is useful but does not replace executable E2E coverage. Mock only narrow boundaries for controlled failures; do not mock Git and SQLite in the tests intended to verify them. Prefer behavioral assertions over large snapshots that reviewers routinely accept without inspection.

## Required scenarios

- Unit: Past/Future equality boundaries, Real-time delay edges, timestamp normalization/precision, malformed metadata, schema validation, IDs, preserved entry times, saved type definitions, and deterministic output order.
- Integration: selected-set all-or-nothing recording; duplicate IDs across scope; unrelated staged and unstaged content preserved; move/removal history; saved configuration versions; stale revision/hash rejection; ordinary Git commits never establishing registration.
- Recovery: crash/failure at each specified publication stage, repeated requests, missing/corrupt/stale index, missing historical evidence, and newer working files never overwritten from recovery state.
- E2E lifecycle: write a draft; run `check --dry-run`; run `check`; list JSON and human output; edit; observe pending status; run `check`; move and run `check` with both paths; remove and run `check`; inspect `history`; full-clone to another directory; run `check` for automatic index recovery and compare IDs, entry times, definitions, and revisions. There are no separate record or rebuild commands.
- E2E contract: invalid flags/roots, paths containing spaces, scope escape attempts, exit codes, machine-readable incomplete results, empty successful results, stdout/stderr separation, and cancellation. Test help against supported flags.
- Dry run: compare relevant source bytes, Git refs/index, persistent SQLite and sidecars before/after; no registration or persistent mutation. An unchanged normal check must not create another accepted revision.
- Concurrency: two separate CLI processes competing to register, an editor changing a selected file during validation, reader/writer overlap, bounded lock wait, cancellation, and retries. Use explicit barriers or controlled adapter fault points instead of timing sleeps.

Inject a clock into the core. Exercise a future item that expires between dry run and registration deterministically; do not add a production backdating flag merely to simplify E2E tests. Real executable tests can use safely separated future/past fixtures; exact boundary races belong in controlled core integration tests.

Each test owns temporary files, database, repository, Git identity/configuration, and environment. Disable dependence on global hooks, signing, locale, credentials, or a user's home. Use a local bare remote and ordinary clone for recovery tests. Normal tests require no external network. `t.Parallel` is appropriate only for isolated state; subprocess races need their own synchronization.

## Database adapter contract tests

Run the same domain-level suite against each real supported backend. Cover filters, deterministic ordering, timestamps, absent/null metadata, recorded revisions, transaction visibility, errors, and recovery. Begin with SQLite and two configured SQLite targets; switching targets must preserve semantic results and leave the old target unchanged. Test dataset isolation and no fallback on connection failure. Additional adapters need cross-backend reconstruction and executable E2E evidence before being called supported. See [TL-DATABASE](requirements/TL-DATABASE.md).

## SQL migration verification

Run the actual embedded migrations against the actual selected driver. Test empty-to-current and every supported previous schema-to-current, failure rollback or documented recoverable state, concurrent startup, and refusal of an unknown newer schema. Check schema version tracking and semantic equivalence between an upgraded index and one rebuilt from the same accepted history.

Migration ownership and behavior are specified in [architecture](architecture.md#sql-schema-and-migrations). A successful migration does not prove preserved historical registration; verify query results as well as tables. `sqlc` generation does not execute or validate runtime migration behavior.

## Static checks and types

Go compilation and tests provide static type checking. `go vet` is additional heuristic analysis. Require `gofmt`, `go build ./...`, curated lint, and relevant tests. Recommended lint selection: `govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, plus SQL resource checks after evaluating noise. Handle close/rows/transaction errors intentionally. Avoid broad lint-disable comments and “enable everything” configurations. See [tooling options](tooling.md).

## CI contract

The first code change supplies one local verification interface used by CI. Expected task names are defined in [development setup](development.md); they are not available yet.

| When | Required checks |
| --- | --- |
| Every PR | Documentation links/traceability, format, build/types, lint, unit, integration, executable E2E, migrations, generated-code drift, dependency checks, CGO-free build |
| Every PR on a supported race platform | `go test -race ./...` including exercised adapters; separate process-concurrency tests |
| Before claiming an OS is supported | Native unit/integration/E2E runs on that OS; cross-compilation alone is insufficient |
| Scheduled and parser/storage changes | Time-bounded fuzzing, repeated concurrency/crash tests, controlled benchmark comparison |
| Release | All gates, supported-platform executable smoke tests, reproducible version metadata and checksums, documented compatibility |

The [race detector](https://go.dev/doc/articles/race_detector) requires CGO enabled and appropriate compiler support; the CGO-free release build is a separate check. It only detects races in executed paths and cannot establish cross-process safety.

[Native fuzzing](https://go.dev/doc/security/fuzz/) should cover parsers, timestamps, identifiers, and acceptance-record codecs. Commit useful regression seeds; normal tests run seeds while dedicated jobs explore more inputs. Set explicit fuzz time/resource budgets.

Report coverage, especially uncovered timing and recovery branches, but do not use a percentage as a substitute for acceptance evidence. Set any numeric coverage floor after meaningful initial tests exist. To measure spawned executable paths, use [Go binary coverage](https://go.dev/doc/build-cover); normal package coverage does not automatically include a separately built binary.

CI must fail on missing required tests, uncommitted generation changes, or skipped required gates. Platform exclusions need a reason and a tracked resolution. A flaky test is a defect to investigate, not a reason for blind retries or deletion.
