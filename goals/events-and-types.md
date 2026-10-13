---
id: 01M4SXMJ8GXW317PX0W6SPJ1P5
entered_at: 2026-10-13T06:01:46Z
entry_timing: future
type: goal
timeline_at: 2026-11-07T23:59:59Z
created_by: claude
goal: Let each timeline define its own event types and keep events as readable Markdown files
owner: Badhree Babu
status: open
---

## Why

Anything worth remembering can be an event, so users must define their own types. Timeline has no built-in knowledge of investing, prediction scoring, or any particular agent. Plain Markdown with YAML metadata lets humans and agents add, read, and update events with any editor.

## How

- [Find the Timeline root from the nearest timeline.yaml or --root](../tasks/root-discovery.md)
- [Decode and validate timeline.yaml strictly](../tasks/decode-config.md)
- [Read event files into frontmatter, sections, list items, and links](../tasks/decode-event-files.md)
- [Report every problem with its file, line, field or section, and reason](../tasks/report-diagnostics.md)

## Done when

- The [markets example](../examples/README.md) and this repository's `timeline.yaml` load, with no consumer code imported.
- Invalid field values, unknown fields, wrong field types, and missing sections are each reported.
- Root discovery works from nested directories and with `--root`, and reports missing or invalid configuration and paths outside the configured folders.

## Why it will work

Markdown and YAML are readable without Timeline, so nothing is locked in. Treating unknown fields as errors catches agent typos the moment a file is checked, instead of letting bad data spread quietly.

## Design

### Root discovery

A tracked `timeline.yaml` marks the Timeline root. By default, search the current directory and then its parents and use the nearest one. Report configuration errors rather than falling back to another root. Resolve configured folders relative to the root, resolve path arguments relative to the caller's working directory, and reject paths outside the configured folders. Only configured folders are scanned, so a root can share a repository with code and other files.

```sh
tline --root /path/to/my-project check
```

An explicit root requires `timeline.yaml` at that location. Timeline never invents types or creates configuration.

### `timeline.yaml`

```yaml
version: 1            # configuration format version
name: markets         # display name of this timeline
types:
  news:
    folder: news
    entry_timing: past          # past | future | real_time
    fields:
      text: { type: string, required: true }
      ticker: { type: string, required: true }
  observation:
    folder: observations
    entry_timing: real_time
    max_delay: 15m              # required for real_time; Go duration syntax
    status: { values: [open, closed], closed: [closed] }
    sections: [Source]          # required level-2 body headings
  goal:
    folder: goals               # built-in; see goals-and-tasks
```

- `fields` declare the type-specific frontmatter. Field types are `string`, `number`, `bool`, `time` (RFC 3339), and `list` with an `items` scalar type. `values` restricts a scalar or list item to listed values. `required` defaults to false.
- `sections` lists `##` headings the body must contain. It is optional.
- `status` is optional. It lists a type's allowed `status` values and which are `closed`. A type with `status` requires a `status` field in every event. See [past due](two-times-and-timing.md#past-due).
- The type names `goal` and `task` are reserved for the [built-in types](goals-and-tasks.md). A root enables each by declaring it, chooses its folder, and may add `fields` and `sections`; the built-in fields, sections, timing rule, and statuses cannot be removed or redefined.
- A type cannot redefine the shared fields below. Each folder belongs to one type.
- Frontmatter fields that are neither shared nor declared are errors.

### Event files

Each event is one Markdown file in its type's folder: YAML frontmatter, then a Markdown body. File names are free; Timeline never renames files. Shared frontmatter fields:

| Field | Meaning | Set by |
| --- | --- | --- |
| `id` | Unique event ID, kept across moves | `tline check` at first recording |
| `type` | Configured type name; must match the folder's type | Author |
| `timeline_at` | Event time | Author |
| `entered_at` | Entry time: when the event was first recorded | `tline check` at first recording |
| `entry_timing` | The type's timing rule when the event was first recorded | `tline check` at first recording |
| `created_by` | Free-text label for the human or agent that added the event | Author |

`type`, `timeline_at`, and `created_by` are required. A draft omits `id`, `entered_at`, and `entry_timing`; see [recording](record-events.md). Field values keep their declared types, so `ticker: 123` is a number, not a string; quote it to make it text. Missing stays distinct from empty.

### Links

Events can link to other events, sections, external documents, or sources with relative Markdown links and optional heading anchors. A target with spaces is written percent-encoded, `[x](list%20files.md)`, or in angle brackets, `[x](<list files.md>)`. In every event, a link that points into a configured type folder must resolve to an event file; `tline check` reports one that does not. Beyond that, Timeline gives links no meaning except in [goals and tasks](goals-and-tasks.md#design), and that meaning [belongs in plugins](../decisions/relations-in-plugins.md). Backlinks and link repair are later work.
