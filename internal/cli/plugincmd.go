package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/document"
)

// coreCommands are handled by the CLI itself; plugins cannot take them.
var coreCommands = map[string]bool{"check": true, "help": true}

// pluginCommand is a command and the plugin that registered it.
type pluginCommand struct {
	plugin core.Plugin
	cmd    core.Command
}

// pluginCommands collects the commands of every plugin. A command that clashes
// with a core command or another plugin's command is an error.
func pluginCommands() (map[string]pluginCommand, error) {
	cmds := map[string]pluginCommand{}
	for _, p := range plugins() {
		cp, ok := p.(core.CommandProvider)
		if !ok {
			continue
		}
		for _, c := range cp.Commands() {
			switch prev, dup := cmds[c.Name]; {
			case coreCommands[c.Name]:
				return nil, fmt.Errorf("plugin %s registers command %q, which is a core command", p.Name(), c.Name)
			case dup:
				return nil, fmt.Errorf("command %q is registered by both plugin %s and plugin %s", c.Name, prev.plugin.Name(), p.Name())
			case c.Name == "" || strings.HasPrefix(c.Name, "-") || c.Run == nil:
				return nil, fmt.Errorf("plugin %s registers an invalid command %q", p.Name(), c.Name)
			}
			cmds[c.Name] = pluginCommand{plugin: p, cmd: c}
		}
	}
	return cmds, nil
}

// pluginUsage lists plugin commands for tline --help.
func pluginUsage() string {
	cmds, err := pluginCommands()
	if err != nil || len(cmds) == 0 {
		return ""
	}
	names := make([]string, 0, len(cmds))
	for n := range cmds {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("\nCommands from plugins:\n")
	for _, n := range names {
		pc := cmds[n]
		fmt.Fprintf(&b, "  %-15s %s (%s plugin)\n", n, pc.cmd.Summary, pc.plugin.Name())
	}
	return b.String()
}

// runPluginCommand runs a plugin command over the root's events.
func runPluginCommand(ctx context.Context, opts Options, pc pluginCommand, args []string, stdout, stderr io.Writer) int {
	// --json may come after the command, as in "tline goal list --json".
	var rest []string
	for _, a := range args {
		switch a {
		case "--json", "-json":
			opts.JSON = true
		case "-h", "--help", "-help":
			usage := pc.cmd.Usage
			if usage == "" {
				usage = pc.cmd.Summary
			}
			return writeOut(stdout, stderr, strings.TrimRight(usage, "\n")+"\n")
		default:
			rest = append(rest, a)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		diagnose(stderr, "finding working directory: %v", err)
		return ExitFailure
	}
	root, err := config.FindRoot(opts.Root, cwd)
	if err != nil {
		if errors.Is(err, config.ErrNoRoot) {
			return usageError(stderr, err.Error())
		}
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	cfg, err := config.Load(root, plugins())
	if err != nil {
		var cerr *config.Error
		if errors.As(err, &cerr) {
			for _, d := range cerr.Diagnostics {
				diagnose(stderr, "%s", d)
			}
			diagnose(stderr, "%s is invalid; run 'tline check' for details", config.FileName)
			return ExitValidation
		}
		diagnose(stderr, "reading %s: %v", config.FileName, err)
		return ExitFailure
	}
	docs, _, err := document.Scan(ctx, root, cfg)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}

	out, err := core.RunCommand(ctx, core.CheckInput{Config: cfg, Docs: docs, Now: clock(), Plugins: plugins()}, pc.plugin, pc.cmd, rest, opts.JSON)
	if err != nil {
		var uerr core.UsageError
		if errors.As(err, &uerr) {
			return usageError(stderr, pc.cmd.Name+": "+uerr.Msg)
		}
		diagnose(stderr, "%s: %v", pc.cmd.Name, err)
		return ExitFailure
	}

	text := out.Text
	if opts.JSON {
		b, err := json.MarshalIndent(out.JSON, "", "  ")
		if err != nil {
			diagnose(stderr, "%s: encoding result: %v", pc.cmd.Name, err)
			return ExitFailure
		}
		text = string(b)
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if c := writeOut(stdout, stderr, text); c != ExitOK {
		return c
	}
	if out.Incomplete {
		if !opts.JSON {
			diagnose(stderr, "%s: result is incomplete; run 'tline check' to see the problems", pc.cmd.Name)
		}
		return ExitIncomplete
	}
	return ExitOK
}
