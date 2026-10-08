# Engineering workflow

Start at the [context entry point](README.md). These practices are required for future implementation. Tool recommendations remain proposals until selected in [tooling](tooling.md).

## Implementation philosophy

Build small, understandable vertical slices with explicit contracts. Correct registration and recoverable history take priority over throughput. Keep pure validation separate from filesystem, Git, clock, and SQLite effects. Prefer ordinary Go values and functions; introduce small interfaces at real substitution or testing boundaries, owned by their consumers. Avoid one interface per struct, generic repositories, global state, and frameworks that obscure the acceptance path.

Use named domain types for IDs, timestamps, timing rules, revision identities, and structured errors. Validate external YAML, JSON, paths, and SQL results at their boundaries. Dynamic configured metadata needs an explicit validated representation; do not spread unchecked `map[string]any` through the core. Keep SQL and driver types inside the index adapter. Compile-time types do not replace runtime validation.

Errors must retain their cause and identify the operation, path, and violated requirement where useful. The CLI maps domain outcomes to documented exit codes and JSON; library code does not print, call `os.Exit`, or depend on terminal state. JSON stdout must remain parseable and include structured validation diagnostics, conflicts, and completeness status. Incidental human/debug logging belongs on stderr. Stable ordering and explicit time zones make behavior reproducible.

Keep changes reviewable. Each change includes the requirement, implementation, meaningful tests, context updates, and evidence from relevant checks. Fix a bug with a regression case that fails for the original behavior. Do not trade away validation or history guarantees to make a benchmark pass.

## Requirements before code

1. Create a feature branch from the agreed integration base. Inspect the working tree and preserve user edits.
2. Add or edit the requirement first, using the [index and template](requirements/README.md). Describe observable behavior, failure cases, compatibility, non-goals, and acceptance criteria. Existing behavior changes retain the same stable ID; genuinely new behavior gets a new ID.
3. Align system design and architecture where necessary. Resolve material ambiguity before dependent implementation; independent work may continue. A direct user request authorizes in-scope requirement edits without another approval ceremony.
4. Map each acceptance criterion to unit, integration, or executable E2E evidence. Define package boundaries and shared interfaces before parallel implementation.
5. Implement and test the smallest useful slice. A requirement-only branch is a valid starting point for agents; code must follow the branch's requirement changes, not remembered conversation.
6. Integrate, run required checks, and update requirement status with actual test paths/symbols and commands. Report unimplemented criteria explicitly.

Use `proposed`, `ready`, `in progress`, or `verified` status. “Ready” means the behavior is sufficiently specified, not that code exists. “Verified” requires implementation and passing acceptance evidence. Never mark work complete based only on generated files or a coverage percentage. Do not claim approval or measured performance that has not occurred.

## Context at every level

`AGENTS.md` points to `docs/README.md`; it does not duplicate this handbook. Each package adds `doc.go` when its first code is added, explaining purpose, owned invariants, dependencies, effects, concurrency/lifetime rules, error semantics, and relevant requirement IDs. Exported symbols document behavior, units, ownership, cancellation, and important edge cases. Use comments for reasoning and constraints, not line-by-line narration. Follow [Go documentation conventions](https://go.dev/doc/comment).

Tests document externally meaningful scenarios and link to requirement IDs in names or comments. SQL migrations explain the schema transition, compatibility, and recovery implications. Generated code identifies its generator and source inputs. User-visible configuration, JSON, and exit codes need examples and compatibility rules before release. A package README is warranted for a complicated recovery protocol, not for repeating `doc.go`.

Update context in the same change as code. A renamed package or requirement must not leave dead links. Record important decisions and rejected alternatives in the relevant canonical section, not a growing collection of overlapping architecture documents. Use Git history for superseded decisions.

## Parallel agents and worktrees

Implementation work should use subagents for independent slices, each in its own branch and Git worktree, followed by integration. Parallelize only after the shared requirement and contracts are defined. For a truly indivisible change, document why serial work is appropriate rather than creating overlapping edits for appearances.

The coordinating agent owns decomposition, shared contracts, acceptance criteria, integration, and the final result. Each assignment includes requirement IDs, a base commit, worktree path, owned paths, allowed dependencies, expected tests, and handoff format. Assign one owner to shared files such as `go.mod`, `go.sum`, CI, generated SQL, migration numbering, and root documentation. Workers propose changes to those files through the owner. A worker must verify its working directory before editing; tools must be explicitly directed to its assigned worktree.

Example, after a committed integration base exists:

```sh
git worktree add -b feat/validation ../timeline-validation <base-commit>
git worktree add -b feat/index ../timeline-index <base-commit>
```

Do not force worktree creation over a checked-out branch. Do not commit a user's unrelated files merely to create a base. Establish a committed integration base containing the requirements and shared contracts before splitting implementation. Preserve unrelated user edits.

Workers test and commit only their own changes, then report commit IDs, requirement criteria covered, check results, interface changes, and remaining risks. Integrate dependent contracts first and independent branches afterward, using reviewable merges or cherry-picks. Resolve semantic conflicts against requirements; rerun contract, integration, and E2E tests on the combined tree. A green worker branch is insufficient. Remove worktrees only after changes are safely integrated and no uncommitted work remains.

Worktrees have separate working files and indexes but share repository state including most refs. Do not let workers concurrently rewrite shared refs, run destructive cleanup, or use a common test repository/database. `git worktree lock` prevents pruning; it is not a runtime acceptance lock. See [Git worktree documentation](https://git-scm.com/docs/git-worktree).

## Review and handoff

A PR describes the observable before/after behavior, requirement IDs, tests run, migration or compatibility impact, and any remaining limitations. Review history integrity, transaction boundaries, concurrency, and recovery separately from style. Automated checks must be reproducible locally. Use narrow, justified lint suppressions; never disable a failing gate merely to merge.

The first code change must establish the [foundation requirements](requirements/foundation.md). No production behavior should accumulate while testing, lint, migrations, or environment setup are deferred to a later cleanup.
