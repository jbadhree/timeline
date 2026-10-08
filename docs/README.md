# Start here

This is the single context entry point for humans and agents. `cmd/tline/main.go` will be the executable entry point; this file is the equivalent for understanding the project. Code and development automation do not exist yet. Proposed paths and commands below become real in the first implementation change.

## Read in this order

1. [System design](system-design.md): what Timeline does and its product boundaries.
2. [Requirements index](requirements/README.md): stable requirement IDs, acceptance criteria, and implementation status.
3. [Architecture](architecture.md): ownership, dependency direction, the [data model](architecture.md#data-model), storage, and recovery.
4. [Engineering workflow](engineering.md): implementation philosophy, context conventions, and parallel agent work.
5. [Testing](testing.md) and [performance](performance.md): required evidence for correctness and efficiency.
6. [Development setup](development.md): local, remote, and CI environment contract.
7. [Tooling options](tooling.md): researched recommendations and alternatives, with sources.
8. [Examples](../examples/README.md): worked Timeline roots with draft configuration and event files.

For a specific task, start here, then read its requirement and the affected package documentation. Read adjacent contracts before changing an interface; do not load every research note for routine work.

## Where information belongs

| Question | Canonical location |
| --- | --- |
| What must the product do? | System design; detailed new behavior in a linked requirement |
| Which requirements are being implemented? | Requirements index and individual acceptance criteria |
| How are components connected? | Architecture |
| How do we work and verify changes? | Engineering, testing, performance, and development guides |
| Why was a tool selected? | Tooling options, with selection status and evidence |
| What does a package own? | Its `doc.go`; a package README only for substantial operational detail |
| What does a symbol promise? | Its Go documentation comment |
| Why is this non-obvious line necessary? | A nearby comment referencing the invariant or requirement |
| What is unfinished in this branch? | The relevant requirement's status and PR handoff |

The system design and architecture remain authoritative. Requirement details must agree with them; update both in the same change when behavior or architecture changes. Research is supporting evidence, not permission to change requirements. Avoid copied specifications, generated summaries treated as truth, and a second architecture document.

The earlier [product survey](research.md) and [build comparison](build-vs-customize.md) explain the standalone-core decision. They are optional background.
