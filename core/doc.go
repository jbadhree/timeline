// Package core holds Timeline's domain values and rules: configuration and
// type definitions, event documents, timestamps, timing rules, the built-in
// goal and task types, and validation.
//
// Core is pure. It never reads files, parses YAML, or reads the clock; adapters decode input into the values defined here, and callers
// inject the current time. Validation results are data (Diagnostics), never
// printed. Results are sorted so output does not depend on input order.
//
// Owning goals: goals/events-and-types.md, goals/two-times-and-timing.md,
// and goals/goals-and-tasks.md.
package core
