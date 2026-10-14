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
	"github.com/jbadhree/timeline/internal/plugins/external"
)

// coreCommands are handled by the CLI itself; plugins cannot take them.
var coreCommands = map[string]bool{"check": true, "help": true, "history": true, "plugins": true}

// pluginCommand is a command and the plugin that registered it.
type pluginCommand struct {
	plugin core.Plugin
	cmd    core.Command
}

func isExternal(p core.Plugin) bool {
	_, ok := p.(*external.Plugin)
	return ok
}

// commandClash is a command name taken twice, or a core command taken by a
// plugin. With an external plugin involved it is a configuration error.
type commandClash struct {
	msg      string
	external bool
}

func (e *commandClash) Error() string { return e.msg }

// pluginCommands collects the commands of the given plugins.
func pluginCommands(ps []core.Plugin) (map[string]pluginCommand, error) {
	cmds := map[string]pluginCommand{}
	for _, p := range ps {
		cp, ok := p.(core.CommandProvider)
		if !ok {
			continue
		}
		for _, c := range cp.Commands() {
			switch prev, dup := cmds[c.Name]; {
			case coreCommands[c.Name]:
				return nil, &commandClash{fmt.Sprintf("plugin %s registers command %q, which is a core command", p.Name(), c.Name), isExternal(p)}
			case dup:
				return nil, &commandClash{fmt.Sprintf("command %q is registered by both plugin %s and plugin %s", c.Name, prev.plugin.Name(), p.Name()), isExternal(p) || isExternal(prev.plugin)}
			case c.Name == "" || strings.HasPrefix(c.Name, "-") || c.Run == nil:
				return nil, &commandClash{fmt.Sprintf("plugin %s registers an invalid command %q", p.Name(), c.Name), isExternal(p)}
			}
			cmds[c.Name] = pluginCommand{plugin: p, cmd: c}
		}
	}
	return cmds, nil
}

// reportClash returns 3 for a configuration error caused by an external
// plugin, and 1 for a clash among built-in plugins, which is a bug in tline.
func reportClash(err error, stderr io.Writer) int {
	diagnose(stderr, "%v", err)
	var clash *commandClash
	if errors.As(err, &clash) && clash.external {
		return ExitValidation
	}
	return ExitFailure
}

// pluginUsage lists plugin commands for tline --help: the built-in ones, and
// those of the timeline the command runs in, if its plugins are approved.
func pluginUsage(ctx context.Context, opts Options, stderr io.Writer) string {
	var b strings.Builder
	if cmds, err := pluginCommands(plugins()); err == nil && len(cmds) > 0 {
		b.WriteString("\nCommands from plugins:\n")
		writeCommands(&b, cmds, "")
	}
	root, err := findRootQuietly(opts)
	if err != nil {
		return b.String()
	}
	data, err := os.ReadFile(root + string(os.PathSeparator) + config.FileName)
	if err != nil {
		return b.String()
	}
	specs, _ := config.ExternalPlugins(data)
	if len(specs) == 0 {
		return b.String()
	}
	if err := trustPlugins(root, specs); err != nil {
		var names []string
		for _, s := range specs {
			names = append(names, s.Name)
		}
		fmt.Fprintf(&b, "\nThis timeline's external plugins (%s) are not approved, so their commands are not listed.\nRun 'tline plugins' to review them.\n", strings.Join(names, ", "))
		return b.String()
	}
	tl, err := loadTimeline(ctx, root, stderr)
	if err != nil {
		fmt.Fprintf(&b, "\nThis timeline's external plugins could not be loaded: %v\n", err)
		return b.String()
	}
	var ext []core.Plugin
	for _, p := range tl.plugins {
		if isExternal(p) {
			ext = append(ext, p)
		}
	}
	if cmds, err := pluginCommands(ext); err == nil && len(cmds) > 0 {
		b.WriteString("\nCommands from this timeline's external plugins:\n")
		writeCommands(&b, cmds, ", external")
	}
	return b.String()
}

func writeCommands(b *strings.Builder, cmds map[string]pluginCommand, kind string) {
	names := make([]string, 0, len(cmds))
	for n := range cmds {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		pc := cmds[n]
		fmt.Fprintf(b, "  %-15s %s (%s plugin%s)\n", n, pc.cmd.Summary, pc.plugin.Name(), kind)
	}
}

// findRootQuietly finds the timeline the command runs in, if any.
func findRootQuietly(opts Options) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return config.FindRoot(opts.Root, cwd)
}

// runNamedCommand runs a command that is not a core command: a built-in
// plugin's, or one from the timeline's external plugins.
func runNamedCommand(ctx context.Context, opts Options, name string, args []string, stdout, stderr io.Writer) int {
	builtin, err := pluginCommands(plugins())
	if err != nil {
		return reportClash(err, stderr)
	}
	bpc, isBuiltin := builtin[name]
	if isBuiltin {
		for _, a := range args {
			if a == "-h" || a == "--help" || a == "-help" {
				usage := bpc.cmd.Usage
				if usage == "" {
					usage = bpc.cmd.Summary
				}
				return writeOut(stdout, stderr, strings.TrimRight(usage, "\n")+"\n")
			}
		}
	}

	root, err := findRootQuietly(opts)
	if err != nil {
		if !isBuiltin {
			return usageError(stderr, fmt.Sprintf("unknown command %q", name))
		}
		if errors.Is(err, config.ErrNoRoot) {
			return usageError(stderr, err.Error())
		}
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	tl, err := loadTimeline(ctx, root, stderr)
	if err != nil {
		var cerr *config.Error
		if errors.As(err, &cerr) {
			for _, d := range cerr.Diagnostics {
				diagnose(stderr, "%s", d)
			}
			diagnose(stderr, "%s is invalid; run 'tline check' for details", config.FileName)
			return ExitValidation
		}
		var uerr *untrustedError
		if !isBuiltin && errors.As(err, &uerr) {
			diagnose(stderr, "%q is not a core command; this timeline's external plugins may provide it", name)
		}
		return reportLoadError(err, opts.JSON, stdout, stderr)
	}
	all, err := pluginCommands(tl.plugins)
	if err != nil {
		return reportClash(err, stderr)
	}
	pc, ok := all[name]
	if !ok {
		if isBuiltin {
			return usageError(stderr, fmt.Sprintf("%s: the %s plugin is not enabled in %s; add it to plugins", name, bpc.plugin.Name(), config.FileName))
		}
		return usageError(stderr, fmt.Sprintf("unknown command %q", name))
	}
	return runPluginCommand(ctx, opts, tl, pc, args, stdout, stderr)
}

// runPluginCommand runs a plugin command over the timeline's events.
func runPluginCommand(ctx context.Context, opts Options, tl *loaded, pc pluginCommand, args []string, stdout, stderr io.Writer) int {
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

	docs, _, err := document.Scan(ctx, tl.root, tl.cfg)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}

	out, err := core.RunCommand(ctx, core.CheckInput{Config: tl.cfg, Docs: docs, Now: clock(), Plugins: tl.plugins}, pc.plugin, pc.cmd, rest, opts.JSON)
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
