# Timeline repository instructions

Start at the [primary goal](goals/primary-goal-build-timeline.md). It explains what Timeline is and links to every goal and task. This repository is a Timeline root: [`timeline.yaml`](timeline.yaml) enables the built-in goal and task types, and the design for each part of the system is in its goal's `## Design` section.

- Keep Timeline reusable. Consumer-specific types, domain logic, and agent runtimes belong to the applications using it. `goal` and `task` come from the built-in goals plugin; other types are configured or come from [plugins](goals/plugins.md). Examples are illustrative data.
- Keep checked-in documentation independent of particular consumers. Private handoff notes belong in ignored local files; they must not be required to understand or implement Timeline.
- Follow the Go core and `tline` CLI boundary in the [engineering foundation](goals/engineering-foundation.md). Do not build the TUI or other [later directions](goals/primary-goal-build-timeline.md#later-directions) unless asked, and do not make them harder.
- Preserve the distinction between drafts and recorded events, and between event time and original entry time. Markdown and `timeline.yaml` are the only stored state; there is no database and no version control requirement.
- Read events through the core interface described in [query the timeline](goals/query-the-timeline.md#design), so the SQLite record store can be added later without changing behavior.
- Preserve existing user work and unrelated staged changes. Keep dependencies modest and language simple.
- Add application code only when implementation is requested.
- Record each design decision in `decisions/` (marking any it replaces `superseded`) and link it from the goal's Why. Change behavior by editing the goal that owns it, or adding a goal or task linked from a parent's `## How`. Keep its `## Design` and `## Done when` in agreement; `## Done when` is what tests must demonstrate.
- Every behavior arrives with unit, integration, and end-to-end tests of the built `tline` binary. Run `make check` (and `make race` for concurrent code) before pushing; never weaken a check to make it pass.

For documentation changes, verify local links and whitespace. Report actual checks and limitations; do not claim tooling, passing tests, or benchmark results exist before they do.
