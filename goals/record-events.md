---
type: goal
timeline_at: 2026-11-21T23:59:59Z
created_by: claude
goal: Validate events and stamp new ones with tline check
owner: Badhree Babu
status: open
---

## Why

Humans and agents edit files with whatever tools they like, so Timeline needs one explicit step that turns a draft into a recorded event with a trustworthy entry time. `tline check` is the only thing that writes the entry stamps; every view depends on that.

## How

- Validate files with the rules from [events and types](events-and-types.md) and [timing](two-times-and-timing.md), plus root-wide checks such as duplicate IDs.
- For each valid draft, assign `id`, `entered_at`, and `entry_timing` and write them into the file.
- Take a writer lock and re-read each file before writing, so concurrent checks and editors never overwrite each other.

## Done when

- A draft can be dry-run, then checked, and comes out with its three stamps and every other byte unchanged.
- A Future draft whose time passes between dry run and check is rejected.
- Nothing is written unless every selected file is valid, and a check with no drafts writes nothing.
- A file edited between validation and writing is rejected, not overwritten.
- Two concurrent `tline check` processes never stamp the same draft twice.

## Why it will work

Stamping is a small, local edit: three lines at the top of the frontmatter, written atomically per file. A lock and a re-read before each write are enough to stop concurrent writers, and an interrupted check leaves every file either a complete draft or a complete recorded event, so rerunning it finishes the job.

## Design

### Workflow

An agent writes `predictions/example.md` without `id` or `entered_at` and runs `tline check predictions/example.md`. If validation fails, it fixes the file and runs the command again. A human follows the same workflow in any editor.

With no paths, `check` processes all configured folders. With paths, it processes only those files; root-wide rules such as duplicate IDs still consider every file. The help text must say clearly that a successful check edits draft files. Results say whether files were stamped, nothing changed, or the run was a dry run.

`tline check --dry-run` validates without changing any file. It does not reserve an entry time: a Future draft that passes a dry run can fail later if its time has passed by then.

### Drafts and recorded events

A **draft** is an event file without `id`, `entered_at`, and `entry_timing`. A **recorded event** has all three. `tline check` writes them together, at the top of the frontmatter, and leaves the rest of the file's bytes unchanged; a file with only some of them is invalid.

Only `tline check` writes the stamps, and it never changes them once written. Authors and agents must not write or edit them. Without a stored record of past checks, Timeline cannot detect a stamp that was typed in or edited by hand; it only checks that stamps are well formed and consistent with the event's timing. Detecting that is the job of the [record store](primary-goal-build-timeline.md#later-directions), a later direction.

Recorded events can be edited, moved, or removed like any file. Each check validates every event against the current type definition, and judges a recorded event's timing by its own `entered_at` and `entry_timing`, so a type's timing rule changing later does not reinterpret it. If a type gains a required field, existing events need it too, or the field should start as optional.

### Stamping a check

1. Take the writer lock for the root (`.tline/lock`), so only one check stamps at a time across processes.
2. Scan and validate the selected files and root-wide rules. Stop on any failure.
3. Read the entry time once from the clock, after validation succeeds.
4. For each draft: re-read it, confirm it still has the content that was validated, then write it with its stamps through a temporary file and rename. Stop at the first file that changed and report it; files already stamped are complete recorded events, and the rest stay drafts.

`created_by` stays the event's own attribution label; it is not an authenticated identity.
