---
id: 01M4WFZBQRW8APPX4725DMRM4B
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review the plugin facts and history comparison in the core
owner: Badhree Babu
status: done
---

## How

- Read `core/plugin.go` and `core/history.go`.
- Check that a plugin cannot judge another plugin's events, and that list item indexes and past due are right.
- Check frozen field and section comparison, and that diagnostic text is valid UTF-8 when it shortens long text.

## Findings

- **Fixed.** Shortened section text in frozen-section messages could split a multi-byte character and produce invalid UTF-8. Test: `TestExcerptKeepsCharactersWhole`.
- **Fixed.** When an event of a history type was removed, a new event was recorded at its path, and the check stopped before saving the new event's v1, every later check reported the new event as the old one with a changed `id`, so the missing copy could never be saved. A new event is now recognized because it was entered no earlier than the old history's last copy and with a different `entered_at` from the old event's v1; a hand-changed `id` keeps the old `entered_at` and is still rejected. Tests: `TestNewEventAtARemovedEventsPath` (core), `TestNewEventAtARemovedPathRepairsItsMissingCopy` (CLI), and `TestNewEventAtARemovedPathAfterAStoppedCheck` (end to end).
- **Acceptable.** A plugin's diagnostic for a file outside its types is replaced by one that says so and still fails that file, as designed. List item indexes and past due are correct.
