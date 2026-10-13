# Timeline

Timeline gives agents, and the humans working with them, a sense of time. It records events on a timeline with two times for each: when the event happened (or is expected), and when it entered the timeline. News is learned after the fact, predictions are made in advance, and live observations are recorded as they happen. Future events, such as tasks and goals, can go past due.

Events are Markdown files with YAML metadata, edited by humans and agents with any tool. Users define their own event types and Past, Future, or Real-time entry rules. Goals and tasks ship as a built-in plugin: a goal answers what, why, by when, how, how we know it is done, why it will work, and who owns it; a task is a simpler step whose result is obvious.

## Using tline

- [Quickstart](docs/quickstart.md): create a timeline, write and check events, and use goals and tasks.
- [Instructions for agents](docs/agents.md): a short page to give an AI agent that keeps a timeline.

## Start here

This repository plans and designs Timeline as Timeline goals and tasks. Start at the [primary goal](goals/primary-goal-build-timeline.md), which explains the system and links to every goal and task.

- [`goals/`](goals/): what we are building and why, with the design for each part.
- [`tasks/`](tasks/): concrete steps whose result is obvious.
- [`decisions/`](decisions/): design decisions, current and superseded, linked from the goals they shaped.
- [`examples/`](examples/README.md): worked configuration and event files.

## Install

Releases are published on the [GitHub releases page](https://github.com/jbadhree/timeline/releases) for macOS and Linux, on Intel (`amd64`) and ARM (`arm64`). Download the archive for your platform, check it against `SHA256SUMS`, and put `tline` on your `PATH`:

```sh
shasum -a 256 -c SHA256SUMS --ignore-missing
tar -xzf tline_v0.1.0_darwin_arm64.tar.gz
mv tline_v0.1.0_darwin_arm64/tline /usr/local/bin/
```

The macOS binaries are not signed, so macOS blocks a downloaded copy at first. Allow it with `xattr -d com.apple.quarantine /usr/local/bin/tline`. With Go installed you can instead run `go install github.com/jbadhree/timeline/cmd/tline@latest`, which macOS does not block.

Windows is not supported yet.

## Development

Requires Go (the `toolchain` line in `go.mod` downloads the pinned version automatically) and Make, on macOS or Linux. No credentials or global configuration are needed.

```sh
make check   # format, CGO-free build, lint, unit and end-to-end tests
make race    # tests with the race detector
make plan    # validate this repository's goals and tasks with tline
make release VERSION=v0.1.0   # build release archives into dist/
```

Pushing a tag such as `v0.1.0` runs [the release workflow](.github/workflows/release.yml): every check on macOS and Linux, the release build, a smoke test of each binary on its own OS, then a GitHub prerelease with the archives and `SHA256SUMS`. It refuses to publish without a `LICENSE` file.

`make check` installs the pinned golangci-lint into `bin/` on first run. CI runs both targets on macOS and Linux for every push to `main` and every pull request.

## Status

`tline check` validates a Timeline root (configuration, fields, timing, goal and task answers, links, and goal cycles) and stamps every valid draft with its `id`, `entered_at`, and `entry_timing`; `--dry-run` validates without writing. `tline goal list | show | tree` and `tline task list | show` view goals and tasks, flagging past-due ones; they come from the built-in goals plugin. `tline list` for all events is not built yet. The planned Go executable is `tline`, with `check`, `list`, `goal`, and `task` commands backed by a reusable core. Every command reads the files directly; there is no database and no version control requirement. A SQLite record store with `tline history` is a later direction.

Applications keep their own configuration, events, and domain logic and use an installed `tline` binary. Read [AGENTS.md](AGENTS.md) before making changes.

## License

Timeline is source-available under the [PolyForm Noncommercial License 1.0.0](LICENSE). You may use, change, and share it for any noncommercial purpose: personal use, study, hobby projects, and use by charities, schools, research, and government organizations. Selling it, or using it for a commercial purpose, needs separate permission from the copyright holder. It is not an open-source license.

## Contributing

Contributions are welcome. By submitting one, you agree that it is licensed under the same terms as the project. Start with [AGENTS.md](AGENTS.md) and the [primary goal](goals/primary-goal-build-timeline.md).
