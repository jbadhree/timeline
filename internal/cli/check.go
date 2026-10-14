package cli

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/config"
	"github.com/jbadhree/timeline/internal/document"
	"github.com/jbadhree/timeline/internal/history"
	"github.com/jbadhree/timeline/internal/lock"
	"github.com/jbadhree/timeline/internal/plugins/goals"
)

// Effects that tests replace.
var (
	now         = time.Now
	entropy     = rand.Reader
	lockWait    = 10 * time.Second
	beforeStamp = func(path string) {} // called just before each draft is rewritten
)

const checkUsage = `Usage:
  tline check [--dry-run] [paths...]

Validates event files and stamps every valid draft: it writes id,
entered_at, and entry_timing at the top of the draft's frontmatter and
leaves the rest of the file unchanged. A draft is a file without those
three fields. Nothing is written unless every selected file is valid.

With no paths, every file in the configured folders is checked; with paths
(files or folders), only those are checked and stamped, though rules that
span the root, such as duplicate IDs and links, still consider every file.

Flags:
  --dry-run   validate only; change no files
  --json      machine-readable output on stdout

Environment:
  TLINE_LOCK_WAIT   how long to wait for another check's lock (default 10s)
`

// stampResult describes one draft that was stamped.
type stampResult struct {
	Path        string          `json:"path"`
	ID          string          `json:"id"`
	EnteredAt   string          `json:"entered_at"`
	EntryTiming core.TimingRule `json:"entry_timing"`
}

// checkOutput is the --json result of check.
type checkOutput struct {
	Root        string             `json:"root"`
	DryRun      bool               `json:"dry_run"`
	Recorded    bool               `json:"recorded"` // true when the check stamped a draft or saved a history copy
	Valid       bool               `json:"valid"`
	Events      []core.EventResult `json:"events"`
	Diagnostics []core.Diagnostic  `json:"diagnostics"`
	Stamped     []stampResult      `json:"stamped"`
	// History lists the history copies this check saved.
	History []historyResult `json:"history"`
	// Conflict explains why stamping stopped early, if it did.
	Conflict string `json:"conflict,omitempty"`
}

func runCheck(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "")
	asJSON := fs.Bool("json", false, "")

	// Allow flags after paths, as in "tline check goals/x.md --dry-run".
	var paths []string
	for rest := args; ; {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return writeOut(stdout, stderr, checkUsage)
			}
			return usageError(stderr, "check: "+err.Error())
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		paths = append(paths, rest[0])
		rest = rest[1:]
	}
	opts.JSON = opts.JSON || *asJSON

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

	out := checkOutput{Root: root, DryRun: *dryRun, Events: []core.EventResult{}, Diagnostics: []core.Diagnostic{}, Stamped: []stampResult{}, History: []historyResult{}}

	if !*dryRun {
		wait := lockWait
		if v := os.Getenv("TLINE_LOCK_WAIT"); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil || d < 0 {
				return usageError(stderr, fmt.Sprintf("TLINE_LOCK_WAIT=%q is not a duration such as 10s", v))
			}
			wait = d
		}
		release, err := lock.Acquire(ctx, filepath.Join(root, ".tline", "lock"), wait)
		if err != nil {
			if errors.Is(err, lock.ErrHeld) {
				out.Conflict = err.Error()
				return renderCheck(stdout, stderr, opts.JSON, out, "")
			}
			diagnose(stderr, "taking the writer lock: %v", err)
			return ExitFailure
		}
		defer func() {
			if err := release(); err != nil {
				diagnose(stderr, "releasing the writer lock: %v", err)
			}
		}()
	}

	tl, err := loadTimeline(ctx, root, stderr)
	if err != nil {
		var cerr *config.Error
		if !errors.As(err, &cerr) {
			return reportLoadError(err, opts.JSON, stdout, stderr)
		}
		out.Diagnostics = cerr.Diagnostics
		return renderCheck(stdout, stderr, opts.JSON, out, config.FileName+" is invalid; no events were checked.")
	}
	cfg := tl.cfg

	docs, snap, err := document.Scan(ctx, root, cfg)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	selected, err := selectPaths(root, cwd, cfg, docs, paths)
	if err != nil {
		return usageError(stderr, "check: "+err.Error())
	}

	store := history.Store{Root: root}
	hist, err := loadHistory(store, cfg)
	if err != nil {
		diagnose(stderr, "reading history: %v", err)
		return ExitFailure
	}
	started, err := startedHistory(store, cfg)
	if err != nil {
		diagnose(stderr, "reading history: %v", err)
		return ExitFailure
	}

	in := core.CheckInput{Config: cfg, Docs: docs, Selected: selected, Now: clock(), Plugins: tl.plugins, History: hist}
	res, err := core.Check(in)
	if err == nil && !*dryRun && res.Valid() {
		// The entry time is read after validation, immediately before
		// writing. Recheck against it, so a Future draft whose time passed
		// during validation is not recorded.
		in.Now = clock()
		res, err = core.Check(in)
	}
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	out.Events, out.Diagnostics, out.Valid = res.Events, res.Diagnostics, res.Valid()
	if *dryRun || !out.Valid {
		return renderCheck(stdout, stderr, opts.JSON, out, "")
	}

	for _, e := range res.Events {
		if !e.Draft {
			continue
		}
		stamped, data, conflict, err := stampDraft(root, cfg, snap, e, in.Now)
		if err != nil {
			diagnose(stderr, "stamping %s: %v", e.Path, err)
			renderCheck(stdout, stderr, opts.JSON, out, "")
			return ExitFailure
		}
		if conflict != "" {
			out.Conflict = conflict
			break
		}
		out.Stamped = append(out.Stamped, stamped)
		// Save v1 before going on, so a stopped check loses as little as
		// possible; a missing copy is saved by the next check.
		if t, _ := cfg.TypeByName(e.Type); t.History {
			n, err := store.Save(t.Folder, stamped.ID, data, in.Now, history.Recorded, e.Path)
			if err != nil {
				diagnose(stderr, "saving history for %s: %v", e.Path, err)
				renderCheck(stdout, stderr, opts.JSON, out, "")
				return ExitFailure
			}
			out.History = append(out.History, historyResult{Path: e.Path, ID: stamped.ID, Version: n, Reason: history.Recorded})
		}
	}
	if out.Conflict == "" {
		saved, conflict, err := saveRecordedHistory(root, cfg, store, snap, started, res.Events, in.Now)
		out.History = append(out.History, saved...)
		out.Conflict = conflict
		if err != nil {
			diagnose(stderr, "saving history: %v", err)
			renderCheck(stdout, stderr, opts.JSON, out, "")
			return ExitFailure
		}
	}
	out.Recorded = len(out.Stamped) > 0 || len(out.History) > 0
	return renderCheck(stdout, stderr, opts.JSON, out, "")
}

// plugins returns the built-in plugins tline can run: the goals plugin. A
// timeline may also name external plugins; see loadTimeline.
var plugins = builtinPlugins

func builtinPlugins() []core.Plugin { return []core.Plugin{goals.New()} }

// clock returns the current time at Timeline's whole-second precision.
func clock() time.Time { return now().UTC().Truncate(time.Second) }

// stampDraft writes the stamps into one validated draft and returns the
// stamped content. A non-empty conflict means the file changed after
// validation and was left alone.
func stampDraft(root string, cfg *core.Config, snap document.Snapshot, e core.EventResult, at time.Time) (stampResult, []byte, string, error) {
	t, ok := cfg.TypeByName(e.Type)
	if !ok {
		return stampResult{}, nil, "", fmt.Errorf("unknown type %q", e.Type)
	}
	id, err := core.NewULID(at, entropy)
	if err != nil {
		return stampResult{}, nil, "", err
	}
	path := filepath.Join(root, filepath.FromSlash(e.Path))
	data, err := os.ReadFile(path)
	if err != nil {
		return stampResult{}, nil, "", err
	}
	stamp := document.Stamp{ID: id, EnteredAt: at, EntryTiming: t.Timing}
	updated, err := document.InsertStamp(data, stamp)
	if err != nil {
		return stampResult{}, nil, "", err
	}
	beforeStamp(e.Path)
	if err := document.Rewrite(path, snap[e.Path], updated); err != nil {
		if errors.Is(err, document.ErrChanged) {
			return stampResult{}, nil, fmt.Sprintf("%s changed after it was validated; it was not stamped. Run tline check again.", e.Path), nil
		}
		return stampResult{}, nil, "", err
	}
	return stampResult{Path: e.Path, ID: id, EnteredAt: at.Format(time.RFC3339), EntryTiming: t.Timing}, updated, "", nil
}

// selectPaths turns path arguments (relative to cwd) into the root-relative
// event paths they select. Nil means every file.
func selectPaths(root, cwd string, cfg *core.Config, docs []core.Document, args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	selected := []string{}
	for _, arg := range args {
		abs := arg
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(cwd, abs)
		}
		rel, err := filepath.Rel(root, filepath.Clean(abs))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s is outside the Timeline root %s", arg, root)
		}
		rel = filepath.ToSlash(rel)
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", arg, errors.Unwrap(err))
		}
		inFolder := false
		for _, t := range cfg.Types {
			if rel == t.Folder || strings.HasPrefix(rel, t.Folder+"/") || (info.IsDir() && (rel == "." || strings.HasPrefix(t.Folder, rel+"/"))) {
				inFolder = true
				break
			}
		}
		if !inFolder {
			return nil, fmt.Errorf("%s is not inside a configured type folder", arg)
		}
		for _, d := range docs {
			if d.Path == rel || rel == "." || strings.HasPrefix(d.Path, rel+"/") {
				selected = append(selected, d.Path)
			}
		}
	}
	return selected, nil
}

func renderCheck(stdout, stderr io.Writer, asJSON bool, out checkOutput, failure string) int {
	code := ExitOK
	switch {
	case out.Conflict != "":
		code = ExitConflict
	case len(out.Diagnostics) > 0:
		code = ExitValidation
	}
	if asJSON {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			diagnose(stderr, "encoding result: %v", err)
			return ExitFailure
		}
		if c := writeOut(stdout, stderr, string(b)+"\n"); c != ExitOK {
			return c
		}
		return code
	}

	var b strings.Builder
	for _, d := range out.Diagnostics {
		b.WriteString(d.String() + "\n")
	}
	if len(out.Diagnostics) > 0 {
		b.WriteString("\n")
	}
	switch {
	case failure != "":
		b.WriteString(failure + "\n")
	case out.Conflict != "" && len(out.Events) == 0:
		b.WriteString("Conflict: " + out.Conflict + "\n")
	default:
		invalid, drafts := 0, 0
		for _, e := range out.Events {
			if !e.Valid {
				invalid++
			}
			if e.Draft {
				drafts++
			}
		}
		prefix := "Checked"
		if out.DryRun {
			prefix = "Dry run: checked"
		}
		fmt.Fprintf(&b, "%s %d %s (%d %s, %d recorded): ", prefix, len(out.Events), plural(len(out.Events), "event", "events"), drafts, plural(drafts, "draft", "drafts"), len(out.Events)-drafts)
		if len(out.Diagnostics) == 0 {
			b.WriteString("all valid.")
		} else {
			fmt.Fprintf(&b, "%d with problems, %d %s.", invalid, len(out.Diagnostics), plural(len(out.Diagnostics), "problem", "problems"))
		}
		switch {
		case out.DryRun:
			b.WriteString(" Nothing was written.\n")
		case len(out.Diagnostics) > 0:
			b.WriteString(" Nothing was written.\n")
		case len(out.Stamped) == 0 && out.Conflict == "":
			b.WriteString(" No drafts to stamp.\n")
		default:
			fmt.Fprintf(&b, " Stamped %d %s:\n", len(out.Stamped), plural(len(out.Stamped), "draft", "drafts"))
			for _, st := range out.Stamped {
				fmt.Fprintf(&b, "  %s  %s  %s\n", st.Path, st.ID, st.EntryTiming)
			}
			if len(out.Stamped) > 0 {
				fmt.Fprintf(&b, "Entered at %s.\n", out.Stamped[0].EnteredAt)
			}
		}
		if len(out.History) > 0 {
			fmt.Fprintf(&b, "Saved %d history %s:\n", len(out.History), plural(len(out.History), "copy", "copies"))
			for _, h := range out.History {
				fmt.Fprintf(&b, "  %s  v%d  %s\n", h.Path, h.Version, h.Reason)
			}
		}
		if out.Conflict != "" {
			b.WriteString("Conflict: " + out.Conflict + "\n")
		}
	}
	if c := writeOut(stdout, stderr, b.String()); c != ExitOK {
		return c
	}
	return code
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
