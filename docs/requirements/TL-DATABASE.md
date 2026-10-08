# TL-DATABASE: Configurable index database

Status: ready for implementation. SQLite is the initial supported backend; other adapters are future work.

## Purpose and canonical context

A user must be able to point Timeline at another supported database through configuration without changing core behavior or item documents. The database remains a rebuildable index. Markdown, configuration, and accepted Git history remain authoritative. See [architecture](../architecture.md#database-selection-and-adapter-contract).

SQLite is the default local backend at `.tline/index.sqlite`. Backend independence is required from the first implementation; implementing multiple backends immediately is not. A connection string alone cannot make an unsupported database compatible.

## Required behavior

- Configuration selects the backend and its target. Omitting database configuration selects local SQLite. SQLite paths are relative to the Timeline root unless explicitly absolute. Do not silently fall back to SQLite when a configured target fails.
- Keep backend selection and non-secret settings in `timeline.yaml`. Server adapters must support an environment-variable reference for connection credentials; do not require secrets in tracked configuration or expose connection strings in diagnostics. Final field names and validation rules must be specified alongside the overall configuration schema before implementation.
- Select adapters at the composition root. Core use cases and CLI result models use domain values and a narrow index contract, never SQL dialects, connection strings, concrete database handles, or generated SQL types.
- All supported adapters preserve the same filtering, ordering, time precision, null/missing metadata semantics, history, pending-state reporting, error categories, and recovery behavior. Each adapter owns its SQL, migrations, transactions, pooling, and database-specific retry behavior.
- Changing the configured target selects a separate derived index. Normal `check` initializes/migrates that target as appropriate and reconstructs it from complete accepted history before processing pending changes. Reconstruction itself never re-registers items. The former database is left untouched; switching does not require copying its rows.
- Bind an index to the intended Timeline dataset and validate that binding before reuse. Specify dataset identity, target ownership, clone/worktree behavior, and namespace isolation before implementing target reuse. Refuse ambiguous or foreign populated targets rather than mixing records or deleting unrelated data. Shared-server namespace creation must be explicit and limited to Timeline-owned objects.
- Missing history, unsupported backends, invalid configuration, unavailable targets, and incompatible schemas produce actionable errors. No silent fallback, record invention, or supposedly complete stale results.
- Dry runs never create, migrate, or mutate a persistent target. Use temporary state from authoritative history where needed. Database selection does not remove Git/filesystem publication coordination or establish multi-machine collaboration.

## Acceptance criteria

- TL-DATABASE/AC-1: Default configuration uses local SQLite; explicit SQLite targets resolve correctly and remain isolated.
- TL-DATABASE/AC-2: Unsupported backends and invalid/unavailable targets fail clearly without fallback or source/history mutation; diagnostics redact credentials.
- TL-DATABASE/AC-3: Switching between two empty SQLite targets and reconstructing produces equivalent IDs, original entry times, saved type definitions, revisions, and query results. The original target remains unchanged.
- TL-DATABASE/AC-4: A populated target belonging to another dataset is rejected without mutation; ownership and identity are covered by explicit tests.
- TL-DATABASE/AC-5: Core/CLI packages have no concrete driver or generated SQL dependency. A reusable index contract test suite runs against the real SQLite adapter from the first implementation.
- TL-DATABASE/AC-6: Every additional backend must pass that same contract suite plus its own migration, concurrency, cancellation, failure, and executable E2E tests before being advertised as supported. Cross-backend reconstruction must preserve semantic results.
- TL-DATABASE/AC-7: Dry-run target selection/recovery leaves all persistent databases and authoritative sources unchanged.

## Verification and status

All criteria are unimplemented. SQLite-to-SQLite target switching establishes the initial recovery contract; it does not prove support for another engine. Add actual test paths and evidence here as implementation proceeds. PostgreSQL is a candidate for a later adapter, not a current dependency or commitment to a hosted service.
