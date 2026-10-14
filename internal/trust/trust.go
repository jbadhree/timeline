// Package trust records which timelines' external plugins the user has
// approved. Approvals live in the user's configuration directory, never in a
// timeline, so a timeline cannot approve its own plugins.
//
// An approval is the timeline's absolute root and a fingerprint of its
// external plugin settings and of every plugin program inside the root. A
// change to either needs approval again.
//
// Owning goal: goals/plugins.md.
package trust

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jbadhree/timeline/internal/config"
)

// Status is whether a timeline's plugins are approved.
type Status int

// Approval states.
const (
	NotApproved Status = iota
	Approved
	Changed // approved before, but the settings or programs changed since
)

// FileName is the approvals file in the configuration directory.
const FileName = "approved-plugins"

// Dir returns the configuration directory: $TLINE_CONFIG_DIR, else
// $XDG_CONFIG_HOME/tline, else ~/.config/tline.
func Dir() (string, error) {
	if d := os.Getenv("TLINE_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "tline"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tline"), nil
}

// Fingerprint identifies a timeline's external plugin settings and the
// contents of the plugin programs that live inside the root. Programs found
// on PATH are identified by name only: they are installed tools, not part
// of the timeline.
func Fingerprint(root string, specs []config.ExternalPlugin) (string, error) {
	sorted := append([]config.ExternalPlugin{}, specs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	// Writes to a hash never fail.
	h := sha256.New()
	for _, s := range sorted {
		_, _ = fmt.Fprintf(h, "plugin %q run %q timeout %s\n", s.Name, s.Run, s.Timeout)
		if !strings.ContainsRune(s.Run, '/') {
			continue
		}
		path := s.Run
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(path))
		}
		data, err := os.ReadFile(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			_, _ = fmt.Fprintf(h, "missing\n")
		case err != nil:
			return "", err
		default:
			sum := sha256.Sum256(data)
			_, _ = fmt.Fprintf(h, "program %x\n", sum)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Store reads and writes approvals in a configuration directory.
type Store struct {
	Dir string
}

func (s Store) file() string { return filepath.Join(s.Dir, FileName) }

// read returns approvals by root.
func (s Store) read() (map[string]string, error) {
	f, err := os.Open(s.file())
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fp, root, ok := strings.Cut(sc.Text(), " ")
		if ok && len(fp) == 64 {
			out[root] = fp
		}
	}
	return out, sc.Err()
}

// Status reports whether root's plugins, with fingerprint fp, are approved.
func (s Store) Status(root, fp string) (Status, error) {
	approved, err := s.read()
	if err != nil {
		return NotApproved, err
	}
	switch got, ok := approved[root]; {
	case !ok:
		return NotApproved, nil
	case got == fp:
		return Approved, nil
	}
	return Changed, nil
}

// Approve records root's plugins with fingerprint fp.
func (s Store) Approve(root, fp string) error {
	approved, err := s.read()
	if err != nil {
		return err
	}
	approved[root] = fp
	return s.write(approved)
}

// Revoke removes root's approval. It reports whether there was one.
func (s Store) Revoke(root string) (bool, error) {
	approved, err := s.read()
	if err != nil {
		return false, err
	}
	if _, ok := approved[root]; !ok {
		return false, nil
	}
	delete(approved, root)
	return true, s.write(approved)
}

func (s Store) write(approved map[string]string) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	roots := make([]string, 0, len(approved))
	for r := range approved {
		roots = append(roots, r)
	}
	sort.Strings(roots)
	var b strings.Builder
	b.WriteString("# Timelines whose external plugins tline may run. Managed by tline plugins trust.\n")
	for _, r := range roots {
		b.WriteString(approved[r] + " " + r + "\n")
	}
	tmp, err := os.CreateTemp(s.Dir, ".approved-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.file())
}
