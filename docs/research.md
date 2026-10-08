# Existing tools for Timeline

Research date: 5 October 2026. This is a comparison of official documentation and public source code, not a hands-on product test or an architecture decision. It covers the main relevant categories, not every product on the market.

The [build versus customize comparison](build-vs-customize.md) recommends a small standalone Timeline core with existing editors. The [architecture](architecture.md) describes the current implementation plan; this survey is supporting research.

## Finding

No verified product meets the complete [Timeline design](system-design.md) out of the box. Several could supply useful parts. Basic Memory and SilverBullet are relevant integration candidates; the follow-up comparison favors a small Timeline core with an existing editor.

The most distinctive requirement is the entry timing rule. Storing an event date and a creation date is common. Enforcing a type's Past, Future, or Real-time rule at registration, then preserving that original meaning through edits and recovery, is a separate capability. I did not verify that complete behavior in the shortlisted products.

## What counts as a match

- Configurable item types, required fields, statuses, and folders.
- One authoritative Markdown file with metadata per item, with ordinary links between files and sections.
- Separate event and registration times, with type-specific timing validation.
- Human and agent access, including edits made outside the application.
- Search, filters, ordering, and access to the underlying document.
- Preserved accepted history, including moved and removed items; rebuilding an index must not invent new registration dates.

Markdown export does not satisfy authoritative Markdown storage. Likewise, a warning or formula that flags a late prediction does not necessarily prevent it from being accepted as a valid prediction.

## Closest candidates

| Tool | Availability | What already fits | What needs work | Assessment |
| --- | --- | --- | --- | --- |
| **Basic Memory** | Open source, AGPL-3.0; free local use, optional paid cloud | Markdown shared by humans and agents, local indexing, links, search, CLI and MCP access | Timeline timing rules, accepted-history behavior, and integration with arbitrary type definitions need verification or development | Optional search and agent-access integration |
| **SilverBullet** | MIT server and CLI; desktop and hosting options have separate terms | Markdown folders, links, queries, programmable behavior, human editor | Type validation, registration rules, and accepted history | Strong alternative, especially for a customizable interface |
| **Obsidian + Bases** | Proprietary; free core, optional paid services | Local Markdown, YAML properties, links, filtered views | Shared validation and registration service, history, and a timeline display if wanted | Good human interface; does not replace the Timeline engine |
| **Decap CMS** | Open-source CMS | Folder collections of Markdown files, configurable fields, Git-backed editing | Timeline queries and timing/history rules | Good ready-made web forms for our files |
| **Front Matter CMS** | Open-source VS Code extension | Editing structured frontmatter, content fields and relationships | Standalone service, timeline queries, timing rules | Good editor for people already using VS Code |
| **Markwhen / Meridiem** | Public source components and commercial offerings; check each component's license | Text-based timelines, event properties, visualizations, editor integrations | Its timeline documents are not our one-item-per-Markdown-file store; timing rules need custom code | Good display candidate |
| **Aeon Timeline** | Paid desktop application | Custom item types, fields, relationships, timeline and other views | Authoritative per-item Markdown and supported headless agent writes were not established | Closest finished timeline application, but a weaker architectural fit |

Sources: [Basic Memory repository](https://github.com/basicmachines-co/basic-memory), [SilverBullet](https://silverbullet.md/), [Obsidian Bases](https://obsidian.md/help/bases), [Obsidian pricing](https://obsidian.md/pricing), [Decap folder collections](https://decapcms.org/docs/collection-folder/), [Front Matter fields](https://frontmatter.codes/docs/content-creation/fields/), [Front Matter source](https://github.com/estruyf/vscode-front-matter), [Markwhen](https://markwhen.com/), [Meridiem](https://docs.markwhen.com/meridiem/), [Aeon customization](https://www.aeontimeline.com/features/structure-templates-and-customization), and [Aeon pricing](https://www.aeontimeline.com/pricing).

### Basic Memory deserves a closer look

Its file-based human/agent workflow is unusually close to our intended storage model. Its schema system can describe task-like note types, required metadata, facts, and relationships. Validation defaults to warnings; the documentation also describes a strict mode. We still need to check exactly which write paths enforce it. [Schema documentation](https://docs.basicmemory.com/concepts/schema-system).

The current development source also distinguishes authored semantic times, such as occurred and due times, from record timestamps. That is promising, but I have not confirmed which released version contains this code. It is not proof of our entry timing rules or of registration history surviving a rebuild. [Temporal source on the main branch](https://raw.githubusercontent.com/basicmachines-co/basic-memory/main/src/basic_memory/temporal.py).

Before choosing it, test ordinary Markdown links, custom metadata round trips, external file edits, stable identity on moves, and accepted historical snapshots. Its own knowledge conventions must not become a requirement for every Timeline application. A custom integration is plausible; its size is not yet established.

### Editors and displays can be reused independently

SilverBullet's programmable queries and CLI make it worth testing as both an editor and an integration surface. Obsidian Bases provides structured views over Markdown properties. Neither should be assumed to enforce our registration rules just because it can show the relevant fields. [SilverBullet CLI](https://silverbullet.md/cli), [Obsidian Bases syntax](https://obsidian.md/help/bases/syntax).

Decap's folder collections closely match our type-folder arrangement. Front Matter exposes metadata editing inside VS Code. These could save us from building forms, but editor-level checks alone cannot govern direct file writes by agents. [Decap configuration](https://decapcms.org/docs/configuration-options/), [Front Matter fields](https://frontmatter.codes/docs/content-creation/fields/).

A Markwhen view could be generated from accepted Timeline records. That would let us reuse its display while keeping Markdown items authoritative. This is a proposed integration, not a verified existing connector. [Markwhen properties](https://docs.markwhen.com/syntax/properties), [timeline visualization](https://docs.markwhen.com/visualizations/timeline/).

## Alternatives with a different storage model

These become more attractive if we relax the requirement that Markdown files are authoritative. Keeping that requirement would mean building and maintaining an adapter, with clear ownership of every edit.

| Tool | Availability | Relevant capabilities | Main mismatch |
| --- | --- | --- | --- |
| **Baserow** | Open-source core, paid plans and hosting | Typed fields, relationships, event dates and automatically recorded creation dates; timeline views on relevant plans | Records live in its database; timing enforcement and Markdown integration need additional work |
| **Airtable** | Commercial SaaS with free and paid plans | Configurable tables and fields, linked records, automations, paid timeline view | Database records are authoritative; its timeline view does not represent timestamp-level detail |
| **Notion** | Commercial SaaS with free and paid plans | Pages, database properties, relationships and timeline views | Markdown is an export format, not the live source of database state |
| **Capacities** | Commercial app with free and paid plans | Typed knowledge objects, properties and date-based organization | Object workspace rather than our canonical folder of item files |
| **Tana** | Commercial service | Supertags provide structured node types with fields and queries | Node-based workspace rather than per-item Markdown storage |
| **Anytype** | Source-available local-first app | Connected objects and a local API | Its internal object store is a different model; local-first does not mean Markdown-authoritative |
| **Directus** | Source-available/commercial platform | Configurable data model, APIs and content versioning | Database-oriented foundation; using it would introduce a different backend and still require timing logic |

Sources: [Baserow date fields](https://baserow.io/user-docs/date-and-time-fields), [Baserow plans](https://baserow.io/pricing), [Airtable timeline documentation](https://support.airtable.com/articles/1069623552-timeline-view-overview), [Notion timelines](https://www.notion.com/help/timelines), [Notion exports](https://www.notion.com/help/export-your-content), [Capacities properties](https://docs.capacities.io/reference/properties), [Capacities plans](https://capacities.io/pricing), [Tana supertags](https://outliner.tana.inc/learn/features/supertags), [Anytype source and license](https://github.com/anyproto/anytype-ts), [Anytype developer documentation](https://developers.anytype.io/), [Directus content versions](https://docs.directus.io/reference/system/versions), and [Directus plans and licensing terms](https://directus.com/pricing).

Baserow is the strongest candidate in this group for a configurable operational tracker. That is my assessment of the documented features, not a recommendation to change our storage decision. For a quick demonstration, it could model the types and two dates; a demonstration would not establish reliable registration enforcement.

## Other useful foundations

- **TinaCMS:** Git-oriented content collections are another possible authoring layer. We would still need the Timeline engine. [Collection documentation](https://tina.io/docs/reference/collections).
- **Logseq:** relevant for linked knowledge and agent-readable notes, but its file and database directions must be distinguished before choosing an edition. Do not assume every edition preserves Markdown as the source of truth. [Repository](https://github.com/logseq/logseq).
- **XTDB:** its valid-time and system-time history addresses the related question of when something is true versus when the system knew it. Those concepts are useful, but are not identical to an event timestamp plus a type's entry rule. It is a database foundation, not a finished Markdown Timeline application. [Time in XTDB](https://docs.xtdb.com/about/time-in-xtdb.html).
- **vis-timeline:** a reusable visual timeline component if we build an interface. It does not supply storage, item definitions, or registration semantics. [Repository](https://github.com/visjs/vis-timeline).

## Recommendation and evaluation gates

Build the focused Timeline core described in the architecture and reuse an existing editor. If evaluating a product integration later, keep Timeline's rules independent from the editor or agent and use the following checks.

A small evaluation should answer these concrete questions:

1. Can we add a new item type by configuration and store its items in its own Markdown folder?
2. Can we accept yesterday's News, reject a newly entered prediction about yesterday, and distinguish a fresh observation from a stale one?
3. Can a manually edited `entered_at` field falsely claim that a prediction was registered earlier? It must not.
4. Can both an agent and a human edit the files without silent overwrites, lost fields, or stale results presented as current?
5. Does a Future item retain its original classification after its date passes?
6. Can we move, amend, or remove an item, then rebuild the index and recover the same registration time and past content?
7. Can we change a type definition without silently reinterpreting earlier accepted items?

If a candidate passes storage and access checks but not timing and history checks, compare extending it with a focused Timeline implementation that reuses its editor or display. Do not assume a wrapper will be small until all write paths and recovery behavior are understood.

No product was installed or benchmarked for this research. Release compatibility, performance, concurrent editing, and complete history recovery remain untested. Paid plans and component licenses should be checked again when selecting a version or buying a service.
