---
id: 01M4SXMJ8GTP8XCHCGE00Z69N6
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: goal
timeline_at: 2026-10-31T23:59:59Z
created_by: claude
goal: Set up the Go module, package boundaries, tests, and CI so every later change arrives tested
owner: Badhree Babu
status: open
---

## Why

Tests, lint, and CI are cheapest to add before there is behavior to protect, and recording mistakes are expensive to undo once events carry wrong entry times.

## How

- [Pin the Go toolchain and developer tools](../tasks/pin-go-toolchain.md)
- [Add the tline shell with --help, --version, --root, and --json](../tasks/tline-shell.md)
- [Add make check for formatting, build, lint, and tests](../tasks/make-check.md)
- [Run make check and make race in CI on macOS and Linux](../tasks/ci-workflow.md)

## Done when

- A clean checkout builds and passes all checks with one documented command, without personal credentials or global configuration.
- An end-to-end test runs the built `tline` binary.
- A deliberately failing test fails CI.

## Why it will work

A small first slice proves the toolchain on every platform before the recording logic depends on it, and every later goal's `## Done when` then has a place to become tests.

## Design

### Language and packaging

Use Go for the CLI and core. Build platform-specific `tline` executables without CGO so users do not need Go installed. `tline` needs nothing else at runtime. Keep dependencies modest. Consumers use an installed `tline` binary and keep their own configuration, events, and domain logic.

### Packages

Validation, stamping, queries, and goal and task views live in a reusable core. The CLI parses arguments and renders results; it does not reimplement rules. A later TUI or other front end uses the same core and must not add another writable source of truth. Use one Go module, and add packages when their first behavior is implemented, not as empty scaffolding.

| Path | Responsibility |
| --- | --- |
| `cmd/tline/main.go` | Process entry, cancellation, wiring, exit code |
| `internal/cli/` | Arguments, terminal and JSON rendering, exit-code mapping |
| `core/` | Use cases, domain values, validation, and the narrow interfaces adapters implement |
| `internal/config/` | Root discovery and `timeline.yaml` decoding |
| `internal/document/` | Markdown and frontmatter decoding, file writes |
| `tests/integration/`, `tests/e2e/` | Real-filesystem and built-executable scenarios |

`core/` must not import the CLI or read files itself. Inject the clock and effects, pass cancellation explicitly, and keep dependencies acyclic without a shared `utils` package. Each package documents ownership and invariants in `doc.go`.

### Tests

| Level | Boundary | Examples |
| --- | --- | --- |
| Unit | Pure logic, no file access | Timing boundaries, field validation, decoding, ordering |
| Integration | Real temporary folders | All-or-nothing stamping, files changed during a check, concurrent checks |
| End to end | The built `tline` binary | Flags, root discovery, exit codes, JSON, stdout and stderr, effects across commands |

Each test owns its temporary files and environment, and needs no network. Concurrency tests use separate processes and explicit synchronization, not sleeps.

### Data layout

```text
my-project/
  timeline.yaml
  .tline/            local runtime state, not shared
    lock
  goals/
  tasks/
  news/
  predictions/
```

Folders are chosen in `timeline.yaml`. Event files and `timeline.yaml` are the timeline; `.tline/` only holds the writer lock and can be deleted when no check is running. Projects that use version control should ignore `.tline/`.
