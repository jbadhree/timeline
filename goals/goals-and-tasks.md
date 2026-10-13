---
id: 01M4SXMJ8GT6A54VTYG452Y8XH
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: goal
timeline_at: 2026-12-19T23:59:59Z
created_by: claude
goal: Give goals and tasks their rules and views, as the built-in goals plugin
owner: Badhree Babu
status: achieved
---

## Why

Much of what agents and people put on a timeline is intended work. A goal that states what, why, by when, how, how we will know, why it will work, and who owns it can be checked for completeness, broken into smaller goals and tasks, and followed until it is achieved or goes past due. This plan is the first user.

Decided in [goal and task ship as a built-in plugin](../decisions/goals-and-tasks-as-plugin.md) and [every goal step must link](../decisions/goal-steps-must-link.md). The rules and views live in the goals plugin described in [plugins](plugins.md).

## How

- [Validate goal and task answers, links, and cycles](../tasks/validate-goals-and-tasks.md)
- [Build the tree of goals and tasks from How links](../tasks/goal-tree.md)
- [Implement tline goal list, show, and tree](../tasks/goal-views.md)
- [Implement tline task list and show](../tasks/task-views.md)

## Done when

- A goal or task missing any required answer is rejected, except a why inherited from a parent goal.
- A goal step that does not link to a goal or task, a goal link that does not resolve, a cycle of goals, and a task linking to a goal or task are each rejected.
- Extra fields and sections configured on the built-in types are validated like any other.
- `goal show` presents all seven answers and `task show` all five; `goal tree` renders this repository from the [primary goal](primary-goal-build-timeline.md) down to its tasks.
- Broken links and missing answers in unchanged goals are reported by the views rather than hidden.

## Why it will work

Goals and tasks are ordinary events with a fixed shape, so their views read the same files as `list`, not a second store. Fixed section names make every goal readable the same way by people, agents, and the CLI.

## Design

### The two types

Both types come from the built-in goals [plugin](plugins.md) in `internal/plugins/goals`. The core knows nothing about them: it checks their fields, statuses, and timing from the plugin's type templates, and the plugin adds every rule below from the facts the core gives it.

A goal is something we intend to achieve by a date. A task is work whose result is obvious: we know when it is done, and it needs no explanation. "List the files in a folder: run `ls`" is a task, not a goal. Both use the Future rule.

| Question | Location | Goal | Task |
| --- | --- | --- | --- |
| What | frontmatter `goal` or `task` | Required, non-empty string | Required, non-empty string |
| Why | `## Why` section | Required unless a goal links to it from `## How` | Same as goal |
| By when | `timeline_at` | Required | Required |
| How | `## How` section | Required list of one or more steps; every step must link to a goal or task | Required list of one or more plain instructions; must not link to a goal or task |
| Done when | `## Done when` section | Required list; items can link to an outcome validation event | Not used |
| Why it will work | `## Why it will work` section | Required, non-empty | Not used |
| Who owns it | frontmatter `owner` | Required, non-empty string; separate from `created_by` | Same as goal |
| Status | frontmatter `status` | `open`; closed `achieved`, `missed`, `dropped` | `open`; closed `done`, `dropped` |

A root enables each type in `timeline.yaml`, for example `goal: { folder: goals }`, and may add its own `fields` and `sections`. Goals and tasks may contain other sections, such as the `## Design` sections in this plan. Sections are found by exact `##` heading text. A task with `## Done when` or `## Why it will work` is not an error, but those sections are not treated as answers; if a task needs them, it is probably a goal.

A task:

```markdown
---
type: task
timeline_at: 2026-12-20T17:00:00Z
created_by: agent
task: List the files in the release folder
owner: Ada
status: open
---

## How
- Run `ls release/`.
```

The goals in this folder are complete goal examples.

### Links and the tree

Every step in a goal's `## How` must link to a goal or task, so all work under a goal has an owner, a date, and a status, and can go past due. A step may have text around its link, or link to several goals and tasks; plain-text steps, and steps that link only to URLs, documents, or other kinds of events, are rejected. A linked goal or task is a **child** of the goal; a task may be the child of more than one goal. Child links must not form a cycle.

Links in `## Why`, `## Done when`, and other sections are optional; a goal's Why can link to the [decisions](../decisions/plan-with-goals.md) behind it, or to any other event. When a goal is checked, every link in `## How` or `## Done when` that points into a configured folder must resolve to an event file; links to URLs, anchors, and files outside the folders are allowed there. A task's `## How` may link to documents, URLs, or other events, but not to a goal or task, so tasks are always leaves.

Views resolve links to event `id`s, so a link still identifies the event the goal meant after the linked file moves. A stale path in an unchanged goal is reported as broken, not silently dropped.

A child may omit `## Why`; the views then give its parents' goal statements as the reason. If a child loses its last parent while it has no `## Why`, the views report the missing answer.

Achievement is for the owner to judge. An outcome validation is an ordinary future event a timeline can define; recording its result is how achievement becomes evidence rather than opinion.

### Views

| Command | Shows |
| --- | --- |
| `tline goal list` | Goals by target date with status, owner, and past-due flags; filters by owner, status, and past due |
| `tline goal show <id or path>` | One goal as its seven answers, with the type, status, and past-due flag of every linked step |
| `tline goal tree [id or path]` | Goals and tasks beneath a goal, depth first in `## How` order, or beneath every top-level goal |
| `tline task list` | Tasks by due date with status, owner, parent goal, and past-due flags |
| `tline task show <id or path>` | One task as its five answers |

A goal or task is named by its ID, its path, or its file name with or without `.md`, such as `tline goal show plugins`; an ambiguous name lists the matches. List filters are `--owner`, `--status`, and `--past-due`, and drafts are flagged. Every view supports `--json`.

The views read the event files and the injected clock and never change files. They show drafts separately and mark results incomplete, as `list` does.
