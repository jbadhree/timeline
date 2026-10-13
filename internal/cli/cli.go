package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime/debug"
)

// Exit codes returned by Run.
const (
	ExitOK         = 0 // success
	ExitFailure    = 1 // operational failure, such as an unwritable output stream
	ExitUsage      = 2 // invalid flags, arguments, or command
	ExitValidation = 3 // files or configuration failed validation
	ExitConflict   = 4 // another check holds the lock, or a file changed during the check
	ExitIncomplete = 5 // a view could not account for every relevant event
)

// Version is the release version. Release builds set it with
// -ldflags "-X github.com/jbadhree/timeline/internal/cli.Version=v0.1.0";
// otherwise it falls back to module build information.
var Version = ""

// Options are the global flags shared by every command.
type Options struct {
	// Root is the Timeline root given with --root. Empty means search upward
	// from the working directory for the nearest timeline.yaml.
	Root string
	// JSON selects machine-readable output on stdout.
	JSON bool
}

const usage = `tline records events on a timeline and reads them back.

Usage:
  tline [flags] <command> [arguments]

Flags:
  --root <path>   Timeline root (default: nearest timeline.yaml upward)
  --json          machine-readable output on stdout
  --version       print the version and exit
  --help          print this help and exit

Commands:
  check           validate event files and stamp new drafts
  help            print this help

Run 'tline <command> --help' for a command's flags.
`

// Run executes tline with args (excluding the program name) and returns the
// process exit code. ctx carries cancellation from the caller.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var opts Options
	var showVersion bool
	fs := flag.NewFlagSet("tline", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.Root, "root", "", "")
	fs.BoolVar(&opts.JSON, "json", false, "")
	fs.BoolVar(&showVersion, "version", false, "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return writeOut(stdout, stderr, usage+pluginUsage())
		}
		return usageError(stderr, err.Error())
	}

	if showVersion {
		return printVersion(stdout, stderr, opts.JSON)
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return usageError(stderr, "no command given")
	}
	switch rest[0] {
	case "help":
		return writeOut(stdout, stderr, usage+pluginUsage())
	case "check":
		return runCheck(ctx, opts, rest[1:], stdout, stderr)
	}
	cmds, err := pluginCommands()
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	if pc, ok := cmds[rest[0]]; ok {
		return runPluginCommand(ctx, opts, pc, rest[1:], stdout, stderr)
	}
	return usageError(stderr, fmt.Sprintf("unknown command %q", rest[0]))
}

func usageError(stderr io.Writer, msg string) int {
	diagnose(stderr, "%s\nRun 'tline --help' for usage.", msg)
	return ExitUsage
}

// writeOut writes a result to stdout. A result that cannot be delivered is an
// operational failure, reported on stderr.
func writeOut(stdout, stderr io.Writer, s string) int {
	if _, err := io.WriteString(stdout, s); err != nil {
		diagnose(stderr, "writing output: %v", err)
		return ExitFailure
	}
	return ExitOK
}

// diagnose writes one diagnostic line to stderr. A failure to write it is
// ignored: stderr is the last place left to report anything.
func diagnose(stderr io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(stderr, "tline: "+format+"\n", args...)
}

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

func printVersion(stdout, stderr io.Writer, asJSON bool) int {
	v := currentVersion()
	if asJSON {
		b, err := json.Marshal(v)
		if err != nil {
			diagnose(stderr, "encoding version: %v", err)
			return ExitFailure
		}
		return writeOut(stdout, stderr, string(b)+"\n")
	}
	if v.Commit != "" {
		return writeOut(stdout, stderr, fmt.Sprintf("tline %s (%s)\n", v.Version, v.Commit))
	}
	return writeOut(stdout, stderr, fmt.Sprintf("tline %s\n", v.Version))
}

func currentVersion() versionInfo {
	v := versionInfo{Version: Version}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		if v.Version == "" {
			v.Version = "devel"
		}
		return v
	}
	if v.Version == "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v.Version = info.Main.Version
	}
	if v.Version == "" {
		v.Version = "devel"
	}
	dirty := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			v.Commit = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if v.Commit != "" && dirty {
		v.Commit += "-dirty"
	}
	return v
}
