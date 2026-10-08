# Development environment

Start at [project context](README.md). This is the setup contract for the first implementation change. There is currently no `go.mod`, Taskfile, dev container, or executable. The commands below are the interface to implement, not a claim that setup works today. Do not mark TL-FOUNDATION verified until a clean remote checkout can run them.

## Bootstrap contract

Provide native instructions for each supported OS and `.devcontainer/devcontainer.json` for remote agents. Both must use the same pinned Go and developer-tool versions and verification tasks. Document CPU, memory, disk, network, and supported filesystem prerequisites using observed setup/test needs.

Required tools will be Git, the selected Go toolchain, the selected task runner, and pinned lint/SQL-generation tools. Race-enabled tests also need platform-appropriate CGO/compiler support; release builds must be checked separately with CGO disabled. No external database server, cloud account, consumer application checkout, personal Git configuration, or credentials should be needed for ordinary development.

Pin the minimum supported Go language version in `go.mod` and the development/CI toolchain separately. Track `go.sum`; version-pin generators and linters. Explain automatic toolchain downloads and how restricted environments preinstall/cache them. Avoid global installs or shell-profile modifications when repo-local tooling works.

The first implementation PR must replace this provisional setup section with tested, copyable install commands for the chosen versions. A bootstrap task must detect missing prerequisites and print actionable errors, be safe to rerun, and avoid silently requesting administrator access. Downloads must use pinned versions and integrity checks where supported.

## Single task interface

Recommended Task interface, pending the [tool selection](tooling.md):

```sh
git clone https://github.com/jbadhree/timeline.git
cd timeline
# Install the pinned prerequisites using the OS-specific instructions to be added.
task setup
task doctor
task verify
```

These task names are a contract to implement:

| Task | Required behavior |
| --- | --- |
| `setup` | Fetch pinned dependencies/tools; no application data mutations |
| `doctor` | Print/check tool versions, Git, CGO/race prerequisites, and actionable setup errors |
| `fmt` / `fmt:check` | Apply formatting / check formatting without changes |
| `build` | Build `tline` with explicit output/version metadata |
| `lint` | Run the curated static checks |
| `test:unit` | Run pure tests |
| `test:integration` | Run real Git/SQLite adapter and migration tests |
| `test:e2e` | Build and run the executable suite |
| `test:race` | Exercise required Go packages with the race detector |
| `generate` / `generate:check` | Generate SQL/types / fail on drift without leaving user files changed |
| `docs:check` | Check local links, requirement references, and required package context |
| `security` | Run dependency vulnerability checks; document network needs |
| `bench` | Run reproducible benchmarks with environment metadata |
| `verify` | Run all local mandatory gates, including CGO-free build and race checks where supported |

Document the underlying Go/tool commands next to task definitions so the wrapper is not essential knowledge. Do not disguise skipped suites as successes. Resolve unsupported local race prerequisites explicitly; CI must still run the required supported-platform lane. CI and developer tasks must not evolve into separate workflows.

## Remote and isolated operation

The dev container installs the same pins and calls the same setup tasks. Keep it usable without production secrets, privileged containers, a mounted Docker socket, or access to consumer applications. Mount source with the correct user permissions. Document first-run network access and cache directories; ordinary tests run without network after dependencies are fetched.

Each agent uses an isolated worktree, build output, test directories, and SQLite files. Shared download caches are acceptable when the tool supports concurrent access. Never share a mutable test repository between workers. Use synthetic fixtures rather than personal data. See [parallel agent workflow](engineering.md#parallel-agents-and-worktrees).

## CI and release readiness

Create CI alongside the first code, invoking the same task interface. Run native tests on the supported OS matrix and a clean dev-container bootstrap smoke check. Cache by toolchain, OS/architecture, and dependency hashes; avoid treating caches as source of truth. Keep permissions minimal and pin third-party actions to full commit IDs as recommended by [GitHub's secure-use guidance](https://docs.github.com/en/actions/reference/security/secure-use).

Choose release tooling later without delaying build validation. Before distributing binaries, document installation, Git runtime prerequisite, supported platforms, version output, configuration/record compatibility, checksums, and how to report a reproducible failure. Local development should not require release credentials.
