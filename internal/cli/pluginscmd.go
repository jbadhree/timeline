package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/plugins/external"
	"github.com/jbadhree/timeline/internal/trust"
	"golang.org/x/term"
)

const pluginsUsage = `Usage:
  tline plugins                 show this timeline's external plugins and whether they are approved
  tline plugins trust [--yes]   approve them, after reviewing the programs they run
  tline plugins untrust         remove the approval

External plugins are programs named in timeline.yaml. tline runs them only
after you approve them for this timeline, and asks again if their settings
or programs change. Approvals are stored in your own configuration
($TLINE_CONFIG_DIR, else $XDG_CONFIG_HOME/tline, else ~/.config/tline), never
in the timeline. In a terminal, trust asks for confirmation; elsewhere it
needs --yes. TLINE_TRUST_PLUGINS=1 skips approval, for CI.

Flags:
  --yes    approve without asking
  --json   machine-readable output on stdout
`

// stdin and stdinIsTerminal are replaced by tests.
var (
	stdin           io.Reader = os.Stdin
	stdinIsTerminal           = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)

type pluginInfo struct {
	Name       string `json:"name"`
	Run        string `json:"run"`
	Timeout    string `json:"timeout"`
	Program    string `json:"program"` // where the program was found, or "" if not found
	InTimeline bool   `json:"in_timeline"`
}

type pluginsOutput struct {
	Root     string       `json:"root"`
	Status   string       `json:"status"` // none, approved, not_approved, changed
	Plugins  []pluginInfo `json:"plugins"`
	Approved bool         `json:"approved"`
}

func runPlugins(opts Options, args []string, stdout, stderr io.Writer) int {
	sub, yes := "", false
	for _, a := range args {
		switch a {
		case "--yes", "-yes", "-y":
			yes = true
		case "--json", "-json":
			opts.JSON = true
		case "-h", "--help", "-help":
			return writeOut(stdout, stderr, pluginsUsage)
		default:
			if sub != "" || strings.HasPrefix(a, "-") {
				return usageError(stderr, fmt.Sprintf("plugins: unexpected argument %q", a))
			}
			sub = a
		}
	}
	if sub != "" && sub != "trust" && sub != "untrust" {
		return usageError(stderr, fmt.Sprintf("plugins: unknown subcommand %q; expected trust or untrust", sub))
	}

	root, err := findRootQuietly(opts)
	if err != nil {
		if errors.Is(err, config.ErrNoRoot) {
			return usageError(stderr, err.Error())
		}
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	data, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	specs, err := config.ExternalPlugins(data)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	dir, err := trust.Dir()
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	store := trust.Store{Dir: dir}

	if sub == "untrust" {
		removed, err := store.Revoke(root)
		if err != nil {
			diagnose(stderr, "%v", err)
			return ExitFailure
		}
		if removed {
			return writeOut(stdout, stderr, fmt.Sprintf("Removed the approval for %s; its external plugins will not run until approved again.\n", root))
		}
		return writeOut(stdout, stderr, fmt.Sprintf("%s had no approval to remove.\n", root))
	}

	out := pluginsOutput{Root: root, Status: "none", Plugins: []pluginInfo{}}
	for _, s := range specs {
		out.Plugins = append(out.Plugins, describeProgram(root, s))
	}
	fp, err := trust.Fingerprint(root, specs)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	if len(specs) > 0 {
		st, err := store.Status(root, fp)
		if err != nil {
			diagnose(stderr, "%v", err)
			return ExitFailure
		}
		out.Status = map[trust.Status]string{trust.Approved: "approved", trust.NotApproved: "not_approved", trust.Changed: "changed"}[st]
	}
	out.Approved = out.Status == "approved"

	if sub == "" || len(specs) == 0 || out.Approved {
		if sub == "trust" && len(specs) == 0 {
			return writeOut(stdout, stderr, "This timeline has no external plugins to approve.\n")
		}
		return renderPlugins(stdout, stderr, opts.JSON, out, sub == "trust")
	}

	// trust: show what will run, then confirm.
	if !opts.JSON {
		if c := writeOut(stdout, stderr, pluginsText(out)); c != ExitOK {
			return c
		}
	}
	if !yes {
		if !stdinIsTerminal() {
			return usageError(stderr, "plugins trust: review the programs above, then approve them with 'tline plugins trust --yes'.\nIf you are an agent, ask your user to review and approve them; do not approve plugins yourself.")
		}
		if c := writeOut(stdout, stderr, "Allow tline to run these programs for this timeline? [y/N] "); c != ExitOK {
			return c
		}
		answer, _ := bufio.NewReader(stdin).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			_ = writeOut(stdout, stderr, "Not approved.\n")
			return ExitFailure
		}
	}
	if err := store.Approve(root, fp); err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	out.Status, out.Approved = "approved", true
	if opts.JSON {
		return writeJSON(stdout, stderr, out)
	}
	return writeOut(stdout, stderr, fmt.Sprintf("Approved %d external %s for %s. They run until their settings or programs change.\n", len(specs), plural(len(specs), "plugin", "plugins"), root))
}

// describeProgram finds where a plugin's program is, without running it.
func describeProgram(root string, s config.ExternalPlugin) pluginInfo {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = external.DefaultTimeout
	}
	info := pluginInfo{Name: s.Name, Run: s.Run, Timeout: timeout.String()}
	if strings.ContainsRune(s.Run, '/') {
		path := s.Run
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(path))
		}
		if _, err := os.Stat(path); err == nil {
			info.Program = path
		}
		rel, err := filepath.Rel(root, path)
		info.InTimeline = err == nil && !strings.HasPrefix(rel, "..")
		return info
	}
	if path, err := exec.LookPath(s.Run); err == nil {
		info.Program = path
	}
	return info
}

func pluginsText(out pluginsOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "External plugins in %s:\n", out.Root)
	for _, p := range out.Plugins {
		where := "not found"
		switch {
		case p.Program != "" && p.InTimeline:
			where = "a program inside this timeline: " + p.Program
		case p.Program != "":
			where = "found on PATH at " + p.Program
		}
		fmt.Fprintf(&b, "  %s\n    runs:    %s\n    program: %s\n    timeout: %s\n", p.Name, p.Run, where, p.Timeout)
	}
	return b.String()
}

func renderPlugins(stdout, stderr io.Writer, asJSON bool, out pluginsOutput, trusting bool) int {
	if asJSON {
		return writeJSON(stdout, stderr, out)
	}
	if len(out.Plugins) == 0 {
		return writeOut(stdout, stderr, "This timeline has no external plugins.\n")
	}
	text := pluginsText(out)
	switch out.Status {
	case "approved":
		if trusting {
			text += "These plugins are already approved.\n"
		} else {
			text += "Approved: tline runs these plugins for this timeline.\n"
		}
	case "changed":
		text += "Not approved: the plugin settings or programs changed since you approved them.\nReview them, then run 'tline plugins trust'.\n"
	default:
		text += "Not approved: tline will not run these programs until you approve them.\nReview them, then run 'tline plugins trust'.\n"
	}
	return writeOut(stdout, stderr, text)
}
