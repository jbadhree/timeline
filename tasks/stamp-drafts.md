---
type: task
timeline_at: 2026-10-23T23:59:59Z
created_by: claude
task: Stamp valid drafts with id, entered_at, and entry_timing
owner: Badhree Babu
status: open
---

## How

- After validation, generate a ULID per draft and read `entered_at` once from the clock.
- Insert the three fields at the top of the frontmatter, leaving every other byte unchanged.
- Write each file through a temporary file and rename, and report what was stamped.
