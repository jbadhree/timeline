# Timeline: build or customize?

Research date: 5 October 2026. This follows the [product survey](research.md). It is a recommendation based on documentation and selected source inspection, not a benchmark, implementation estimate, or accepted technology decision.

## Recommendation

Build a small standalone Timeline core and use existing software to edit and display its Markdown files. For our current local scope, I expect that to be simpler to implement and maintain than forking a knowledge-management product.

Basic Memory remains a useful optional search and agent-access tool, but the parts it provides do not remove our most demanding work: registration, validation, accepted history, and recovery. The [architecture](architecture.md) describes the current plan for the standalone core and CLI.

The conclusion depends on scope. It assumes command-line access and an existing Markdown editor are sufficient initially. If we require a polished shared web application, live collaboration, accounts, and rich search immediately, reusing a product becomes much more attractive.

## What deeper inspection found

### Basic Memory

Basic Memory already supplies Markdown storage, indexing, search, and agent tools. Those are real savings. Its CLI documentation describes schema validation as a command over existing notes; `--strict` makes validation errors produce a failing exit code. That does not by itself establish a mandatory check before every write or external-file import. [CLI reference](https://docs.basicmemory.com/reference/cli-reference).

The inspected development source separates note preparation, file operations, content reconciliation, and publication of indexed relations. It already handles complexities around concurrent changes and persisted content. This is useful existing engineering, but a Timeline integration would need to determine where its acceptance rules run across those paths. This inspection is of the main branch, not a verified released integration API. [Entity service source](https://raw.githubusercontent.com/basicmachines-co/basic-memory/main/src/basic_memory/services/entity_service.py).

Its note schemas also describe observations and relations as well as frontmatter. Adapting our type definitions is therefore more than assuming that two YAML schemas are interchangeable. Its local documentation describes Git integration; that does not establish automatic preservation of our accepted registrations and type versions. [Schema system](https://docs.basicmemory.com/concepts/schema-system), [local user guide](https://docs.basicmemory.com/local/user-guide).

There are three possible approaches:

- A wrapper validates requests before calling Basic Memory. It still needs a separate process for direct file edits and any other write paths.
- A fork changes internal acceptance and sync behavior. It adds upstream upgrade and regression work.
- An independent Timeline core owns acceptance, while Basic Memory supplies optional search. This keeps the boundary clearer, but Basic Memory is no longer replacing the core.

My preference is the third approach if its search becomes useful. We do not currently need semantic search or MCP merely to let local coding agents run commands and read files.

### SilverBullet

SilverBullet is a stronger history candidate than the initial survey established. Its current documentation describes built-in Git-backed revisions, with managed, unmanaged, and disabled modes. Managed mode batches commits after editing pauses. Unmanaged mode reads existing history without committing. That would suit an editor whose accepted Timeline revisions are recorded by a separate core. [Revision documentation](https://docs.silverbullet.md/Revisions).

It also exposes events and APIs to read pages and revisions. These provide useful extension points. However, editor events and revision browsing do not establish type-specific admission checks for every external file change. Automatic snapshots also are not the same thing as explicitly accepted Timeline revisions. [Events](https://docs.silverbullet.md/Event), [space API](https://docs.silverbullet.md/API/space).

SilverBullet is attractive if its interface is a priority. I would integrate it through files and commands before modifying its internals. The exact installed release must be checked against the current documentation.

### Decap CMS

Decap supplies folder collections and configurable editing forms. Its documented hooks include actions before and after saving and publishing. Those hooks operate within the CMS workflow; they cannot intercept an unrelated process editing a Markdown file on disk. We would still need Timeline validation and reconciliation outside the editor. [Folder collections](https://decapcms.org/docs/collection-folder/), [CMS events](https://decapcms.org/docs/registering-events/).

It is a possible future authoring interface, rather than a replacement for the Timeline core.

## Comparing the actual work

These are qualitative engineering judgments, not measured delivery estimates.

| Work | Customize Basic Memory or SilverBullet | Small standalone core |
| --- | --- | --- |
| Markdown parsing and metadata | Reuse existing support; verify round-trip behavior | Use established parsing libraries |
| Types and timing rules | Add our rules and connect them to all relevant writes | Implement directly in one shared validator |
| Registration and accepted history | Map product history to our acceptance semantics, or add separate history handling | Implement explicit registration and Git revision handling |
| External edits and recovery | Integrate with the product's sync/index behavior | Scan files before queries and registration; optimize later |
| Search and views | Significant existing functionality | Start with metadata filters and SQLite queries |
| Human interface | Existing product interface | Use an existing Markdown editor |
| Agent interface | Existing tools, sometimes MCP | Small CLI with structured output |
| Maintenance | Our custom logic plus compatibility with upstream behavior | Our focused code and ordinary library dependencies |

The likely advantage of a standalone core comes from avoiding integration work, not from pretending history and concurrent writes are easy. Both options still need tests for the same correctness requirements.

## How to keep the core small

The initial implementation can have five responsibilities:

1. Read type definitions and validate frontmatter, required sections, and entry timing.
2. Register new items and accept updates, preserving original registration data and the applied type definition.
3. Preserve accepted revisions in Git, detect conflicts, and distinguish drafts from accepted records.
4. Reconcile files and rebuild a SQLite index from documents and accepted history.
5. List and filter items, returning their file paths and validation state.

Use existing YAML/schema libraries, Git, and SQLite rather than implementing those technologies. Keep Markdown bodies intact; ordinary links do not require a graph engine. A scan before relevant operations can avoid a background watcher initially, subject to testing performance with representative data.

The difficult part is step 3 together with step 4: a file save, Git commit, and index update are not one transaction. Interrupted operations, removed files, duplicate IDs, and concurrent writers must have explicit recovery behavior. A quick script that only appends dates would not satisfy the current design.

Use one local acceptance writer at a time and detect changes made since a caller read the item. People and agents can still prepare files directly; unaccepted edits must be visible and cannot silently count as registered events. This is a proposed implementation approach, not a restriction on who can use Timeline.

The current local trust model also matters: preserving history protects against accidental rewriting in normal operation. Local Git and timestamps are not independently tamper-proof against someone who controls the files, clock, and repository. We are not proposing an external notarization service.

## What would change the recommendation?

Prefer a product extension if we need its full interface now, or if a short integration experiment proves that a supported extension point can govern command writes and external edits while preserving our registration history across rebuilds.

For either approach, the first meaningful test is the same: register a future item, edit and move it, remove it, rebuild the index, and still recover its original entry time, type definition, and accepted content. Also reject a new file that tries to claim a past registration time. Passing these checks is more informative than demonstrating a timeline view.

No products were installed or tested for this follow-up. I would choose the standalone core as the working direction, with moderate confidence, and validate history/recovery in the first implementation slice. Exact effort depends on those results; this research does not justify a reliable number of days or lines of code.
