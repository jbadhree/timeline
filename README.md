# Timeline

Timeline gives agents, and the humans working with them, a sense of time. It records events on a timeline with two times for each: when the event happened (or is expected), and when it entered the timeline. News is learned after the fact, predictions are made in advance, and live observations are recorded as they happen. Future events, such as tasks and goals, can go past due.

Events are Markdown files with YAML metadata, edited by humans and agents with any tool. Users define their own event types and Past, Future, or Real-time entry rules. Goals and tasks are built in: a goal answers what, why, by when, how, how we know it is done, why it will work, and who owns it; a task is a simpler step whose result is obvious.

## Start here

This repository plans and designs Timeline as Timeline goals and tasks. Start at the [primary goal](goals/primary-goal-build-timeline.md), which explains the system and links to every goal and task.

- [`goals/`](goals/): what we are building and why, with the design for each part.
- [`tasks/`](tasks/): concrete steps whose result is obvious.
- [`decisions/`](decisions/): design decisions, current and superseded, linked from the goals they shaped.
- [`examples/`](examples/README.md): worked configuration and event files.

## Development

Requires Go (the `toolchain` line in `go.mod` downloads the pinned version automatically) and Make, on macOS or Linux. No credentials or global configuration are needed.

```sh
make check   # format, CGO-free build, lint, unit and end-to-end tests
make race    # tests with the race detector
make plan    # validate this repository's goals and tasks with tline
```

`make check` installs the pinned golangci-lint into `bin/` on first run. CI runs both targets on macOS and Linux for every push to `main` and every pull request.

## Status

`tline check --dry-run` validates a Timeline root: configuration, fields, timing, goal and task answers, links, and goal cycles. Stamping drafts, and the list, goal, and task views, are not built yet. The planned Go executable is `tline`, with `check`, `list`, `goal`, and `task` commands backed by a reusable core. Every command reads the files directly; there is no database and no version control requirement. A SQLite record store with `tline history` is a later direction.

Applications keep their own configuration, events, and domain logic and use an installed `tline` binary. Read [AGENTS.md](AGENTS.md) before making changes.
