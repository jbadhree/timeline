// Package core holds Timeline's domain values and rules: configuration and
// type definitions, event documents, timestamps, timing rules, links, and
// validation. It knows nothing about any particular type: goals, tasks, and
// other types with rules come from plugins.
//
// Core is pure. It never reads files, parses YAML, or reads the clock;
// adapters decode input into the values defined here, and callers inject the
// current time. Validation results are data (Diagnostics), never
// printed. Results are sorted so output does not depend on input order.
//
// Plugins (plugin.go) add types, rules, and commands on top of the core rules;
// they see read-only facts and can only add diagnostics.
//
// Owning goals: goals/events-and-types.md, goals/two-times-and-timing.md,
// goals/goals-and-tasks.md, and goals/plugins.md.
package core
