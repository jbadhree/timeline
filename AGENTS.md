# Timeline repository instructions

Start at [docs/README.md](docs/README.md), the single context entry point for humans and agents. Follow its reading path to the authoritative design, requirements, architecture, and engineering guides. Research supports decisions; it does not override requirements.

- Keep Timeline reusable. Consumer-specific types, domain logic, and agent runtimes belong to the applications using it.
- Keep checked-in documentation and context independent of particular consumers. Private handoff notes belong in ignored local files; they must not be required to understand or implement Timeline.
- Follow the architecture's Go core and `tline` CLI boundary. Do not build the deferred TUI during the initial implementation.
- Preserve the distinction between pending files and recorded items, original registration times, saved type definitions, and history. Markdown, configuration, and preserved Git history are authoritative; the configured database is a rebuildable index.
- Use local SQLite initially behind the [configurable database contract](docs/requirements/TL-DATABASE.md). Keep core behavior independent of database drivers and dialects; additional backends must pass the shared contract tests.
- Preserve existing user work and unrelated staged changes. Keep dependencies modest and language simple.
- Keep this initial handoff limited to documentation. Add application code only when implementation is requested.
- Document or change requirements before implementing them. Keep requirement IDs, acceptance tests, and package context connected in the same branch.
- Use subagents in separate branches and worktrees for independent implementation slices, with explicit contracts and ownership; integrate and test the combined result. Follow the [parallel workflow](docs/engineering.md#parallel-agents-and-worktrees).
- Establish the [engineering foundation](docs/requirements/foundation.md) with the first code: types, lint, unit/integration/executable E2E tests, SQL migrations, reproducible setup, and CI. Do not defer these to cleanup.

For documentation changes, verify local links and whitespace. For implementation, follow the [testing](docs/testing.md) and [performance](docs/performance.md) requirements. Report actual checks and limitations; do not claim proposed tooling, passing tests, or benchmark results exist before they do.
