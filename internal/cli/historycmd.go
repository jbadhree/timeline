package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/document"
	"github.com/jbadhree/timeline/internal/history"
)

const historyUsage = `Usage:
  tline history <id|path|name> [--version N] [--json]

Lists the saved versions of an event whose type has history: true, oldest
first, with when and why each was saved and which fields and sections
changed. The event is named by its id, its path, or its file name without
.md. Removed events are found by id.

Flags:
  --version N   print saved version N exactly
  --json        machine-readable output on stdout
`

// historyEntry is one version in tline history output.
type historyEntry struct {
	Version         int      `json:"version"`
	SavedAt         string   `json:"saved_at,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	Path            string   `json:"path,omitempty"`
	ChangedFields   []string `json:"changed_fields"`
	ChangedSections []string `json:"changed_sections"`
}

// historyOutput is the --json result of tline history.
type historyOutput struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Path     string         `json:"path,omitempty"` // the current file; empty if removed
	Removed  bool           `json:"removed"`
	Pending  bool           `json:"pending"` // the current file differs from the latest copy
	Versions []historyEntry `json:"versions"`
}

// historyTarget is an event with history, current or removed.
type historyTarget struct {
	id   string
	typ  *core.TypeDef
	path string // current file, or "" if removed
}

func runHistory(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	asJSON := fs.Bool("json", false, "")
	version := fs.Int("version", 0, "")
	var names []string
	for rest := args; ; {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return writeOut(stdout, stderr, historyUsage)
			}
			return usageError(stderr, "history: "+err.Error())
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		names = append(names, rest[0])
		rest = rest[1:]
	}
	opts.JSON = opts.JSON || *asJSON
	if len(names) != 1 {
		return usageError(stderr, "history: name one event by id, path, or file name")
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
	tl, err := loadTimeline(ctx, root, stderr)
	if err != nil {
		var cerr *config.Error
		if errors.As(err, &cerr) {
			diagnose(stderr, "%v", err)
			return ExitValidation
		}
		return reportLoadError(err, opts.JSON, stdout, stderr)
	}
	cfg := tl.cfg
	docs, _, err := document.Scan(ctx, root, cfg)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	store := history.Store{Root: root}

	target, err := findHistoryTarget(store, cfg, docs, root, cwd, names[0])
	if err != nil {
		return usageError(stderr, "history: "+err.Error())
	}
	versions, err := store.Versions(target.typ.Folder, target.id)
	if err != nil {
		diagnose(stderr, "reading history: %v", err)
		return ExitFailure
	}

	if *version != 0 {
		data, err := store.Read(target.typ.Folder, target.id, *version)
		if errors.Is(err, os.ErrNotExist) {
			return usageError(stderr, fmt.Sprintf("history: %s has no version %d", target.id, *version))
		}
		if err != nil {
			diagnose(stderr, "reading history: %v", err)
			return ExitFailure
		}
		if opts.JSON {
			return writeJSON(stdout, stderr, map[string]any{"id": target.id, "version": *version, "content": string(data)})
		}
		return writeOut(stdout, stderr, string(data))
	}

	out := historyOutput{ID: target.id, Type: target.typ.Name, Path: target.path, Removed: target.path == "", Versions: []historyEntry{}}
	var previous *core.Document
	var latest []byte
	for _, v := range versions {
		data, err := store.Read(target.typ.Folder, target.id, v.N)
		if err != nil {
			diagnose(stderr, "reading history: %v", err)
			return ExitFailure
		}
		doc := document.Parse(target.path, data)
		entry := historyEntry{Version: v.N, Reason: v.Reason, Path: v.EventPath, ChangedFields: []string{}, ChangedSections: []string{}}
		if !v.SavedAt.IsZero() {
			entry.SavedAt = v.SavedAt.UTC().Format(time.RFC3339)
		}
		if previous != nil {
			f, s := core.ChangedParts(previous, &doc)
			entry.ChangedFields, entry.ChangedSections = append(entry.ChangedFields, f...), append(entry.ChangedSections, s...)
		}
		out.Versions = append(out.Versions, entry)
		previous, latest = &doc, data
	}
	if target.path != "" && len(versions) > 0 {
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target.path)))
		if err != nil {
			diagnose(stderr, "reading %s: %v", target.path, err)
			return ExitFailure
		}
		out.Pending = !bytes.Equal(current, latest)
	}
	if opts.JSON {
		return writeJSON(stdout, stderr, out)
	}
	return writeOut(stdout, stderr, renderHistory(out))
}

func writeJSON(stdout, stderr io.Writer, v any) int {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		diagnose(stderr, "encoding result: %v", err)
		return ExitFailure
	}
	return writeOut(stdout, stderr, string(b)+"\n")
}

var reasonText = map[string]string{
	history.Recorded:       "recorded",
	history.Changed:        "changed",
	history.HistoryEnabled: "copied when history was enabled",
	history.Late:           "saved by a later check (the recording check stopped early)",
}

func renderHistory(out historyOutput) string {
	var b strings.Builder
	if out.Removed {
		fmt.Fprintf(&b, "%s (%s): removed; no event file has this id, and its copies remain\n", out.ID, out.Type)
	} else {
		fmt.Fprintf(&b, "%s (%s, id %s)\n", out.Path, out.Type, out.ID)
	}
	if len(out.Versions) == 0 {
		b.WriteString("No saved versions yet; run tline check to save v1.\n")
		return b.String()
	}
	for _, v := range out.Versions {
		when := v.SavedAt
		if when == "" {
			when = "(time unknown)"
		}
		reason := reasonText[v.Reason]
		if reason == "" {
			reason = "saved"
		}
		var changed []string
		changed = append(changed, v.ChangedFields...)
		for _, s := range v.ChangedSections {
			changed = append(changed, "## "+s)
		}
		if v.Version > 1 && v.Reason == history.Changed && len(changed) > 0 {
			reason += ": " + strings.Join(changed, ", ")
		}
		line := fmt.Sprintf("  v%d  %s  %s", v.Version, when, reason)
		if v.Path != "" && v.Path != out.Path {
			line += "  (saved from " + v.Path + ")"
		}
		b.WriteString(line + "\n")
	}
	if out.Pending {
		fmt.Fprintf(&b, "The file differs from v%d; run tline check to save it.\n", out.Versions[len(out.Versions)-1].Version)
	}
	return b.String()
}

// findHistoryTarget resolves an id, path, or file name to an event of a type
// that keeps history. Removed events are found by id in the history folder.
func findHistoryTarget(store history.Store, cfg *core.Config, docs []core.Document, root, cwd, arg string) (historyTarget, error) {
	rel := arg
	if p, err := filepath.Abs(filepath.Join(cwd, arg)); err == nil {
		if r, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(r, "..") {
			rel = filepath.ToSlash(r)
		}
	}
	var found []historyTarget
	seen := map[string]bool{}
	var noHistory *core.TypeDef
	for i := range docs {
		d := &docs[i]
		t, ok := cfg.TypeForPath(d.Path)
		if !ok {
			continue
		}
		id := ""
		if f, ok := d.Field(core.FieldID); ok {
			id = f.Value.Text
		}
		base := filepath.Base(d.Path)
		if arg != id && arg != d.Path && rel != d.Path && arg != base && arg+".md" != base {
			continue
		}
		if !t.History {
			noHistory = t
			continue
		}
		if id == "" {
			return historyTarget{}, fmt.Errorf("%s is a draft; it has no history until tline check records it", d.Path)
		}
		found = append(found, historyTarget{id: id, typ: t, path: d.Path})
		seen[id] = true
	}
	for i := range cfg.Types {
		t := &cfg.Types[i]
		if !t.History || seen[arg] {
			continue
		}
		ids, err := store.IDs(t.Folder)
		if err != nil {
			return historyTarget{}, err
		}
		for _, id := range ids {
			if id == arg {
				found = append(found, historyTarget{id: id, typ: t})
			}
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		if noHistory != nil {
			return historyTarget{}, fmt.Errorf("type %s does not keep history; add history: true to it in %s", noHistory.Name, config.FileName)
		}
		return historyTarget{}, fmt.Errorf("no event with history matches %q", arg)
	}
	var paths []string
	for _, f := range found {
		paths = append(paths, f.path)
	}
	sort.Strings(paths)
	return historyTarget{}, fmt.Errorf("%q matches several events: %s", arg, strings.Join(paths, ", "))
}
