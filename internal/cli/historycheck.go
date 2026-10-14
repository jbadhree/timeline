package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jbadhree/timeline/core"
	"github.com/jbadhree/timeline/internal/document"
	"github.com/jbadhree/timeline/internal/history"
)

// historyResult describes one copy tline check saved.
type historyResult struct {
	Path    string `json:"path"`
	ID      string `json:"id"`
	Version int    `json:"version"`
	Reason  string `json:"reason"`
}

// loadHistory reads what the core needs to protect history types: the first
// version of every event with history, and the path each ID was last saved
// from.
func loadHistory(store history.Store, cfg *core.Config) (core.History, error) {
	h := core.History{FirstVersions: map[string]*core.Document{}, LastPaths: map[string]string{}}
	for _, t := range cfg.Types {
		if !t.History {
			continue
		}
		ids, err := store.IDs(t.Folder)
		if err != nil {
			return h, err
		}
		for _, id := range ids {
			versions, err := store.Versions(t.Folder, id)
			if err != nil {
				return h, err
			}
			if len(versions) == 0 {
				continue
			}
			if p := versions[len(versions)-1].EventPath; p != "" {
				h.LastPaths[p] = id
			}
			if versions[0].N != 1 {
				continue // v1 is missing; nothing to compare against
			}
			data, err := store.Read(t.Folder, id, 1)
			if err != nil {
				return h, err
			}
			first := document.Parse(t.Folder+"/"+id+"/v1.md", data)
			if first.ParseErr == nil {
				h.FirstVersions[id] = &first
			}
		}
	}
	return h, nil
}

// startedHistory records, before this check writes anything, which history
// types already have a history folder. A recorded event without copies is
// then "history-enabled" in a new folder and "late" in an existing one.
func startedHistory(store history.Store, cfg *core.Config) (map[string]bool, error) {
	started := map[string]bool{}
	for _, t := range cfg.Types {
		if t.History {
			ok, err := store.Started(t.Folder)
			if err != nil {
				return nil, err
			}
			started[t.Folder] = ok
		}
	}
	return started, nil
}

// saveRecordedHistory saves a copy of each selected recorded event of a
// history type whose content differs from its latest copy, or that has no
// copy yet. A non-empty conflict means a file changed after validation.
func saveRecordedHistory(root string, cfg *core.Config, store history.Store, snap document.Snapshot, started map[string]bool, events []core.EventResult, at time.Time) ([]historyResult, string, error) {
	var saved []historyResult
	for _, e := range events {
		t, ok := cfg.TypeByName(e.Type)
		if !ok || !t.History || e.Draft || e.ID == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			return saved, "", err
		}
		if sha256.Sum256(data) != snap[e.Path] {
			return saved, fmt.Sprintf("%s changed after it was validated; its history was not saved. Run tline check again.", e.Path), nil
		}
		versions, err := store.Versions(t.Folder, e.ID)
		if err != nil {
			return saved, "", err
		}
		reason := history.Changed
		if len(versions) == 0 {
			reason = history.HistoryEnabled
			if started[t.Folder] {
				reason = history.Late
			}
		} else {
			latest, err := store.Read(t.Folder, e.ID, versions[len(versions)-1].N)
			if err != nil {
				return saved, "", err
			}
			if bytes.Equal(latest, data) {
				continue
			}
		}
		n, err := store.Save(t.Folder, e.ID, data, at, reason, e.Path)
		if err != nil {
			return saved, "", err
		}
		saved = append(saved, historyResult{Path: e.Path, ID: e.ID, Version: n, Reason: reason})
	}
	return saved, "", nil
}
