---
type: task
timeline_at: 2026-11-07T23:59:59Z
created_by: claude
task: Read event files into frontmatter, sections, list items, and links
owner: Badhree Babu
status: done
---

## How

- Split YAML frontmatter from the Markdown body and keep each value's YAML type and line.
- Collect `##` sections, their top-level list items, and the links in those items, ignoring fenced code.
