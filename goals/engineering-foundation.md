---
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
- Add a `tline` shell with `--help`, `--version`, and `--root`.
- Add one local command that runs formatting, build, curated lint, unit, integration, and end-to-end tests.
- Run that command in CI on every pull request, on macOS and Linux.

## Done when

- A clean checkout builds and passes all checks with one documented command, without personal credentials or global configuration.
- An end-to-end test runs the built `tline` binary.
- A deliberately failing test fails CI.

## Why it will work

A small first slice proves the toolchain on every platform before the recording logic depends on it, and every later goal's `## Done when` then has a place to become tests.

## Design

### Language and packaging

Use Go for the CLI and core. Build platform-specific `tline` executables without CGO so users do not need Go installed. Git is a runtime prerequisite; Timeline calls the installed `git` executable. Keep dependencies modest. Consumers use an installed `tline` binary and keep their own configuration, events, and domain logic.

### Packages

Validation, recording, history, queries, and goal and task views live in a reusable core. The CLI parses arguments and renders results; it does not reimplement rules. A later TUI or other front end uses the same core and must not add another writable source of truth. Use one Go module, and add packages when their first behavior is implemented, not as empty scaffolding.

| Path | Responsibility |
| --- | --- |
| `cmd/tline/main.go` | Process entry, cancellation, wiring, exit code |
| `internal/cli/` | Arguments, terminal and JSON rendering, exit-code mapping |
| `core/` | Use cases, domain values, validation, and the narrow interfaces adapters implement |
| `internal/config/` | Root discovery and `timeline.yaml` decoding |
| `internal/document/` | Markdown and frontmatter decoding, file writes |
| `internal/gitstore/` | Timeline commits and history through the `git` executable |
| `tests/integration/`, `tests/e2e/` | Real Git and built-executable scenarios |

`core/` must not import the CLI or Git process code. Inject the clock and effects, pass cancellation explicitly, and keep dependencies acyclic without a shared `utils` package. Each package documents ownership and invariants in `doc.go`.

### Tests

| Level | Boundary | Examples |
| --- | --- | --- |
| Unit | Pure logic, no Git processes | Timing boundaries, field validation, decoding, ordering |
| Integration | Real temporary Git repositories | All-or-nothing recording, moves and removals, preserved staged changes |
| End to end | The built `tline` binary | Flags, root discovery, exit codes, JSON, stdout and stderr, effects across commands |

Each test owns its temporary files, repository, Git identity, and environment, and needs no network. Concurrency tests use separate processes and explicit synchronization, not sleeps.

### Data layout

```text
my-project/
  timeline.yaml
  .tline/            ignored by Git
    lock
  goals/
  tasks/
  news/
  predictions/
```

Folders are chosen in `timeline.yaml`. Event files and `timeline.yaml` are tracked in Git; `.tline/` is not.
