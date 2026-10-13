package cli

import (
	"context"
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
)

// now is the clock; tests replace it.
var now = time.Now

const checkUsage = `Usage:
  tline check --dry-run [paths...]

Validates event files without changing anything. With no paths, every file
in the configured folders is checked; with paths (files or folders), only
those are reported, though rules that span the root, such as duplicate IDs
and goal links, still consider every file.

Stamping is not implemented yet. When it is, a check without --dry-run will
edit draft files, adding their id, entered_at, and entry_timing.

Flags:
  --dry-run   validate only; required until stamping exists
  --json      machine-readable output on stdout
`

// checkOutput is the --json result of check.
type checkOutput struct {
	Root        string             `json:"root"`
	DryRun      bool               `json:"dry_run"`
	Recorded    bool               `json:"recorded"`
	Valid       bool               `json:"valid"`
	Events      []core.EventResult `json:"events"`
	Diagnostics []core.Diagnostic  `json:"diagnostics"`
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

	if !*dryRun {
		return usageError(stderr, "check: stamping drafts is not implemented yet; run 'tline check --dry-run' to validate without changing files")
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

	out := checkOutput{Root: root, DryRun: true, Events: []core.EventResult{}, Diagnostics: []core.Diagnostic{}}
	cfg, err := config.Load(root)
	if err != nil {
		var cerr *config.Error
		if !errors.As(err, &cerr) {
			diagnose(stderr, "reading %s: %v", config.FileName, err)
			return ExitFailure
		}
		out.Diagnostics = cerr.Diagnostics
		return renderCheck(stdout, stderr, opts.JSON, out, config.FileName+" is invalid; no events were checked.")
	}

	docs, err := document.Scan(ctx, root, cfg)
	if err != nil {
		diagnose(stderr, "%v", err)
		return ExitFailure
	}
	selected, err := selectPaths(root, cwd, cfg, docs, paths)
	if err != nil {
		return usageError(stderr, "check: "+err.Error())
	}

	res := core.Check(core.CheckInput{
		Config:   cfg,
		Docs:     docs,
		Selected: selected,
		Now:      now().UTC().Truncate(time.Second),
	})
	out.Events, out.Diagnostics = res.Events, res.Diagnostics
	out.Valid = res.Valid()
	return renderCheck(stdout, stderr, opts.JSON, out, "")
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
	if len(out.Diagnostics) > 0 {
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
	if failure != "" {
		b.WriteString(failure + "\n")
	} else {
		invalid := 0
		drafts := 0
		for _, e := range out.Events {
			if !e.Valid {
				invalid++
			}
			if e.Draft {
				drafts++
			}
		}
		if len(out.Diagnostics) > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "Dry run: checked %d %s (%d %s, %d recorded): ", len(out.Events), plural(len(out.Events), "event", "events"), drafts, plural(drafts, "draft", "drafts"), len(out.Events)-drafts)
		if invalid == 0 && len(out.Diagnostics) == 0 {
			b.WriteString("all valid.")
		} else {
			fmt.Fprintf(&b, "%d with problems, %d %s.", invalid, len(out.Diagnostics), plural(len(out.Diagnostics), "problem", "problems"))
		}
		b.WriteString(" Nothing was recorded.\n")
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
