---
id: 01M4VDZ6EGH9Y2F9D20KKXJZAP
entered_at: 2026-10-13T20:06:26Z
entry_timing: future
type: goal
timeline_at: 2027-03-31T23:59:59Z
created_by: claude
goal: Keep every recorded event, with history and integrity, in a hosted store
owner: Badhree Babu
status: open
---

## How

- [Design the hosted record store](../tasks/record-store-design.md)
- [Send recorded events to the hosted store](../tasks/sync-command.md)
- [Show event history and detect edited stamps](../tasks/hosted-history.md)

## Done when

- Recorded events from several repos reach the hosted store, and `tline history <id>` shows every version.
- Hand-edited stamps are detected, and a customer can export everything as plain Markdown.

## Why it will work

Local files stay the truth and the hosted store only receives what `tline check` recorded, so customers are never locked in, and the store adds what files alone cannot: history, integrity, and one place for many repos.
