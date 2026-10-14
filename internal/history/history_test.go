package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveListRead(t *testing.T) {
	s := Store{Root: t.TempDir()}
	at := time.Date(2026, 10, 14, 1, 0, 0, 0, time.UTC)

	if started, _ := s.Started("predictions"); started {
		t.Fatal("started before any copy")
	}
	if vs, err := s.Versions("predictions", "01ABC"); err != nil || len(vs) != 0 {
		t.Fatalf("versions of unknown event = %v, %v", vs, err)
	}

	n, err := s.Save("predictions", "01ABC", []byte("one"), at, Recorded, "predictions/a.md")
	if err != nil || n != 1 {
		t.Fatalf("Save = %d, %v", n, err)
	}
	n, err = s.Save("predictions", "01ABC", []byte("two"), at.Add(time.Hour), Changed, "predictions/moved name.md")
	if err != nil || n != 2 {
		t.Fatalf("Save = %d, %v", n, err)
	}

	vs, err := s.Versions("predictions", "01ABC")
	if err != nil || len(vs) != 2 {
		t.Fatalf("Versions = %+v, %v", vs, err)
	}
	if vs[0].Reason != Recorded || !vs[0].SavedAt.Equal(at) || vs[1].EventPath != "predictions/moved name.md" {
		t.Errorf("versions = %+v", vs)
	}
	if data, _ := s.Read("predictions", "01ABC", 2); string(data) != "two" {
		t.Errorf("v2 = %q", data)
	}
	if started, _ := s.Started("predictions"); !started {
		t.Error("not started after a copy")
	}
	if ids, _ := s.IDs("predictions"); len(ids) != 1 || ids[0] != "01ABC" {
		t.Errorf("IDs = %v", ids)
	}
	entries, _ := os.ReadDir(filepath.Join(s.Root, "history", "predictions", "01ABC"))
	if len(entries) != 3 {
		t.Errorf("files = %v; want v1.md, v2.md, versions.log and no temporary files", entries)
	}
}

func TestCopiesWithoutLogLinesAndNoOverwrite(t *testing.T) {
	s := Store{Root: t.TempDir()}
	dir := filepath.Join(s.Root, "history", "tasks", "01XYZ")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A copy written by a check that stopped before logging it.
	if err := os.WriteFile(filepath.Join(dir, "v1.md"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	vs, err := s.Versions("tasks", "01XYZ")
	if err != nil || len(vs) != 1 || !vs[0].SavedAt.IsZero() || vs[0].Reason != "" {
		t.Fatalf("unlogged copy = %+v, %v", vs, err)
	}
	if n, err := s.Save("tasks", "01XYZ", []byte("two"), time.Now(), Changed, "tasks/t.md"); err != nil || n != 2 {
		t.Fatalf("Save after an unlogged copy = %d, %v", n, err)
	}
	if data, _ := s.Read("tasks", "01XYZ", 1); string(data) != "one" {
		t.Error("v1 was overwritten")
	}
	log, _ := os.ReadFile(filepath.Join(dir, LogName))
	if !strings.HasPrefix(string(log), "v2 ") {
		t.Errorf("log = %q", log)
	}
}

func TestSaveRejectsUnsafeIDs(t *testing.T) {
	s := Store{Root: t.TempDir()}
	for _, id := range []string{"", "../x", "a/b", "."} {
		if _, err := s.Save("tasks", id, []byte("x"), time.Now(), Recorded, "tasks/t.md"); err == nil {
			t.Errorf("Save accepted id %q", id)
		}
	}
}
