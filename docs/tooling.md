# Tooling options

Research date: 5 October 2026. These are recommendations based on primary documentation, not installed tooling or benchmark results. Engineering requirements are mandatory; the library selections below remain proposed. Pin exact compatible versions in the first implementation change after a small compatibility check. Do not install `latest` implicitly in CI.

Database policy is now agreed: local SQLite first, with configurable targets and replaceable adapters under [TL-DATABASE](requirements/TL-DATABASE.md). The SQLite driver remains a tooling choice. PostgreSQL is a possible future backend; it is not needed for initial development.

## Recommended starting stack and alternatives

| Concern | Recommended starting option | Alternative and tradeoff |
| --- | --- | --- |
| Unit tests | Standard `testing`, table tests, native fuzzing | Assertion helpers only if repeated test code justifies another dependency |
| Executable E2E | Standard `os/exec` harness first | `testscript` for readable multi-command fixtures if it reduces harness complexity |
| Lint/types | Go compiler, `gofmt`, pinned golangci-lint with a curated set | `go vet` plus standalone Staticcheck is a smaller stack with more task wiring |
| SQLite | `modernc.org/sqlite` for CGO-free builds | `ncruces/go-sqlite3`, also CGO-free; compare actual workloads before deciding |
| Typed SQL | `sqlc` with committed generated Go | Hand-written `database/sql` for a very small query surface; more manual scan maintenance |
| SQL migrations | Embedded SQL with Goose's library Provider API | golang-migrate for separate up/down files and its library ecosystem |
| Task entry point | A pinned Task runner for portable task names | Make for Unix-focused contributors; direct Go commands remain documented either way |
| Environment | Native setup plus a pinned dev container using the same tasks | Native-only is lighter, but misses the requested ready-to-use remote environment |

Recommended combination: standard Go tests plus a built-binary harness, curated golangci-lint, modernc SQLite, sqlc, and Goose. Validate the SQL combination together before committing to it. Testscript is optional; do not introduce two overlapping workflow DSLs by default.

## Testing and static analysis evidence

The standard [testing package](https://pkg.go.dev/testing) supplies tests and benchmarks. [Testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) offers isolated fixtures and a portable script language, not a system shell. Use real executable tests for process startup, exits, signals, and multi-process behavior even if in-process helpers are also used.

[Go vet](https://pkg.go.dev/cmd/vet) checks suspicious constructs; compilation is the type checker. [Golangci-lint](https://golangci-lint.run/docs/linters/) aggregates checks; use an explicit list and pin its binary version compatible with the selected Go compiler. Its [compatibility FAQ](https://golangci-lint.run/docs/welcome/faq/) explains build-toolchain constraints. Keep formatter and lint configuration distinct.

[Govulncheck](https://go.dev/doc/security/vuln/) adds known-vulnerability analysis. It complements dependency review and tests. Separate network-dependent vulnerability updates from offline ordinary test execution, while requiring the security check in CI.

## SQL compatibility check before selection

[Modernc SQLite](https://pkg.go.dev/modernc.org/sqlite) is a CGO-free `database/sql` driver; follow its warning about the matching `modernc.org/libc` dependency version. [Ncruces SQLite](https://github.com/ncruces/go-sqlite3) is an alternative to evaluate. Compare startup, binary size, memory, cancellation, transaction/locking behavior, and rebuild/query performance. Verify the runtime SQLite fixes described in [performance](performance.md).

[Sqlc's SQLite guide](https://docs.sqlc.dev/en/latest/tutorials/getting-started-sqlite.html) generates Go from SQL, but its [support matrix](https://docs.sqlc.dev/en/latest/reference/language-support.html) currently labels Go/SQLite support Beta. Verify every used SQL feature, nullability, dynamic metadata representation, and actual driver behavior. Generation is not runtime validation or migration execution. Fixed-width migration filenames must sort consistently; see [sqlc schema and migration inputs](https://docs.sqlc.dev/en/latest/howto/ddl.html).

[Goose Provider](https://pressly.github.io/goose/documentation/provider/) can run migrations through a supplied database handle. Prefer the library with our chosen driver over distributing its multi-driver CLI. [Golang-migrate](https://github.com/golang-migrate/migrate) is the alternative. Whichever is chosen, Timeline owns compatibility, cross-process startup coordination, unknown-schema refusal, and recovery policy. Do not assume a migration library provides all of them.

The selection experiment must create and upgrade a database, generate and execute representative queries, cancel work, contend across processes, rebuild from accepted fixtures, and build without CGO on target platforms. Record results here when complete. No driver has yet been benchmarked for Timeline.

## Reproducible tooling and setup

[Go tool dependencies](https://go.dev/doc/modules/managing-dependencies) can pin Go-based developer tools through `go.mod` on Go 1.24 or later. This facility is not a recommendation to choose 1.24 as the project's minimum. Choose a supported release when implementation begins and record minimum versus development toolchain separately; [Go toolchain selection](https://go.dev/doc/toolchain) can otherwise trigger downloads. Some tools prefer prebuilt distributions: use their supported pinned installation method instead of forcing all tools into one dependency graph.

[Task](https://taskfile.dev/docs/getting-started) supplies a task-file entry point; keep its tasks thin wrappers over documented commands. The [Dev Container specification](https://containers.dev/) supplies an interoperable remote-environment definition. Pin images/features and keep local and CI verification equivalent. A container cannot replace native macOS/Windows filesystem tests.

## Decisions still needed

Before implementation: select Go/tool versions, task runner, SQL stack after the experiment, and initial supported OS/architecture matrix. Before registration: finalize configuration/acceptance formats, timestamp precision, and locking/publication/recovery protocol in the architecture. Before release: define measured performance budgets and release tooling/channel. These choices must not postpone the first-code quality gates.
