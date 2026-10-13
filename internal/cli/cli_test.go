package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		code, stdout, stderr := run(t, args...)
		if code != ExitOK {
			t.Errorf("%v: exit %d, want %d", args, code, ExitOK)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("%v: stdout lacks usage: %q", args, stdout)
		}
		if stderr != "" {
			t.Errorf("%v: unexpected stderr: %q", args, stderr)
		}
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no command", nil, "no command given"},
		{"unknown command", []string{"frobnicate"}, `unknown command "frobnicate"`},
		{"unknown flag", []string{"--nope"}, "flag provided but not defined"},
		{"root without value", []string{"--root"}, "flag needs an argument"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := run(t, tt.args...)
			if code != ExitUsage {
				t.Errorf("exit %d, want %d", code, ExitUsage)
			}
			if stdout != "" {
				t.Errorf("usage errors must leave stdout empty, got %q", stdout)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr %q lacks %q", stderr, tt.want)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, _ := run(t, "--version")
	if code != ExitOK || !strings.HasPrefix(stdout, "tline ") {
		t.Fatalf("exit %d, stdout %q", code, stdout)
	}
}

func TestRunVersionJSON(t *testing.T) {
	old := Version
	Version = "v9.9.9"
	t.Cleanup(func() { Version = old })

	code, stdout, stderr := run(t, "--json", "--version")
	if code != ExitOK || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	var got versionInfo
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v: %q", err, stdout)
	}
	if got.Version != "v9.9.9" {
		t.Errorf("version %q, want v9.9.9", got.Version)
	}
}

func TestRunVersionWriteFailure(t *testing.T) {
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"--json", "--version"}, failingWriter{}, &errOut)
	if code != ExitFailure {
		t.Errorf("exit %d, want %d", code, ExitFailure)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

var errWrite = &writeError{}

type writeError struct{}

func (*writeError) Error() string { return "write failed" }
