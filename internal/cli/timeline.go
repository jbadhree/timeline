package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/plugins/external"
	"github.com/jbadhree/timeline/internal/trust"
)

// ExitUntrusted is returned when a timeline's external plugins have not been
// approved by the user, or changed since they were.
const ExitUntrusted = 6

// describedPlugins caches external plugins by root and name, so each is
// described at most once per tline run. Run resets it.
var describedPlugins = map[string]*external.Plugin{}

// loaded is a timeline's configuration and the plugins it enables.
type loaded struct {
	root    string
	cfg     *core.Config
	plugins []core.Plugin // enabled plugins, in configuration order
}

// untrustedError means a timeline names external plugins the user has not
// approved, or whose settings or programs changed since approval.
type untrustedError struct {
	root    string
	specs   []config.ExternalPlugin
	changed bool
}

func (e *untrustedError) Error() string {
	var b strings.Builder
	if e.changed {
		b.WriteString("this timeline's external plugins changed since you approved them; they run programs on your computer:\n")
	} else {
		b.WriteString("this timeline uses external plugins, which run programs on your computer, and you have not approved them:\n")
	}
	for _, s := range e.specs {
		fmt.Fprintf(&b, "  %s: %s\n", s.Name, s.Run)
	}
	fmt.Fprintf(&b, "Review these programs, then run 'tline plugins trust' in %s to allow them.\nRun 'tline plugins' to see what they are.", e.root)
	return b.String()
}

// trustPlugins checks that the user approved a timeline's external plugins.
// TLINE_TRUST_PLUGINS=1 skips the check, for environments such as CI.
func trustPlugins(root string, specs []config.ExternalPlugin) error {
	if os.Getenv("TLINE_TRUST_PLUGINS") == "1" {
		return nil
	}
	dir, err := trust.Dir()
	if err != nil {
		return err
	}
	fp, err := trust.Fingerprint(root, specs)
	if err != nil {
		return err
	}
	switch st, err := (trust.Store{Dir: dir}).Status(root, fp); {
	case err != nil:
		return err
	case st == trust.Approved:
		return nil
	case st == trust.Changed:
		return &untrustedError{root: root, specs: specs, changed: true}
	}
	return &untrustedError{root: root, specs: specs}
}

// loadTimeline reads root's configuration. External plugins must be
// approved; each is started and described before types are built.
func loadTimeline(ctx context.Context, root string, stderr io.Writer) (*loaded, error) {
	resolved := map[string]core.Plugin{}
	resolve := func(specs []config.ExternalPlugin) ([]core.Plugin, error) {
		if err := trustPlugins(root, specs); err != nil {
			return nil, err
		}
		var out []core.Plugin
		for _, s := range specs {
			key := root + "\x00" + s.Name
			p := describedPlugins[key]
			if p == nil {
				p = external.New(ctx, s.Name, s.Run, s.Timeout, root, stderr)
				describedPlugins[key] = p
			}
			if err := p.Describe(); err != nil {
				return nil, err
			}
			resolved[s.Name] = p
			out = append(out, p)
		}
		return out, nil
	}
	cfg, err := config.LoadWith(root, plugins(), resolve)
	if err != nil {
		return nil, err
	}
	l := &loaded{root: root, cfg: cfg}
	byName := map[string]core.Plugin{}
	for _, p := range plugins() {
		byName[p.Name()] = p
	}
	for name, p := range resolved {
		byName[name] = p
	}
	for _, name := range cfg.Plugins {
		if p, ok := byName[name]; ok {
			l.plugins = append(l.plugins, p)
		}
	}
	return l, nil
}

// reportLoadError reports an error from loadTimeline that is not a
// configuration error, returning the exit code: 6 for unapproved plugins,
// with a JSON explanation when asJSON, and 1 for anything else, such as a
// plugin that failed to describe itself.
func reportLoadError(err error, asJSON bool, stdout, stderr io.Writer) int {
	var uerr *untrustedError
	if !errors.As(err, &uerr) {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	if asJSON {
		type plugin struct {
			Name    string `json:"name"`
			Run     string `json:"run"`
			Timeout string `json:"timeout,omitempty"`
		}
		out := struct {
			Error   string   `json:"error"`
			Root    string   `json:"root"`
			Changed bool     `json:"changed"`
			Plugins []plugin `json:"plugins"`
		}{Error: "plugins_not_approved", Root: uerr.root, Changed: uerr.changed}
		for _, s := range uerr.specs {
			p := plugin{Name: s.Name, Run: s.Run}
			if s.Timeout > 0 {
				p.Timeout = s.Timeout.String()
			}
			out.Plugins = append(out.Plugins, p)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		_ = writeOut(stdout, stderr, string(b)+"\n")
	}
	diagnose(stderr, "%v", err)
	return ExitUntrusted
}
