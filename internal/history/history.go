// Package history stores copies of earlier event versions as plain files:
// history/<type folder>/<event id>/v1.md, v2.md, ... and a versions.log with
// one line per copy. Copies are exact, numbered from 1, and never overwritten.
//
// The package only reads and writes files; core decides what a change means,
// and tline check decides when to save. Callers that save hold the root's
// writer lock.
//
// Owning goal: goals/event-history.md.
package history

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Folder is the history folder under the root.
const Folder = "history"

// LogName is the file beside the copies that records when and why each was
// saved.
const LogName = "versions.log"

// Why a copy was saved.
const (
	Recorded       = "recorded"        // v1, saved when the draft was stamped
	Changed        = "changed"         // a later version
	HistoryEnabled = "history-enabled" // v1 of an event recorded before its type kept history
	Late           = "late"            // v1 saved by a later check, after an interrupted one
)

// Version describes one saved copy.
type Version struct {
	N         int
	SavedAt   time.Time // zero if the log has no line for this copy
	Reason    string    // empty if the log has no line for this copy
	EventPath string    // the event's path when the copy was saved, if logged
}

// Store reads and writes the history of one Timeline root.
type Store struct {
	Root string
}

func (s Store) typeDir(typeFolder string) string {
	return filepath.Join(s.Root, Folder, filepath.FromSlash(typeFolder))
}

func (s Store) eventDir(typeFolder, id string) string {
	return filepath.Join(s.typeDir(typeFolder), id)
}

func copyName(n int) string { return "v" + strconv.Itoa(n) + ".md" }

// Started reports whether the type folder has any history yet.
func (s Store) Started(typeFolder string) (bool, error) {
	_, err := os.Stat(s.typeDir(typeFolder))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// IDs lists the event IDs that have history in the type folder.
func (s Store) IDs(typeFolder string) ([]string, error) {
	entries, err := os.ReadDir(s.typeDir(typeFolder))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// Versions lists an event's copies, oldest first, with their log details.
// An event without history has none.
func (s Store) Versions(typeFolder, id string) ([]Version, error) {
	dir := s.eventDir(typeFolder, id)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	logged, err := readLog(filepath.Join(dir, LogName))
	if err != nil {
		return nil, err
	}
	var out []Version
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "v") || !strings.HasSuffix(name, ".md") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "v"), ".md"))
		if err != nil || n < 1 || copyName(n) != name {
			continue
		}
		v := logged[n]
		v.N = n
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N < out[j].N })
	return out, nil
}

// Read returns copy n of an event.
func (s Store) Read(typeFolder, id string, n int) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.eventDir(typeFolder, id), copyName(n)))
}

// Save stores data as the event's next version and logs it, returning the
// version number. The copy is written to a temporary file and renamed into
// place, and an existing version is never replaced.
func (s Store) Save(typeFolder, id string, data []byte, savedAt time.Time, reason, eventPath string) (int, error) {
	if id == "" || strings.ContainsAny(id, `/\.`) {
		return 0, fmt.Errorf("invalid event id %q", id)
	}
	dir := s.eventDir(typeFolder, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	versions, err := s.Versions(typeFolder, id)
	if err != nil {
		return 0, err
	}
	n := 1
	if len(versions) > 0 {
		n = versions[len(versions)-1].N + 1
	}
	target := filepath.Join(dir, copyName(n))
	if _, err := os.Stat(target); err == nil {
		return 0, fmt.Errorf("%s already exists", target)
	}

	tmp, err := os.CreateTemp(dir, ".v*.tmp")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return 0, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return 0, err
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return 0, err
	}

	logFile, err := os.OpenFile(filepath.Join(dir, LogName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return n, err
	}
	line := fmt.Sprintf("v%d %s %s %s\n", n, savedAt.UTC().Format(time.RFC3339), reason, eventPath)
	if _, err := logFile.WriteString(line); err != nil {
		_ = logFile.Close()
		return n, err
	}
	return n, logFile.Close()
}

// readLog parses versions.log: "vN <time> <reason> <event path>" per line.
// Malformed lines are ignored; the copies themselves are the record.
func readLog(path string) (map[int]Version, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[int]Version{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[int]Version{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), " ", 4)
		if len(parts) != 4 || !strings.HasPrefix(parts[0], "v") {
			continue
		}
		n, err := strconv.Atoi(parts[0][1:])
		if err != nil {
			continue
		}
		at, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			continue
		}
		out[n] = Version{N: n, SavedAt: at, Reason: parts[2], EventPath: parts[3]}
	}
	return out, sc.Err()
}
