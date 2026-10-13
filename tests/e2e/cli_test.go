package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var tlineBin string

func TestMain(m *testing.M) {
	os.Exit(buildAndRun(m))
}

func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "tline-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(os.Stderr, "removing build dir:", err)
		}
	}()

	tlineBin = filepath.Join(dir, "tline")
	build := exec.Command("go", "build", "-o", tlineBin, "github.com/jbadhree/timeline/cmd/tline")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building tline: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

type result struct {
	code           int
	stdout, stderr string
}

func tline(t *testing.T, dir string, args ...string) result {
	t.Helper()
	return tlineEnv(t, dir, nil, args...)
}

// tlineEnv runs tline with extra environment variables such as "NAME=value".
func tlineEnv(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, tlineBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running tline %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	return result{code, out.String(), errOut.String()}
}

func TestHelp(t *testing.T) {
	r := tline(t, t.TempDir(), "--help")
	if r.code != 0 || !strings.Contains(r.stdout, "Usage:") || r.stderr != "" {
		t.Fatalf("got %+v", r)
	}
}

func TestVersionJSON(t *testing.T) {
	r := tline(t, t.TempDir(), "--json", "--version")
	if r.code != 0 {
		t.Fatalf("got %+v", r)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &v); err != nil || v.Version == "" {
		t.Fatalf("stdout %q is not version JSON: %v", r.stdout, err)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{{}, {"frobnicate"}, {"--nope"}, {"--root", "some dir"}} {
		r := tline(t, t.TempDir(), args...)
		if r.code != 2 {
			t.Errorf("%v: exit %d, want 2 (%+v)", args, r.code, r)
		}
		if r.stdout != "" {
			t.Errorf("%v: usage errors must leave stdout empty, got %q", args, r.stdout)
		}
		if !strings.Contains(r.stderr, "tline --help") {
			t.Errorf("%v: stderr lacks help hint: %q", args, r.stderr)
		}
	}
}
