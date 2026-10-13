---
type: goal
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
goal: List and filter events by reading files and Git directly, with no database
owner: Badhree Babu
status: open
---

## Why

People and agents need to read events in time order, find what is upcoming or past due, and open the document behind each result. A database would only be a copy of the files and Git that could drift out of date.

## How

- Define the core interface for reading recorded state and implement it with batched Git reads.
- Implement `tline list` with `--json`.
- [Measure read speed on generated timelines](../tasks/measure-read-speed.md)

## Done when

- `list` orders by event time then ID, filters by event time, entry time, type, timing rule, configured fields, and past due, and includes file paths.
- Pending changes are labeled separately, and a result that cannot account for every relevant file says it is incomplete.
- Invalid events never disappear silently from a result that claims to be complete.
- Measurements show whether a cache is needed, and at what size.

## Why it will work

With no database there is nothing to rebuild or keep in sync: a fresh clone answers immediately. Batched Git reads keep ordinary timelines fast, and the measurements warn us before users notice if it stops being fast.

## Design

### Reading recorded state

Each command scans the configured folders and reads the Timeline commits it needs from Git, in batches, for example with one `git cat-file --batch` process rather than a process per event. `core/` defines a small interface for recorded events, revisions, and configuration versions; the Git adapter implements it. A cache can be added behind this interface later without changing files, commands, or rules.

### Queries

`list` returns recorded events ordered by event time, then ID. Users can:

- read events in event-time order
- filter by event time, entry time, type, entry timing rule, configured fields such as status, and past due
- open the document behind each result
- find events with missing or invalid information

A view distinguishes recorded events from changes that have not been recorded yet. A query that cannot reconcile relevant files reports itself as incomplete; it never looks like an empty or complete result.
