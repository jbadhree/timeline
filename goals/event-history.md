---
id: 01M4VWWXJ0WQB8KNC9E3EX1HRN
entered_at: 2026-10-14T00:27:20Z
entry_timing: future
type: goal
timeline_at: 2026-10-31T23:59:59Z
created_by: claude
goal: Keep earlier versions of recorded events as plain file copies, and freeze what must not change
owner: Badhree Babu
status: open
---

## Why

A timeline must show what was known when, so readers need earlier versions of recorded events, and a prediction's claim must not be quietly rewritten after its outcome is known. Decided in [keep event history as plain file copies, per type](../decisions/history-as-file-copies.md).

## How

- [Configure history and frozen fields per type](../tasks/history-config.md)
- [Save history copies in tline check](../tasks/history-copies.md)
- [Reject changes to frozen fields, frozen sections, and stamps](../tasks/frozen-rules.md)
- [Add the tline history command](../tasks/history-command.md)
- [Document history and add a frozen prediction to the markets example](../tasks/history-docs.md)

## Done when

- Stamping a draft of a history type saves v1; changing it and checking saves v2; checking an unchanged event saves nothing; a dry run writes nothing.
- A change from v1 to a frozen field or section, or to any stamp, is rejected with the original value; other changes are accepted.
- Enabling history on a type with recorded events saves each as v1, noted as copied when history was enabled, and a copy missing after an interrupted check is saved by the next check.
- Moved events keep their history by ID, and removed events keep their copies.
- `tline history` lists versions with their saved time and changed fields and sections, as text and JSON, and prints any version.

## Why it will work

Copies are exact files, so they need no database, no version control, and no special tool to read, and the event file stays the single latest version. `tline check` already holds the writer lock and reads every event, so saving a copy when content changes costs one comparison per event.

## Design

### Configuration

```yaml
types:
  prediction:
    folder: predictions
    entry_timing: future
    history: true
    frozen: [pred, ticker, timeline_at, "## Reasoning"]
```

- `history: true` keeps copies of every recorded version of the type's events. It is allowed on configured and plugin types.
- `frozen` lists fields, by name, and sections, as `"## Heading"`, that must keep their value from v1. It requires `history: true`. Fields must be the type's own or shared fields other than the stamps.
- History types also reject any change to `id`, `entered_at`, or `entry_timing` from v1.

### Storage

```text
my-timeline/
  timeline.yaml
  predictions/nvda.md                         the latest version
  history/predictions/<event id>/v1.md        exact copies, oldest first
  history/predictions/<event id>/v2.md
  history/predictions/<event id>/versions.log
```

- `history/` is at the root, outside every type folder, so copies are never read as events; no type folder may be `history` or inside it, in any letter case, since most macOS disks ignore case. It is not under `.tline/`, which holds only local runtime state: users keep `history/` with their timeline, for example by committing it, and never edit it.
- Each `vN.md` is a byte-for-byte copy of the event file when it was saved. Copies are never overwritten: a new version always gets the next number.
- `versions.log` has one line per copy: `vN <saved at, UTC> <reason> <event path>`. Reasons are `recorded` (v1 saved when the draft was stamped), `changed` (a later version), `history-enabled` (v1 of an event recorded before its type kept history), and `late` (v1 saved by a later check, because the check that recorded the event stopped before saving it). A copy without a log line shows an unknown time.

### Saving copies

`tline check`, outside a dry run and only when every selected file is valid, under the writer lock:

1. Stamps drafts as before, then saves each stamped history-type draft as v1 with reason `recorded` and its `entered_at` as the saved time.
2. For every other selected recorded event of a history type, compares the file with its latest copy. With no copies, it saves v1: `history-enabled` if the type's history folder did not exist yet, otherwise `late`. With a different latest copy, it saves the next version as `changed`. An identical file gets no copy.
3. Re-reads each file before copying and stops with a conflict if it changed since validation.
4. Writes each copy, then its log line, then reports success. If a check stops between rewriting a file and saving its copy, the next check saves the missing copy.

### Frozen rules

During every check, including dry runs, a recorded event of a history type is compared with its v1, if there is one:

- each `frozen` field must have the same YAML value as in v1; a removed or changed field is reported with v1's value;
- each frozen section must have the same text as in v1, ignoring surrounding blank lines; a changed or removed section is reported with the start of v1's text;
- `id`, `entered_at`, and `entry_timing` must equal v1's.

A recorded file whose `id` has no copies, at a path whose history belongs to a different ID that no current file has, is reported as having a changed `id`, unless it is a new event recorded at that path after the other event was removed: its `entered_at` is no earlier than the time that history's latest copy was saved, and differs from the other event's v1, which a changed `id` would keep. Its missing v1 is then saved as `late`.

### Moves and removals

History is keyed by ID, so moving or renaming an event file keeps its history, and the next copy logs the new path. Removing an event file leaves its copies; `tline history <id>` still lists them and says the event was removed.

### The history command

```text
tline history <id|path|name> [--json] [--version N]
```

The argument is an event ID, a path, or a file name without `.md`. The command lists each version with its saved time, reason, and which fields and sections changed from the version before, and says when the current file differs from the latest copy (run `tline check` to save it) or the event was removed. `--version N` prints that copy exactly.
