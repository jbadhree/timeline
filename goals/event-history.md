---
type: goal
timeline_at: 2026-11-28T23:59:59Z
created_by: claude
goal: Keep every event's identity and full history across edits, moves, and removals
owner: Badhree Babu
status: open
---

## Why

Corrections and removals must stay visible, and earlier versions must be recoverable, or the timeline cannot show what was known at a given moment. History also shows when a goal or task date slipped.

## How

- Implement `tline history <id>` by reading Timeline commits reachable from `HEAD`.
- Follow an event's `id` across moves, and read the `timeline.yaml` in force at each commit.
- Report missing history and conflicting IDs instead of inventing records.

## Done when

- History shows created, updated, moved, and removed revisions, oldest first, and any recorded version can be read back.
- The configuration in force for a revision is read from its commit, and changing `timeline.yaml` does not reinterpret recorded events.
- A full clone shows the same events, entry times, and history; a shallow clone reports missing history instead of re-registering events.
- Two branches that independently record the same `id` are reported as a conflict after merging.
- Reusing a removed event's path never merges it with the new event.

## Why it will work

Git keeps every commit, and the `id` in each file ties revisions together regardless of path, so history needs no separate store.

## Design

### History

`tline history <id>` lists the Timeline commits that touched an event, oldest first: commit, recorded time, Git author, kind (created, updated, moved, removed), and paths. Any recorded version can be read back from its commit. The type definition in force at that time is `timeline.yaml` at the same commit.

History is read from Timeline commits reachable from `HEAD`. A clone must include that history. Further policy for merged histories is later work.

### Data model

Built in memory by each command from the files and Git; nothing here is stored except through Markdown, `timeline.yaml`, and Timeline commits.

| Entity | Identity and fields | Source of truth |
| --- | --- | --- |
| Event | `id`; `type`, `entered_at`, `entry_timing`, `created_by`, and the Timeline commit that first recorded it | Frontmatter in the first Timeline commit |
| Revision | Timeline commit + event `id`; recorded time, Git author, kind, old and new path, content, `timeline_at`, field values | One Timeline commit's change to one event file |
| Goal link | Goal revision + position in `## How` or `## Done when`; target event `id` | A goal revision's body |
| Configuration version | Timeline commit + `timeline.yaml` content | `timeline.yaml` as of each Timeline commit |
| Pending change | Path, content hash, parsed draft values, diagnostics | The working tree; never stored |

Each revision belongs to one event and one Timeline commit, and one commit holds at most one revision per event. A removal revision has an old path and no content. Removed events keep all earlier revisions. Revisions are ordered by commit ancestry, not timestamps alone.
