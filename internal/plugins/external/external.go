// Package external runs external plugins: programs outside the tline binary,
// in any language, that speak tline's JSON plugin protocol. A Plugin
// implements the same core interfaces as built-in plugins, so the core gives
// it the same facts and the same limits.
//
// tline starts the program once per request, in the timeline's root, writes
// one JSON request to its stdin, reads one JSON response from its stdout,
// and passes its stderr through. The protocol is documented in
// docs/plugins.md; see protocol.go for the encoding.
//
// Owning goal: goals/plugins.md.
package external

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbadhree/timeline/core"
)

// DefaultTimeout is how long a plugin has to answer when its configuration
// names no timeout.
const DefaultTimeout = 10 * time.Second

// maxResponse limits how much output a plugin may return.
const maxResponse = 64 << 20

// Plugin is one external plugin named in a timeline's configuration.
type Plugin struct {
	name    string
	run     string
	timeout time.Duration
	root    string
	stderr  io.Writer
	ctx     context.Context

	described   bool
	describeErr error
	desc        describeResponse
	types       []core.TypeDef
}

// New returns the external plugin called name in the timeline at root. run is
// a path relative to root, or a program name found on PATH. The plugin's
// stderr is written to stderr. ctx bounds every request.
func New(ctx context.Context, name, run string, timeout time.Duration, root string, stderr io.Writer) *Plugin {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Plugin{name: name, run: run, timeout: timeout, root: root, stderr: stderr, ctx: ctx}
}

// Name implements core.Plugin.
func (p *Plugin) Name() string { return p.name }

// Describe asks the plugin what it provides. It runs at most once per Plugin;
// later calls return the first result.
func (p *Plugin) Describe() error {
	if p.described {
		return p.describeErr
	}
	p.described = true
	p.describeErr = p.describe()
	return p.describeErr
}

func (p *Plugin) describe() error {
	var d describeResponse
	if err := p.call(describeRequest{Protocol: Protocol, Request: reqDescribe}, &d); err != nil {
		return err
	}
	supported := false
	for _, v := range d.Protocols {
		supported = supported || v == Protocol
	}
	if !supported {
		return fmt.Errorf("plugin %s supports protocol versions %v, but this tline speaks version %d", p.name, d.Protocols, Protocol)
	}
	if d.Name != p.name {
		return fmt.Errorf("plugin %s describes itself as %q; its name in timeline.yaml must match", p.name, d.Name)
	}
	for _, tj := range d.Types {
		t, err := decodeTemplate(tj)
		if err != nil {
			return fmt.Errorf("plugin %s: %w", p.name, err)
		}
		p.types = append(p.types, t)
	}
	for _, c := range d.Commands {
		if c.Name == "" || strings.HasPrefix(c.Name, "-") || strings.ContainsAny(c.Name, " \t") {
			return fmt.Errorf("plugin %s describes an invalid command name %q", p.name, c.Name)
		}
	}
	p.desc = d
	return nil
}

// Types implements core.TypeProvider with the templates from Describe.
func (p *Plugin) Types() []core.TypeDef { return p.types }

// Validates reports whether the plugin has validation rules.
func (p *Plugin) Validates() bool { return p.desc.Validates }

// Validate implements core.Validator.
func (p *Plugin) Validate(in core.PluginInput) ([]core.Diagnostic, error) {
	if !p.desc.Validates {
		return nil, nil
	}
	var resp validateResponse
	err := p.call(validateRequest{Protocol: Protocol, Request: reqValidate, facts: encodeFacts(p.root, in)}, &resp)
	return resp.Diagnostics, err
}

// Commands implements core.CommandProvider with the commands from Describe.
func (p *Plugin) Commands() []core.Command {
	var out []core.Command
	for _, c := range p.desc.Commands {
		name := c.Name
		out = append(out, core.Command{
			Name: name, Summary: c.Summary, Usage: c.Usage,
			Run: func(_ context.Context, in core.CommandInput) (core.CommandOutput, error) {
				return p.command(name, in)
			},
		})
	}
	return out
}

func (p *Plugin) command(name string, in core.CommandInput) (core.CommandOutput, error) {
	req := commandRequest{
		Protocol: Protocol, Request: reqCommand, facts: encodeFacts(p.root, in.PluginInput),
		Command: name, Args: append([]string{}, in.Args...), JSON: in.JSON, Diagnostics: in.Diagnostics,
	}
	if req.Args == nil {
		req.Args = []string{}
	}
	var resp commandResponse
	if err := p.call(req, &resp); err != nil {
		return core.CommandOutput{}, err
	}
	if resp.UsageError != "" {
		return core.CommandOutput{}, core.UsageError{Msg: resp.UsageError}
	}
	out := core.CommandOutput{Text: resp.Text, Incomplete: resp.Incomplete}
	if len(resp.JSON) > 0 {
		out.JSON = resp.JSON
	}
	return out, nil
}

// program returns the plugin's executable: a path inside the root, or a
// program found on PATH.
func (p *Plugin) program() (string, error) {
	if strings.ContainsRune(p.run, '/') || strings.ContainsRune(p.run, filepath.Separator) {
		path := p.run
		if !filepath.IsAbs(path) {
			path = filepath.Join(p.root, filepath.FromSlash(path))
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("plugin %s: program %s not found", p.name, p.run)
		}
		if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			return "", fmt.Errorf("plugin %s: program %s is not executable", p.name, p.run)
		}
		return path, nil
	}
	path, err := exec.LookPath(p.run)
	if err != nil {
		return "", fmt.Errorf("plugin %s: program %s not found on PATH", p.name, p.run)
	}
	return path, nil
}

// call sends one request and decodes the response into resp.
func (p *Plugin) call(req, resp any) error {
	prog, err := p.program()
	if err != nil {
		return err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(p.ctx, p.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, prog)
	cmd.Dir = p.root
	cmd.Stdin = bytes.NewReader(append(body, '\n'))
	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = p.stderr
	// A plugin's own child processes may keep its output open after it is
	// killed; stop waiting for them shortly after.
	cmd.WaitDelay = time.Second

	err = cmd.Run()
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		return fmt.Errorf("plugin %s did not answer within %s", p.name, p.timeout)
	case p.ctx.Err() != nil:
		return p.ctx.Err()
	case err != nil:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("plugin %s exited with status %d", p.name, exitErr.ExitCode())
		}
		return fmt.Errorf("plugin %s could not run: %w", p.name, err)
	case out.over:
		return fmt.Errorf("plugin %s returned more than %d bytes", p.name, maxResponse)
	}
	if err := json.Unmarshal(out.Bytes(), resp); err != nil {
		return fmt.Errorf("plugin %s returned invalid JSON: %v", p.name, err)
	}
	return nil
}

// limitedBuffer keeps at most maxResponse bytes and records whether more
// arrived.
type limitedBuffer struct {
	bytes.Buffer
	over bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > maxResponse {
		b.over = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
