---
type: goal
timeline_at: 2026-11-21T23:59:59Z
created_by: claude
goal: Validate and record events with tline check
owner: Badhree Babu
status: open
---

## Why

Humans and agents edit files with whatever tools they like, so Timeline needs one explicit step that turns an edit into a recorded fact with a trustworthy entry time. Recording is the only writer; every view depends on its guarantees.

## How

- Find pending changes by comparing the working tree with recorded state.
- Validate them with the rules from [events and types](events-and-types.md) and [timing](two-times-and-timing.md), plus root-wide checks such as duplicate IDs.
- Assign `id`, `entered_at`, and `entry_timing`, and record everything in one Timeline commit.
- [Recover cleanly when a check stops between writing IDs and committing](check-crash-recovery.md)
- [Record event edits that someone already committed with plain Git](hand-committed-edits.md)

## Done when

- A draft can be dry-run, recorded, edited, moved, and removed, each with the expected Timeline commit and history.
- A Future event whose time passes between dry run and recording is rejected.
- A file that claims an `id`, `entered_at`, or `entry_timing` that history does not support is rejected.
- Selected changes are recorded all together or not at all.
- Unrelated staged and unstaged changes survive a check, a check with nothing pending creates no commit, and two concurrent `tline` processes never both record.

## Why it will work

Git already supplies durable history and conditional ref updates; Timeline adds rules on top rather than building its own store. A writer lock, a re-read before committing, and a conditional branch update together stop concurrent writers and editors from overwriting each other silently.

## Design

### Workflow

An agent writes `predictions/example.md` without `id` or `entered_at` and runs `tline check predictions/example.md`. If validation fails, it fixes the file and runs the command again. A human follows the same workflow in any editor.

With no paths, `check` processes all configured folders. With paths, it processes only those changes; a move must include both paths. Root-wide dependencies such as duplicate IDs are still checked. There are no separate init or record commands. The help text must say clearly that a successful check edits event files and creates a Git commit. Results say whether changes were recorded, nothing changed, or the run was a dry run.

`tline check --dry-run` validates without changing files or Git. It does not reserve an entry time: a Future event that passes a dry run can fail later if its time has passed by then.

### Recorded versus pending

A **Timeline commit** is a Git commit created by `tline check`, identified by a `Timeline-Check: 1` commit trailer. An event is recorded at its content in the latest Timeline commit that touched it. Any difference in the working tree is pending: a new file, an edit, a move, or a removal. Ordinary Git commits record nothing.

A draft omits `id`, `entered_at`, and `entry_timing`. A successful check inserts all three at the top of the frontmatter and leaves the rest of the file's bytes unchanged. Draft metadata cannot establish registration: a file that claims an `id`, `entered_at`, or `entry_timing` that history does not support fails validation, and so does an update that changes any of them.

### Recording a check

1. Take the writer lock for the root (`.tline/lock`), so only one check records at a time across processes.
2. Read recorded state from Git.
3. Scan the selected files and compare them with recorded state to find pending changes.
4. Validate the pending changes and root-wide dependencies. Stop on any failure.
5. For new events, assign `id` and `entered_at`, copy the type's `entry_timing`, and write them into the file.
6. Re-read each selected file and confirm it still has the content that was validated. Reject the run if a file changed in the meantime.
7. Create one Timeline commit containing only the selected event files and `timeline.yaml` if it differs from `HEAD`. Build it with a temporary Git index, update the branch only if `HEAD` is unchanged, then update the user's Git index entries for the recorded paths only. Unrelated staged and unstaged changes are left alone.

The commit's author and committer come from the user's Git configuration. `created_by` stays the event's own attribution label. Neither is an authenticated identity. A check with no pending changes creates no commit; a configuration-only change is recorded like any other.
