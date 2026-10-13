// Command tline records and reads a Timeline root.
//
// It only wires the process to the CLI package: signals, arguments, standard
// streams, and the exit code. See goals/primary-goal-build-timeline.md.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/jbadhree/timeline/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
