# Writing a plugin

A plugin adds what `timeline.yaml` alone cannot express: rules across fields and sections, rules that need outside knowledge, and commands such as `tline prediction due`. Timeline ships one built-in plugin, `goals`. This page is about **external plugins**: programs you write, in any language, that live in your own repository.

If you only need fields, allowed values, required sections, and timing rules, you do not need a plugin; see the [quickstart](quickstart.md).

## How it works

You name the program in `timeline.yaml`. When `tline` needs the plugin, it starts the program in the timeline's folder, writes one JSON request to its standard input, and reads one JSON response from its standard output. Anything the program writes to standard error is shown to the user, so use it for logs. Each request starts the program once; it does not need to keep running.

There are three requests:

| Request | When | The plugin returns |
| --- | --- | --- |
| `describe` | Once per `tline` run, before anything else | Its name, protocol versions, type templates, whether it validates, and its commands |
| `validate` | During every `tline check`, including dry runs | Problems found in its events |
| `command` | When a user runs one of its commands | Text, JSON, and whether the result is incomplete |

A plugin only reads what `tline` sends it. It must not write event files: `tline check` re-reads every draft before stamping it and stops if one changed.

## Walkthrough: a predictions plugin in Python

This plugin adds a `prediction` type with two rules config cannot express, and a `tline prediction due` command. The full version is [examples/plugins/tools/tline-predictions](../examples/plugins/tools/tline-predictions); a Go plugin is in [examples/plugins/incidents](../examples/plugins/incidents/main.go).

### 1. Write the program

```python
#!/usr/bin/env python3
import json, sys

DESCRIBE = {
    "name": "predictions",
    "protocols": [1],
    "types": [{
        "name": "prediction",
        "entry_timing": "future",
        "fields": {"claim": {"type": "string", "required": True}},
        "status": {"values": ["open", "correct", "wrong"], "closed": ["correct", "wrong"]},
    }],
    "validates": True,
    "commands": [{"name": "prediction", "summary": "list predictions that are due",
                  "usage": "Usage:\n  tline prediction due\n"}],
}

def validate(req):
    problems = []
    for e in req["events"]:
        closed = e["fields"].get("status") in ("correct", "wrong")
        if closed and "Outcome" not in [s["heading"] for s in e["sections"]]:
            problems.append({"path": e["path"], "section": "Outcome",
                             "message": "is required once a prediction is closed"})
    return {"diagnostics": problems}

def command(req):
    if req["args"] != ["due"]:
        return {"usage_error": "expected 'due'"}
    due = [e for e in req["events"] if e["past_due"]]
    return {"text": "".join(e["path"] + "\n" for e in due),
            "json": {"due": [e["path"] for e in due]},
            "incomplete": bool(req["diagnostics"])}

req = json.load(sys.stdin)
handler = {"describe": lambda r: DESCRIBE, "validate": validate, "command": command}[req["request"]]
json.dump(handler(req), sys.stdout)
```

Save it as `tools/tline-predictions` in your timeline's repository and make it executable with `chmod +x tools/tline-predictions`.

### 2. Name it in `timeline.yaml`

```yaml
version: 1
plugins:
  - goals
  - predictions: { run: ./tools/tline-predictions, timeout: 10s }
types:
  prediction: { from: predictions/prediction, folder: predictions }
```

- `run` is a path relative to the timeline's folder, or the name of a program on your `PATH`.
- `timeout` is optional and defaults to `10s`.
- `from: predictions/prediction` builds the type from the plugin's template. You choose the folder, and may add fields and sections, as for any type.

A plugin can also add its rules to a type you configured yourself:

```yaml
types:
  claim:
    folder: claims
    entry_timing: future
    validators: [predictions]
```

The plugin then receives `claim` events too.

### 3. Approve it, then use it

```text
$ tline plugins
External plugins in /home/ada/my-work:
  predictions
    runs:    ./tools/tline-predictions
    program: a program inside this timeline: /home/ada/my-work/tools/tline-predictions
    timeout: 10s
Not approved: tline will not run these programs until you approve them.
Review them, then run 'tline plugins trust'.

$ tline plugins trust
...
Allow tline to run these programs for this timeline? [y/N] y
Approved 1 external plugin for /home/ada/my-work. They run until their settings or programs change.

$ tline check
$ tline prediction due
```

`tline --help`, run inside the timeline, lists the plugin's commands.

## Approval

A plugin is a program that runs with your permissions, so `tline` runs a timeline's external plugins only after you approve them for that timeline. This keeps `tline` safe to run in a repository someone else wrote.

- `tline plugins` shows each plugin, the program it runs, where that program is, and whether it is approved.
- `tline plugins trust` approves them. In a terminal it asks first; anywhere else it needs `--yes`.
- `tline plugins untrust` removes the approval.
- Approval records the timeline's folder and a fingerprint of its plugin settings and of every plugin program inside the folder. If either changes, `tline` asks again. Programs found on your `PATH` are identified by name only, because they are installed tools rather than part of the timeline. Only the program file itself is fingerprinted: files it reads or imports, such as Python modules beside it, are not, so review changes to them before you run `tline` after pulling someone else's changes.
- The review shows each plugin's `run` setting exactly. If it contains control or other invisible characters, it is shown quoted with them escaped, so a timeline cannot hide part of what you approve.
- Approvals are stored in your own configuration directory: `$TLINE_CONFIG_DIR`, else `$XDG_CONFIG_HOME/tline`, else `~/.config/tline`. They are never stored in the timeline, so a timeline cannot approve itself.
- Until approved, commands that need the plugins stop with **exit code 6** and say what to do. With `--json`, they also print `{"error": "plugins_not_approved", "root": ..., "changed": ..., "plugins": [...]}`.
- In CI, where the repository is already your own code, set `TLINE_TRUST_PLUGINS=1` to skip approval.

### If you use an agent

Agents are told not to approve plugins themselves (see [agents.md](agents.md)). If your agent reports exit code 6, ask it to explain the plugins: it can run `tline plugins`, open each program, and summarize what it does, especially whether it reads files outside the timeline, uses the network, or changes anything. When you are satisfied, run `tline plugins trust` yourself, or tell the agent explicitly to run `tline plugins trust --yes`.

## Protocol reference (version 1)

### Requests

Every request is one JSON object with:

| Field | Type | Meaning |
| --- | --- | --- |
| `protocol` | number | Always `1` for this version |
| `request` | string | `describe`, `validate`, or `command` |

`validate` and `command` requests also carry the **facts**:

| Field | Type | Meaning |
| --- | --- | --- |
| `now` | string | The current time, RFC 3339 UTC |
| `root` | string | The timeline's absolute folder; also the program's working directory |
| `config` | object | The timeline's configuration: `name`, `plugins` (enabled plugin names), and `types` |
| `events` | array | The events of the plugin's types, and of types that list it in `validators` |
| `links` | array | Every Markdown link in those events, resolved |

Each entry of `config.types` has the fields of a [type template](#type-templates) plus `folder`, `plugin` (the plugin that provides it, if any), `base` (the template's name, if renamed), `validators`, `history`, and `frozen`.

Each **event** has:

| Field | Type | Meaning |
| --- | --- | --- |
| `path` | string | Path relative to the root, with `/` |
| `type` | string | The type name in `timeline.yaml` |
| `id` | string | The event ID; absent for a draft |
| `draft` | boolean | Whether it has no stamps yet |
| `time` | string | The event time (`timeline_at`), RFC 3339 UTC; absent if invalid |
| `past_due` | boolean | A future event with an open status whose time has passed |
| `fields` | object | Frontmatter, by field name. Strings, numbers, booleans, null, and lists keep their YAML type; timestamps are strings |
| `field_lines` | object | The file line of each field, by name |
| `sections` | array | Each `##` section: `heading`, `line`, `body` (its text), and `items` (top-level list items, each with `text` and `line`) |

Each **link** has `from` (the event's path), `section`, `item` (the list item's index, or `-1` in prose), `line`, `target` (as written), `to` and `to_type` (the event and type it points to, if any), and `broken` (it points into a type folder, but no event is there; `tline` already reports it).

A `command` request adds:

| Field | Type | Meaning |
| --- | --- | --- |
| `command` | string | The command name |
| `args` | array of strings | The arguments after the command name, without `--json` |
| `json` | boolean | Whether the user asked for JSON |
| `diagnostics` | array | Problems `tline check` would report in the plugin's events, as below |

### Responses

**`describe`**

| Field | Type | Meaning |
| --- | --- | --- |
| `name` | string | Must equal the plugin's name in `timeline.yaml` |
| `protocols` | array of numbers | Protocol versions the plugin speaks; must include `1` |
| `types` | array | [Type templates](#type-templates) |
| `validates` | boolean | Whether `tline` should send `validate` requests |
| `commands` | array | Each with `name`, `summary` (one line for `tline --help`), and `usage` (shown for `tline <name> --help`) |

**`validate`**: `{"diagnostics": [...]}`. Each diagnostic has `path` and `message`, and optionally `line`, and `field` or `section`. A diagnostic for a file outside the plugin's events is reported as the plugin's mistake.

**`command`**: `text` (printed without `--json`), `json` (any value, printed with `--json`), `incomplete` (true makes `tline` exit 5), or `usage_error` (a message; `tline` exits 2).

### Type templates

A template has the shape of a type in `timeline.yaml`:

```json
{
  "name": "prediction",
  "entry_timing": "future",
  "max_delay": "15m",
  "fields": { "claim": { "type": "string", "required": true, "values": ["a", "b"] },
              "tags": { "type": "list", "items": "string" } },
  "sections": ["Reasoning"],
  "status": { "values": ["open", "done"], "closed": ["done"] }
}
```

`entry_timing` is `past`, `future`, or `real_time` (which needs `max_delay`). Field types are `string`, `number`, `bool`, `time`, and `list` with `items`. Templates cannot define shared fields such as `timeline_at`, or `status` as a field.

### Errors

`tline` stops with exit code 1 and names the plugin if its program is missing or not executable, exits with a non-zero status, does not answer within its timeout, prints something that is not JSON, describes itself with another name, or does not speak protocol version 1. Configuration errors exit 3: a plugin named like a built-in plugin, or a command or type that another plugin, or a core command, already uses.

### Compatibility

Version 1 only grows: `tline` may add request and response fields, and plugins should ignore fields they do not know. A change that would break plugins gets a new protocol version, and a plugin lists every version it speaks in `protocols`.
