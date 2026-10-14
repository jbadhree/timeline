# Timeline instructions for agents

Give this page to an agent that keeps a Timeline, or copy it into the agent's instructions file. It assumes `tline` is installed; [quickstart.md](quickstart.md) explains everything in more detail.

## What a timeline is

A folder with a `timeline.yaml` that defines event types, each stored in its own folder as Markdown files with YAML frontmatter. Every event has two times: `timeline_at`, when it happened or is due, which you write; and `entered_at`, when it was recorded, which only `tline check` writes. Read `timeline.yaml` first to learn the types, their folders, fields, statuses, required sections, and timing rules.

## Adding or changing an event

1. Write the Markdown file in the type's folder. Include `type`, `timeline_at` (RFC 3339 with an offset, such as `2026-10-13T15:00:00Z`), `created_by` (your name as an agent), and the type's required fields and `##` sections. Use only fields the type declares.
2. Run `tline check --dry-run --json <path>`.
3. If `valid` is false, fix every entry in `diagnostics` and repeat step 2.
4. Run `tline check --json <path>` to record it. Confirm the file is listed in `stamped`.

Edit recorded events the same way: change the file, then check it.

## Rules

- **Never write or change `id`, `entered_at`, or `entry_timing`.** `tline check` stamps them, and a file with only some of them, or with altered values, is rejected.
- **Pick the right event time.** For a `past` type, `timeline_at` must be before now; for a `future` type, after now; for `real_time`, within the type's `max_delay` of now. If you learn about something late, keep the real event time; do not move it to make a check pass.
- **Record plans before their dates.** A `future` draft whose date has passed cannot be recorded.
- **Keep links real.** A Markdown link into a type folder must point to an existing event file, relative to the file containing it.
- **Do not delete or rewrite other writers' events** to make a check pass; report the problem instead.
- **Never edit `history/`.** For a type with `history: true`, `tline check` saves a copy of every recorded version there. A change to a field or section listed in the type's `frozen`, or to any stamp, is rejected with the original value: keep the original, and record a new event, such as an outcome or a correction, instead of rewriting a frozen claim.
- With the goals plugin: a goal needs `goal`, `owner`, `status`, and the sections `## Why`, `## How`, `## Done when`, and `## Why it will work`, and every `## How` item must link to a goal or task. A task needs `task`, `owner`, `status`, and `## How` with plain steps that link to no goal or task. A goal or task linked from a goal's How may omit `## Why`.

## Reading `tline check --json`

```json
{
  "root": "/path/to/my-work",
  "dry_run": true,
  "recorded": false,
  "valid": false,
  "events": [{ "path": "tasks/oops.md", "type": "task", "draft": true, "valid": false }],
  "diagnostics": [
    { "path": "tasks/oops.md", "line": 6, "field": "onwer", "message": "is not a field of type \"task\"" },
    { "path": "tasks/oops.md", "section": "Why", "message": "is required because no goal links to this task from its How section" }
  ],
  "stamped": [],
  "history": []
}
```

Each diagnostic names a file and, where known, a line and a `field` or `section`. `recorded` is true when the check wrote anything; `stamped` lists each stamped draft with its new `id` and `entered_at`, and `history` lists each history copy saved, with its `version` and `reason`. A `conflict` field explains why writing stopped early.

## Exit codes

| Code | Meaning | What to do |
| --- | --- | --- |
| 0 | Success | Continue |
| 1 | `tline` failed | Report the error on stderr |
| 2 | Wrong command, flag, or path | Fix the command |
| 3 | Validation failed; nothing was written | Fix the diagnostics and check again |
| 4 | Another check holds the lock, or a file changed during the check | Wait and run the check again |
| 5 | A view is incomplete because some events are invalid | Run `tline check` and fix them |
| 6 | The timeline's external plugins are not approved, or changed since approval | Help your user review them; see below |

## Reading the timeline

- `tline goal tree --json`, `tline goal list --json`, `tline goal show <name> --json`
- `tline task list --json`, filtered with `--owner`, `--status`, or `--past-due`
- `tline task show <name> --json`
- `tline history <id|path|name> --json` lists an event's saved versions and what changed in each; add `--version N` to read one

Name a goal or task by its `id`, its path, or its file name without `.md`.

## External plugins and approval

A timeline can name external plugins in `timeline.yaml`: programs that `tline` runs to check events and to provide commands. They run with the user's permissions, so `tline` runs them only after the user approves them for that timeline. When a command exits 6:

1. **Do not approve the plugins yourself.** Do not run `tline plugins trust --yes`, and do not set `TLINE_TRUST_PLUGINS=1`, unless your user explicitly tells you to.
2. Run `tline plugins` to list each plugin, the program it runs, and where that program is.
3. Read each program inside the timeline and summarize it for your user: what it checks, and whether it reads files outside the timeline, uses the network, runs other programs, or changes anything. Say plainly if you cannot tell. For a program found on `PATH`, say what you know about it and that its code is not in the timeline.
4. Ask your user whether to approve them. They can run `tline plugins trust` themselves, or tell you to run `tline plugins trust --yes`.

If the plugins were approved before, `tline` reports that they changed: show your user what changed in `timeline.yaml` or in the program before asking again. To write a plugin, see [Writing a plugin](plugins.md).
