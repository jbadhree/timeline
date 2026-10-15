---
id: 01M4WFZBQRG2PMN47CQMMK2KBZ
entered_at: 2026-10-14T06:00:43Z
entry_timing: future
type: task
timeline_at: 2026-10-20T23:59:59Z
created_by: claude
task: Review timing rules and recorded metadata
owner: Badhree Babu
status: done
---

## How

- Read `core/time.go`, `core/ulid.go`, and the timing and recorded-metadata parts of `core/validate.go`.
- Compare them with the Design of goals/two-times-and-timing.md, especially that `entry_timing` is authoritative once recorded.
- Change a type's `entry_timing` after recording an event, and check that the event still passes.
- Try timestamps with offsets, fractional seconds, and the boundaries of each rule.

## Findings

- **Fixed.** A recorded event was judged by its type's *current* timing rule, and an `entry_timing` that no longer matched the type was rejected, so changing a type's rule in `timeline.yaml` broke every recorded event of that type. This contradicted the design and the timing goal's `## Done when`, which was marked achieved. The recorded `entry_timing` now decides; an event entered as `real_time` in a type that is no longer Real-time keeps only the "not entered early" check. Tests: `TestRecordedEventKeepsItsEntryTiming` (core) and `TestRecordedEventKeepsItsTimingRuleWhenTheTypeChanges` (end to end).
- **Acceptable.** Timestamp parsing, whole-second precision, the boundaries of each rule, and ULID generation and validation match the design and are tested.
