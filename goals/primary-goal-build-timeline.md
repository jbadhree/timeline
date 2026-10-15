---
id: 01M4SXMJ8G03MH2GE2XCS2P27S
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: goal
timeline_at: 2026-12-31T23:59:59Z
created_by: claude
goal: Build Timeline, and plan and design it as Timeline goals and tasks
owner: Badhree Babu
status: open
---

# Primary goal: build Timeline

This is the entry point for the Timeline project, for humans and agents alike. It explains what we are building and links to every goal and task needed to build it. The design is not kept in separate documents: each goal holds the design for its part of the system in a `## Design` section.

## What we are building

Timeline gives agents, and the humans working with them, a sense of time. It records events on a timeline the way a person remembers them: what happened, what is happening, and what is expected, along with when each became known.

- **A timeline is a perspective.** Like one person's memory of one subject: work, personal life, or a market. Each holds its own events in its own order. Initially one Timeline root holds one timeline.
- **Events are Markdown files.** Each event is a readable document with YAML metadata, written with any editor by humans or agents. Users define their own event types: news, predictions, observations, or anything else worth remembering. See [events and types](events-and-types.md).
- **Every event has two times.** When it happened or is expected (event time), and when it entered the timeline (entry time). Each type has an entry timing rule: Past (learned after the fact), Future (entered in advance), or Real-time (entered as it happens). Future events can go past due. See [two times and timing rules](two-times-and-timing.md).
- **Recording is explicit.** `tline check` validates files and stamps each new event with its ID, entry time, and timing rule. Until then the file is a draft. See [recording](record-events.md).
- **History is plain copies too.** A type can keep every recorded version as a file under `history/`, and freeze fields and sections so a claim cannot be rewritten after it is recorded. See [event history](event-history.md).
- **Plain files, nothing else.** Markdown and `timeline.yaml` are the only stored state; there is no database and no version control requirement. Every command reads the files directly. See [querying](query-the-timeline.md).
- **Goals and tasks ship as a built-in plugin.** [Plugins](plugins.md) add types, rules, and commands on top of the core. A goal answers seven questions: what, why, by when, how, how we know it is done, why it will work, and who owns it. Every step in a goal's how is another goal or a task, so all work is tracked down to tasks: simpler steps whose result is obvious. See [goals and tasks](goals-and-tasks.md).

```text
Humans and agents edit Markdown files
                  |
          tline check (validate and record)
                  |
   Recorded Markdown files + timeline.yaml
                  |
   tline list / goal / task (read the files)
```

This repository is itself a Timeline root. [`timeline.yaml`](../timeline.yaml) enables goals and tasks, and the `goals/` and `tasks/` folders hold this plan. It also defines an ordinary `decision` type, with no special support in Timeline: `decisions/` records each design decision, `current` or `superseded`, and goals link to them from their Why. Once `tline` exists, `tline goal tree` shows this goal down to its tasks.

## Why

Agents act and predict without a reliable sense of time: they confuse what was expected with what was learned afterward, and they lose track of what is due or overdue. People have the same problem across long-running work. A timeline that keeps both times, and keeps predictions whether or not they came true, lets readers reconstruct what was known at any moment. That accumulated record of expectations, outcomes, and assessments helps future predictions and can later serve as data for evaluation or fine-tuning.

Planning Timeline as Timeline goals and tasks proves the goal format on real work before anyone else depends on it.

Decided in [plan with goals](../decisions/plan-with-goals.md).

## How

- [Engineering foundation](engineering-foundation.md)
- [Events and types](events-and-types.md)
- [Two times and timing rules](two-times-and-timing.md)
- [Record events with `tline check`](record-events.md)
- [Query the timeline](query-the-timeline.md)
- [Event history](event-history.md)
- [Plugins](plugins.md)
- [Goals and tasks](goals-and-tasks.md)
- [Write a quickstart for people and an instruction page for agents](../tasks/user-and-agent-guides.md)
- [Build release executables](../tasks/build-release-executables.md)
- [Review the code written so far](review-the-code.md)
- [Run the first-release demo](../tasks/run-first-release-demo.md)

## Done when

- Every goal linked above is achieved.
- The first-release demo is done on macOS and Linux with released `tline` executables.
- This repository's goals and tasks are recorded with `tline check`, and `tline goal tree` shows this goal down to its tasks.

## Why it will work

Recording and reading are the whole loop, and goals and tasks are views over the same recorded events, so each part builds on tested ground. Plain files keep the first version small and usable in any folder, with any editor and any sync tool. Using the goal format to build the tool exposes gaps in it early.

## Principles

- **Reusable.** Timeline imports no consumer application. Consumer-specific types, domain logic, and agent runtimes belong to the applications using it; `goal` and `task` come from the built-in goals plugin; every other type is configured or comes from a plugin.
- **Records, does not judge.** Timeline checks that events are complete and that their timing fits their type. It does not decide whether a claim is true or a prediction came true; a later event records that assessment.
- **Files are the truth.** Markdown and `timeline.yaml` are authoritative and readable without Timeline, so changing the software that reads them loses nothing. Keeping them in version control is up to the user; Timeline neither requires nor uses it.
- **A draft is not a recorded event, and event time is not entry time.** These two distinctions are never blurred.
- **Many writers, no silent overwrites.** Multiple writers must not silently overwrite one another's recorded changes.

## Command line

| Command | Purpose |
| --- | --- |
| `tline check [paths...]` | Validate files and, if all pass, stamp new drafts with their ID, entry time, and timing rule |
| `tline list` | List events with time and field filters, file paths, past-due flags, and drafts shown separately |
| `tline history <event>` | List or print the saved versions of an event whose type keeps history |
| `tline plugins`, `plugins trust`, `plugins untrust` | Review and approve a timeline's external plugins |
| `tline goal list`, `goal show`, `goal tree` | Goal views |
| `tline task list`, `task show` | Task views |

All commands support readable terminal output and `--json` for agents. JSON on stdout stays parseable; incidental logging goes to stderr. Diagnostics identify the file, field or section, and reason. Exit codes distinguish success, validation failure, conflict, incomplete results, usage errors, and operational failure. So far: 0 success, 1 operational failure, 2 usage error, 3 validation failure, 4 conflict (another check holds the lock, or a file changed during the check), 5 incomplete result (a view could not account for every relevant event), 6 external plugins not approved; all values are provisional until the first release.

## Not in scope

Domain work such as judging predictions, calculating accuracy, carrying out tasks, or maintaining a belief model belongs to the applications using Timeline. No database, hosted service, scheduler, or account system is required.

## Later directions

Intended, not planned yet. Current decisions should not make them harder.

- **Multiple timelines:** several perspectives in one place, and events that belong to more than one timeline.
- **Agent views:** "now" (recent, in progress, upcoming, past due), "as of a date" (only what had been entered by then), and a compact summary an agent can load as context.
- **Learning data export:** predictions paired with what was known when they were made and with later outcome or assessment events.
- **Link graph:** building and traversing the graph formed by all Markdown links.
- **Record store:** a database, local or hosted, that records every stamp and version across repos, imports the file history in `history/`, and detects stamps typed by hand before an event's first copy. It also makes reads fast for large timelines. Until it exists, entry times protect against honest mistakes only.
- **Integrity:** stronger tamper evidence, such as signed records or external timestamps.
- **Plugin marketplace:** sharing plugins between timelines and people, with an approval mechanism that covers everything a plugin runs and loads, not only its program file.
- **A TUI** and other front ends built on the same core.

## Changing the plan

To change what Timeline does, record a [decision](../decisions/), marking any decision it replaces as `superseded`, then edit the goal that owns that part of the system, or add a goal and link it from a parent's `## How`. Keep the `## Design` section and `## Done when` in agreement: `## Done when` is what tests must demonstrate.
