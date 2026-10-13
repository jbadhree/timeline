---
type: goal
timeline_at: 2026-11-14T23:59:59Z
created_by: claude
goal: Record event edits that someone already committed with plain Git
owner: Badhree Babu
status: open
---

## Why

If an event edit is committed with plain Git, the working tree matches `HEAD`, so the Timeline commit that later records it has no diff for that file. History would then miss the change.

## How

- Make each Timeline commit name the events it records, for example in commit trailers listing event IDs.
- Compare recorded state with the previous Timeline commit, not the parent commit, when reading history.
- Compare `timeline.yaml` with the last recorded configuration, not `HEAD`, in recording step 7.

## Done when

- An edit committed by hand shows as pending, is recorded by the next check, and appears in `tline history`.
- A configuration change committed by hand is recorded by the next check.

## Why it will work

Trailers are part of the commit, so they travel with clones and survive any Git tool, and they make each Timeline commit self-describing without relying on diffs.
