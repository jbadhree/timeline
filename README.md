# Timeline

Timeline records when an event happens and when it enters the system. Humans and agents edit Markdown items directly, with configurable types and Past, Future, or Real-time entry rules.

## Design

- [Start here](docs/README.md): the context entry point, requirements, and contributor reading path.
- [System design](docs/system-design.md): the authoritative product behavior.
- [Architecture](docs/architecture.md): the authoritative implementation plan.
- [Examples](examples/README.md): worked configuration and event files.
- [Build versus customize](docs/build-vs-customize.md) and [product research](docs/research.md): supporting context, not specifications.

The initial database is local SQLite. [Database selection](docs/requirements/TL-DATABASE.md) is configurable, with a shared contract for future supported backends; databases remain rebuildable indexes.

## Status

This repository is ready for implementation planning; application code has not been written. The planned Go executable is `tline`, with `check`, `list`, and `history` commands backed by a reusable core. A TUI is deferred. See the architecture for command behavior, storage, recovery, and verification requirements.

Applications retain their own configuration, items, and domain logic and use an installed `tline` binary without a Git submodule.

Implementation follows documented requirements first, with unit, integration, and executable E2E tests from the first code. See the [engineering workflow](docs/engineering.md), [development setup contract](docs/development.md), and [researched tooling options](docs/tooling.md). Tooling and setup are documented proposals, not installed automation yet.

The configuration and acceptance-record formats, timestamp precision, locking and recovery details, minimum Go version, libraries, and release tooling remain implementation choices. See [AGENTS.md](AGENTS.md) before making changes.
