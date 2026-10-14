package trust

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbadhree/timeline/internal/config"
)

func TestApproveRevokeAndChanges(t *testing.T) {
	root := t.TempDir()
	prog := filepath.Join(root, "tools", "plugin")
	if err := os.MkdirAll(filepath.Dir(prog), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prog, []byte("#!/bin/sh\necho one\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	specs := []config.ExternalPlugin{{Name: "p", Run: "./tools/plugin", Timeout: time.Second}, {Name: "q", Run: "some-tool"}}
	fp, err := Fingerprint(root, specs)
	if err != nil {
		t.Fatal(err)
	}
	s := Store{Dir: filepath.Join(t.TempDir(), "config")}

	if st, _ := s.Status(root, fp); st != NotApproved {
		t.Fatalf("status before approval = %v", st)
	}
	if err := s.Approve(root, fp); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Status(root, fp); st != Approved {
		t.Fatalf("status after approval = %v", st)
	}

	// Order of entries does not matter; settings and program contents do.
	if fp2, _ := Fingerprint(root, []config.ExternalPlugin{specs[1], specs[0]}); fp2 != fp {
		t.Error("fingerprint depends on order")
	}
	changedSettings, _ := Fingerprint(root, []config.ExternalPlugin{{Name: "p", Run: "./tools/plugin", Timeout: 2 * time.Second}, specs[1]})
	if st, _ := s.Status(root, changedSettings); st != Changed {
		t.Errorf("changed timeout: status %v", st)
	}
	if err := os.WriteFile(prog, []byte("#!/bin/sh\necho two\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	changedProgram, _ := Fingerprint(root, specs)
	if st, _ := s.Status(root, changedProgram); st != Changed {
		t.Errorf("changed program: status %v", st)
	}

	if ok, err := s.Revoke(root); !ok || err != nil {
		t.Fatalf("Revoke = %v, %v", ok, err)
	}
	if st, _ := s.Status(root, fp); st != NotApproved {
		t.Errorf("status after revoke = %v", st)
	}
	info, err := os.Stat(s.Dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("config dir mode = %v, %v", info, err)
	}
}

func TestDir(t *testing.T) {
	t.Setenv("TLINE_CONFIG_DIR", "/a/b")
	if d, _ := Dir(); d != "/a/b" {
		t.Errorf("TLINE_CONFIG_DIR: %s", d)
	}
	t.Setenv("TLINE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if d, _ := Dir(); d != "/x/tline" {
		t.Errorf("XDG_CONFIG_HOME: %s", d)
	}
}
