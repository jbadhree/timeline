---
id: 01M4SXMJ8GWAZP4VTY5M13GWJ2
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: goal
timeline_at: 2026-12-05T23:59:59Z
created_by: claude
goal: List and filter events by reading the files directly, with no database
owner: Badhree Babu
status: open
---

## Why

People and agents need to read events in time order, find what is upcoming or past due, and open the document behind each result. A database would only be a copy of the files that could drift out of date.

Decided in [no database in the first release](../decisions/no-database-for-v1.md).

## How

- [Define the core interface for reading events](../tasks/read-events-interface.md)
- [Implement tline list with filters and --json](../tasks/tline-list.md)
- [Measure read speed on generated timelines](../tasks/measure-read-speed.md)

## Done when

- `list` orders by event time then ID, filters by event time, entry time, type, timing rule, configured fields, and past due, and includes file paths.
- Drafts are labeled separately, and a result that cannot account for every relevant file says it is incomplete.
- Invalid events never disappear silently from a result that claims to be complete.
- Measurements show whether a cache is needed, and at what size.

## Why it will work

With no database there is nothing to rebuild or keep in sync: any copy of the folder answers immediately. Reading a few thousand small files is fast, and the measurements warn us before users notice if it stops being fast.

## Design

### Reading events

Each command scans the configured folders and parses the event files. `core/` defines a small interface for reading events; the file adapter implements it. The [record store](primary-goal-build-timeline.md#later-directions) can be added behind this interface later without changing files, commands, or rules.

### Queries

`list` returns recorded events ordered by event time, then ID. Users can:

- read events in event-time order
- filter by event time, entry time, type, entry timing rule, configured fields such as status, and past due
- open the document behind each result
- find events with missing or invalid information

A view distinguishes recorded events from drafts. A query that cannot reconcile relevant files reports itself as incomplete; it never looks like an empty or complete result.
