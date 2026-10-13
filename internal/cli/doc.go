// Package cli parses tline arguments, renders terminal and JSON output, and
// maps outcomes to exit codes.
//
// It owns no Timeline rules; validation, recording, and reading belong to the
// core. Run never calls os.Exit and writes only to the streams it is given:
// results go to stdout, which stays parseable with --json, and diagnostics go
// to stderr.
//
// Exit codes are provisional until the first release (see the "Command line"
// section of goals/primary-goal-build-timeline.md).
package cli
