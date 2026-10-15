package cli

import (
	"os"
	"strings"
	"testing"
)

func TestPluginsTrustAsksInATerminal(t *testing.T) {
	t.Setenv("TLINE_CONFIG_DIR", t.TempDir())
	root := writeRoot(t, map[string]string{})
	if err := os.WriteFile(root+"/timeline.yaml", []byte("version: 1\nplugins:\n  - p: { run: some-program }\ntypes: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldIn, oldTTY := stdin, stdinIsTerminal
	t.Cleanup(func() { stdin, stdinIsTerminal = oldIn, oldTTY })
	stdinIsTerminal = func() bool { return true }

	stdin = strings.NewReader("n\n")
	code, stdout, _ := run3("--root", root, "plugins", "trust")
	if code != ExitFailure || !strings.Contains(stdout, "Allow tline to run these programs for this timeline? [y/N] Not approved.") {
		t.Fatalf("declined: exit %d, %q", code, stdout)
	}
	if _, stdout, _ := run3("--root", root, "plugins"); !strings.Contains(stdout, "Not approved") {
		t.Fatalf("declining approved anyway: %q", stdout)
	}

	stdin = strings.NewReader("yes\n")
	if code, stdout, _ := run3("--root", root, "plugins", "trust"); code != ExitOK || !strings.Contains(stdout, "Approved 1 external plugin") {
		t.Fatalf("accepted: exit %d, %q", code, stdout)
	}
	if _, stdout, _ := run3("--root", root, "plugins"); !strings.Contains(stdout, "Approved: tline runs these plugins") {
		t.Errorf("status after approval: %q", stdout)
	}
}

func TestShownEscapesInvisibleCharacters(t *testing.T) {
	for in, want := range map[string]string{
		"./tools/lint":              "./tools/lint",
		"./tools/plugin with space": "./tools/plugin with space",
		"./tools/ünïcode":           "./tools/ünïcode",
		"./a\x1b[2K\r./b":           `"./a\x1b[2K\r./b"`,
		"./a\u202etxt.exe":          `"./a\u202etxt.exe"`, // right-to-left override
		"./a\nb":                    `"./a\nb"`,
	} {
		if got := shown(in); got != want {
			t.Errorf("shown(%q) = %s, want %s", in, got, want)
		}
	}
}
