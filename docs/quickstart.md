# Using tline

This guide takes you from an empty folder to a checked timeline with notes, goals, and tasks. Agents follow the same steps; [agents.md](agents.md) is a short version written for them.

You need the `tline` binary; see [Install](../README.md#install). Every command below is run inside your timeline's folder.

## 1. Create a timeline

A timeline is a folder with a `timeline.yaml` that lists your event types. Create the folder, the file, and one folder per type:

```sh
mkdir my-work && cd my-work
mkdir notes goals tasks
```

`timeline.yaml`:

```yaml
version: 1
name: my-work
plugins: [goals]

types:
  note:
    folder: notes
    entry_timing: past
    fields:
      topic: { type: string, required: true }
      tags: { type: list, items: string }
  goal: { from: goals/goal, folder: goals }
  task: { from: goals/task, folder: tasks }
```

This defines three types:

- `note` is your own type: something that happened and that you write down afterwards (`entry_timing: past`), with a required `topic` and optional `tags`.
- `goal` and `task` come from the built-in goals plugin. Leave them, and `plugins: [goals]`, out if you do not want goals.

`tline` never creates or changes `timeline.yaml`; you own it.

## 2. Write an event

An event is a Markdown file in its type's folder: YAML frontmatter, then anything you like. `notes/kickoff.md`:

```markdown
---
type: note
timeline_at: 2026-10-12T15:00:00Z
created_by: Ada
topic: Project kickoff
tags: [planning]
---

We agreed to ship the first version by the end of the year.
```

Every event has these fields:

| Field | Meaning |
| --- | --- |
| `type` | The type, matching the folder |
| `timeline_at` | **Event time:** when it happened, or when it is due. RFC 3339 with an offset, such as `2026-10-12T15:00:00Z` |
| `created_by` | Who wrote it, a person or an agent |

File names are up to you.

## 3. Check and record it

Validate without changing anything:

```text
$ tline check --dry-run
Dry run: checked 1 event (1 draft, 0 recorded): all valid. Nothing was written.
```

Then record it:

```text
$ tline check
Checked 1 event (1 draft, 0 recorded): all valid. Stamped 1 draft:
  notes/kickoff.md  01M4V9X3BRRZG44GAJBEAVVWBF  past
Entered at 2026-10-13T18:55:23Z.
```

`tline check` adds three lines to the top of each new event, the **stamps**, and changes nothing else:

```yaml
---
id: 01M4V9X3BRRZG44GAJBEAVVWBF
entered_at: 2026-10-13T18:55:23Z
entry_timing: past
type: note
timeline_at: 2026-10-12T15:00:00Z
...
```

- `id` stays with the event even if you rename or move the file.
- `entered_at` is the **entry time**: when the event entered the timeline, taken from the clock. The note happened on the 12th and was recorded on the 13th; Timeline keeps both.
- `entry_timing` is the rule it was entered under.

A file without stamps is a **draft**; a file with them is **recorded**. Never write or change the stamps yourself. You can edit everything else in a recorded event, and running `tline check` again stamps only new drafts.

## 4. Timing rules

Each type's `entry_timing` says how its event time relates to its entry time:

| Rule | Meaning | Example |
| --- | --- | --- |
| `past` | It happened before it was recorded | News, a meeting note, a decision |
| `future` | It was recorded before it happens | A prediction, a goal, a task |
| `real_time` | It was recorded as it happened, within `max_delay` | A live observation; add `max_delay: 15m` |

A future event whose date has already passed cannot be recorded, because it was not written in advance; record plans before their dates. Once recorded, a future event whose date passes while it is still open is **past due**, which is shown, not an error.

## 5. Goals and tasks

With the goals plugin, a goal answers seven questions and a task five. `goals/first-release.md`:

```markdown
---
type: goal
timeline_at: 2026-12-31T23:59:59Z
created_by: Ada
goal: Ship the first version
owner: Ada
status: open
---

## Why

Customers are waiting for it. See the [kickoff](../notes/kickoff.md).

## How

- [Write the user guide](../tasks/user-guide.md)

## Done when

- The first version is downloadable.

## Why it will work

The code is done; only the guide is missing.
```

`tasks/user-guide.md`:

```markdown
---
type: task
timeline_at: 2026-12-15T23:59:59Z
created_by: Ada
task: Write the user guide
owner: Ada
status: open
---

## How

- Draft it, then ask two users to try it.
```

The rules `tline check` enforces:

- A goal needs `goal`, `owner`, `status` (`open`, `achieved`, `missed`, `dropped`), and the sections `## Why`, `## How`, `## Done when`, and `## Why it will work`.
- Every item in a goal's `## How` must link to another goal or a task, so all work is tracked.
- A task needs `task`, `owner`, `status` (`open`, `done`, `dropped`), and `## How` with plain steps. A task's steps must not link to goals or tasks.
- A goal or task that another goal links to may leave out `## Why`; it inherits its parent's. The task above has none.
- Goals must not link to each other in a cycle.
- Any link into a type folder must point to an existing event.

Other sections are allowed anywhere. To see your goals and tasks:

```text
$ tline goal tree
[open] Ship the first version  (2026-12-31)
└─ [open] Write the user guide  (2026-12-15)

$ tline task list
DUE         STATUS  FLAGS  OWNER  TASK                  GOAL                    PATH
2026-12-15  open           Ada    Write the user guide  goals/first-release.md  tasks/user-guide.md
```

`tline goal show first-release` shows a goal's seven answers with the status of every step, and `tline task show user-guide` a task's five. Lists take `--owner`, `--status`, and `--past-due`. Run `tline goal --help` for more.

## 6. When something is wrong

`tline check` lists every problem, and records nothing until all are fixed:

```text
$ tline check --dry-run
tasks/oops.md: section "Why": is required because no goal links to this task from its How section
tasks/oops.md: field "owner": is required
tasks/oops.md:6: field "onwer": is not a field of type "task"
tasks/oops.md:7: field "status": is "someday"; allowed values are open, done, dropped

Dry run: checked 4 events (3 drafts, 1 recorded): 1 with problems, 4 problems. Nothing was written.
```

Unknown fields are errors, so typos such as `onwer` are caught. Add `--json` to any command for machine-readable output.

| Exit code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | `tline` itself failed |
| 2 | Wrong command, flag, or path |
| 3 | Validation failed; nothing was written |
| 4 | Another check is running (`.tline/lock`), or a file changed during the check |
| 5 | A view's result is incomplete because some events are invalid |

## 7. More on types

```yaml
types:
  observation:
    folder: observations
    entry_timing: real_time
    max_delay: 15m
    status: { values: [open, closed], closed: [closed] }
    sections: [Source]
    fields:
      score: { type: number }
      checked: { type: bool }
      seen_at: { type: time }
      level: { type: string, values: [low, high] }
```

- Field types are `string`, `number`, `bool`, `time`, and `list` with `items`. `values` limits the allowed values, and `required: true` makes a field mandatory.
- `sections` lists `##` headings every event of the type must have.
- `status` lists allowed statuses and which are closed; it makes `status` required and lets open future events become past due.
- Goal and task types can take extra `fields` and `sections` too, and can be renamed: `objective: { from: goals/goal, folder: objectives }`.

`tline` finds the timeline by looking for `timeline.yaml` in the current folder and its parents, or use `tline --root path/to/my-work ...`. If you keep the timeline in Git, add `.tline/` to `.gitignore`; it only holds the lock.
