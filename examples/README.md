# Examples

Worked examples of Timeline roots. They illustrate configuration and event files; the types in them are example data, not types built into Timeline.

Files are shown as they look after `tline check` has recorded them. In a draft, `id`, `entered_at`, and `entry_timing` are omitted; `check` assigns the first two, copies `entry_timing` from the type, and inserts all three at the top of the frontmatter. The formats follow [events and types](../goals/events-and-types.md#design).

## markets

[`markets/timeline.yaml`](markets/timeline.yaml) defines two types:

- `news` uses the Past rule: the event happened before it was entered.
- `prediction` uses the Future rule: it is entered before the expected event.

Day 1, 2026-10-01:

| Entered (UTC) | Event time (UTC) | Type | File |
| --- | --- | --- | --- |
| 14:00 | 2026-10-01 13:00 | news | [NVDA expands in China](markets/news/2026-10-01-nvda-expands-in-china.md) |
| 15:00 | 2026-12-31 23:59:59 | prediction | [NVDA all-time high](markets/predictions/2026-10-01-nvda-all-time-high.md) |

The prediction links to the news with an ordinary Markdown link. Its `beliefs` are plain strings referring to the consumer's own world model; Timeline stores them without interpreting them. Its event time is a placeholder horizon by which the prediction should come true.
