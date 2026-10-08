# Requirements index

Requirements are the implementation input. Start from [project context](../README.md), edit requirements on a branch, then give agents the IDs to implement. The current product contracts remain in the existing design documents; this index assigns stable IDs without copying them into another specification.

## Existing product contracts

All rows are documented and unimplemented. Detailed formats and failure protocols must be specified before their implementation starts.

| ID | Canonical requirement | Required verification focus |
| --- | --- | --- |
| TL-ROOT | [Root and configuration](../architecture.md#root-discovery-and-configuration) | Nearest/explicit root, invalid configuration, path containment |
| TL-TYPES | [Items and types](../system-design.md#items-and-types), [format](../architecture.md#markdown-format) | Configurable schemas, required fields/sections, invalid input |
| TL-TIME | [Timing validation](../architecture.md#timing-validation) | Boundary precision, fake entry times, original classification |
| TL-CHECK | [CLI workflow](../architecture.md#cli-workflow) | Selected-set atomicity, dry run, no-op, dependency validation |
| TL-HISTORY | [Accepted history](../architecture.md#writes-and-accepted-history) | Content/config preservation, identity, moves/removals, actors |
| TL-CONFLICT | [Accepted history](../architecture.md#writes-and-accepted-history) | Concurrent writers, stale hashes, unrelated staged files |
| TL-RECOVERY | [Index recovery](../architecture.md#indexing-and-recovery) | Full clone, missing evidence, interruptions, no re-registration |
| TL-QUERY | [Queries](../architecture.md#queries-and-links), [reconciliation](../architecture.md#indexing-and-recovery) | Recorded vs pending, complete vs incomplete, deterministic paths |
| TL-CLI | [CLI workflow](../architecture.md#cli-workflow) | Binary commands, JSON, exit codes, stderr/stdout contract |
| TL-BOUNDARY | [Independence](../system-design.md#independence), [core](../architecture.md#core-and-future-interfaces) | Consumer-neutral core, no consumer application imports or hard-coded types |
| TL-MODEL | [Recorded data model](TL-MODEL.md), [entities and relationships](../architecture.md#data-model) | Stable identities, revision relationships, tombstones, saved definitions, index replay |
| TL-DATABASE | [Configurable index database](TL-DATABASE.md) | Default SQLite, target switching, adapter parity, isolation and recovery |
| TL-FOUNDATION | [Engineering foundation](foundation.md) | Tooling, tests, migration infrastructure, setup, context |
| TL-PERF | [Performance requirements](../performance.md) | Reproducible workloads, bounded resources, measured budgets |

For a change, add a focused `TL-<NAME>.md` using [the template](template.md), or refine the existing canonical section. Register new IDs here and link refinements from the canonical section. Never maintain conflicting copies. Give acceptance criteria stable suffixes such as `TL-CHECK/AC-1`. Link actual test files/symbols when they exist; do not invent evidence paths as if they already passed.

Record the implementation status and evidence beside each new requirement. Existing rows remain unimplemented until a change adds a concrete criterion-to-test mapping. Changes to requirements and tests must be reviewed together so tests cannot silently redefine the requested behavior.
