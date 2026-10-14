package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pluginRoot writes a timeline that uses the fake plugin as "fake". extra is
// appended to timeline.yaml's types. The plugin program is copied into the
// timeline as tools/fake, so approval covers its contents.
func pluginRoot(t *testing.T, extraTypes string, files map[string]string) string {
	t.Helper()
	all := map[string]string{
		"timeline.yaml": "version: 1\nplugins:\n  - goals\n  - fake: { run: ./tools/fake, timeout: 2s }\ntypes:\n  widget: { from: fake/widget, folder: widgets }\n" + extraTypes,
	}
	for k, v := range files {
		all[k] = v
	}
	root := writeFiles(t, all)
	prog, err := os.ReadFile(fakePluginBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "fake"), prog, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// env gives each test its own approval store and request log.
func pluginEnv(t *testing.T, extra ...string) ([]string, string) {
	t.Helper()
	log := filepath.Join(t.TempDir(), "requests.log")
	return append([]string{"TLINE_CONFIG_DIR=" + t.TempDir(), "FAKE_LOG=" + log}, extra...), log
}

// trusted is pluginEnv with approval skipped, for tests about the protocol.
func trusted(t *testing.T, extra ...string) ([]string, string) {
	t.Helper()
	e, log := pluginEnv(t, extra...)
	return append(e, "TLINE_TRUST_PLUGINS=1"), log
}

func requests(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Fields(string(b))
}

const widget = "---\ntype: widget\ntimeline_at: 2099-01-01T00:00:00Z\ncreated_by: e2e\nsize: %s\nstatus: open\n---\n"

func widgetFile(size string) string { return strings.Replace(widget, "%s", size, 1) }

func TestExternalPluginValidates(t *testing.T) {
	root := pluginRoot(t, "  note:\n    folder: notes\n    entry_timing: past\n    validators: [fake]\n    fields: { title: { type: string } }\n", map[string]string{
		"widgets/small.md": widgetFile("3"),
		"widgets/big.md":   widgetFile("12"),
		"widgets/bad.md":   strings.Replace(widgetFile("1"), "status: open", "status: lost", 1),
		"notes/n.md":       "---\ntype: note\ntimeline_at: 2026-01-01T00:00:00Z\ncreated_by: e2e\ntitle: TODO write this\n---\n",
	})
	env, log := trusted(t)
	r := tlineEnv(t, root, env, "check", "--dry-run", "--json")
	var out struct {
		Diagnostics []struct {
			Path, Field, Message string
			Line                 int
		} `json:"diagnostics"`
	}
	if r.code != 3 || json.Unmarshal([]byte(r.stdout), &out) != nil {
		t.Fatalf("check: %+v", r)
	}
	var got []string
	for _, d := range out.Diagnostics {
		got = append(got, d.Path+":"+d.Field+":"+d.Message)
	}
	// Core and plugin diagnostics, merged and sorted by path.
	want := []string{
		"notes/n.md:title:title still says TODO",
		"widgets/bad.md:status:is \"lost\"; allowed values are open, done",
		"widgets/big.md:size:size must be at most 10",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("diagnostics =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if reqs := requests(t, log); strings.Join(reqs, " ") != "describe validate" {
		t.Errorf("requests = %v; want one describe and one validate", reqs)
	}
	if !strings.Contains(r.stderr, "fakeplugin: handling validate") {
		t.Errorf("plugin stderr not passed through: %q", r.stderr)
	}
}

func TestExternalPluginCannotJudgeOtherTypes(t *testing.T) {
	root := pluginRoot(t, "  note: { folder: notes, entry_timing: past }\n", map[string]string{
		"notes/n.md": "---\ntype: note\ntimeline_at: 2026-01-01T00:00:00Z\ncreated_by: e2e\n---\n",
	})
	env, _ := trusted(t, "FAKE_MODE=foreign", "FAKE_FOREIGN=notes/n.md")
	r := tlineEnv(t, root, env, "check", "--dry-run")
	if r.code != 3 || !strings.Contains(r.stdout, "plugin fake reported a problem in a file that is not one of its types: I dislike this file") {
		t.Fatalf("got %+v", r)
	}
}

func TestExternalPluginTypeAndStamping(t *testing.T) {
	root := pluginRoot(t, "", map[string]string{"widgets/w.md": widgetFile("4")})
	env, log := trusted(t)
	r := tlineEnv(t, root, env, "check")
	if r.code != 0 || !strings.Contains(r.stdout, "widgets/w.md") || !strings.Contains(r.stdout, "future") {
		t.Fatalf("check: %+v", r)
	}
	// The type comes from the plugin's template: size is required.
	if err := os.WriteFile(filepath.Join(root, "widgets", "x.md"), []byte(strings.Replace(widgetFile("1"), "size: 1\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if r = tlineEnv(t, root, env, "check", "--dry-run"); r.code != 3 || !strings.Contains(r.stdout, `field "size": is required`) {
		t.Fatalf("template rules: %+v", r)
	}
	if reqs := requests(t, log); strings.Count(strings.Join(reqs, " "), "describe") != 2 {
		t.Errorf("requests = %v; want describe once per run", reqs)
	}
}

func TestExternalPluginCommand(t *testing.T) {
	root := pluginRoot(t, "", map[string]string{"widgets/a.md": widgetFile("1"), "widgets/b.md": widgetFile("2")})
	env, log := trusted(t)

	r := tlineEnv(t, root, env, "widget", "list")
	if r.code != 0 || r.stdout != "widgets/a.md\nwidgets/b.md\n" {
		t.Fatalf("text: %+v", r)
	}
	if reqs := requests(t, log); strings.Join(reqs, " ") != "describe validate command" {
		t.Errorf("requests = %v", reqs)
	}

	r = tlineEnv(t, root, env, "widget", "list", "--json")
	var out struct {
		Widgets []string `json:"widgets"`
		Now     string   `json:"now"`
	}
	if r.code != 0 || json.Unmarshal([]byte(r.stdout), &out) != nil || len(out.Widgets) != 2 || !strings.HasSuffix(out.Now, "Z") {
		t.Fatalf("json: %+v", r)
	}

	if r = tlineEnv(t, root, env, "widget", "sideways"); r.code != 2 || !strings.Contains(r.stderr, "widget: expected list") {
		t.Errorf("usage error: %+v", r)
	}
	if r = tlineEnv(t, root, env, "widget", "--help"); r.code != 0 || r.stdout != "Usage:\n  tline widget list\n" {
		t.Errorf("command help: %+v", r)
	}
	if r = tlineEnv(t, root, env, "--help"); !strings.Contains(r.stdout, "widget          list widgets (fake plugin, external)") {
		t.Errorf("tline --help: %s", r.stdout)
	}

	if err := os.WriteFile(filepath.Join(root, "widgets", "big.md"), []byte(widgetFile("50")), 0o644); err != nil {
		t.Fatal(err)
	}
	if r = tlineEnv(t, root, env, "widget", "list"); r.code != 5 || !strings.Contains(r.stderr, "result is incomplete") {
		t.Errorf("incomplete: %+v", r)
	}
}

func TestExternalPluginFailures(t *testing.T) {
	root := pluginRoot(t, "", map[string]string{"widgets/a.md": widgetFile("1")})
	for _, tc := range []struct {
		mode, want string
	}{
		{"crash", "plugin fake exited with status 3"},
		{"hang", "plugin fake did not answer within 2s"},
		{"badjson", "plugin fake returned invalid JSON"},
		{"version", "plugin fake supports protocol versions [2], but this tline speaks version 1"},
		{"misname", `plugin fake describes itself as "someone-else"`},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			env, _ := trusted(t, "FAKE_MODE="+tc.mode)
			r := tlineEnv(t, root, env, "check", "--dry-run")
			if r.code != 1 || !strings.Contains(r.stderr, tc.want) {
				t.Errorf("got %+v", r)
			}
		})
	}
	t.Run("missing program", func(t *testing.T) {
		if err := os.Remove(filepath.Join(root, "tools", "fake")); err != nil {
			t.Fatal(err)
		}
		env, _ := trusted(t)
		r := tlineEnv(t, root, env, "check", "--dry-run")
		if r.code != 1 || !strings.Contains(r.stderr, "plugin fake: program ./tools/fake not found") {
			t.Errorf("got %+v", r)
		}
	})
}

func TestExternalPluginClashes(t *testing.T) {
	for _, tc := range []struct {
		name, yaml, env string
		code            int
		want            string
	}{
		{"built-in plugin name", "  - goals: { run: ./tools/fake }\n", "", 3, `plugin "goals" is a built-in plugin`},
		{"core command", "  - fake: { run: ./tools/fake }\n", "FAKE_COMMAND=check", 3, `plugin fake registers command "check", which is a core command`},
		{"another plugin's command", "  - goals\n  - fake: { run: ./tools/fake }\n", "FAKE_COMMAND=goal", 3, `command "goal" is registered by both plugin goals and plugin fake`},
		{"another plugin's type", "  - goals\n  - fake: { run: ./tools/fake }\n", "FAKE_TYPE=task", 3, `type "task" is provided by both plugin goals and plugin fake`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := pluginRoot(t, "", nil)
			yaml := "version: 1\nplugins:\n" + tc.yaml + "types:\n  note: { folder: notes, entry_timing: past }\n"
			if err := os.WriteFile(filepath.Join(root, "timeline.yaml"), []byte(yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			env, _ := trusted(t, tc.env)
			args := []string{"check", "--dry-run"}
			if strings.Contains(tc.want, "command") {
				args = []string{"anything"}
			}
			r := tlineEnv(t, root, env, args...)
			if r.code != tc.code || !strings.Contains(r.stdout+r.stderr, tc.want) {
				t.Errorf("got %+v", r)
			}
		})
	}
}

func TestPluginApproval(t *testing.T) {
	root := pluginRoot(t, "", map[string]string{"widgets/a.md": widgetFile("1")})
	env, log := pluginEnv(t)

	// Not approved: nothing runs, and the message says what to do.
	r := tlineEnv(t, root, env, "check", "--dry-run")
	if r.code != 6 || !strings.Contains(r.stderr, "you have not approved them:\n  fake: ./tools/fake") ||
		!strings.Contains(r.stderr, "tline plugins trust") {
		t.Fatalf("unapproved check: %+v", r)
	}
	if reqs := requests(t, log); len(reqs) != 0 {
		t.Fatalf("an unapproved plugin ran: %v", reqs)
	}
	r = tlineEnv(t, root, env, "check", "--json")
	var js struct {
		Error   string `json:"error"`
		Plugins []struct{ Name, Run string }
	}
	if r.code != 6 || json.Unmarshal([]byte(r.stdout), &js) != nil || js.Error != "plugins_not_approved" || js.Plugins[0].Name != "fake" {
		t.Fatalf("unapproved --json: %+v", r)
	}
	if r = tlineEnv(t, root, env, "widget", "list"); r.code != 6 {
		t.Errorf("unapproved external command: %+v", r)
	}
	if r = tlineEnv(t, root, env, "--help"); !strings.Contains(r.stdout, "external plugins (fake) are not approved") {
		t.Errorf("help: %s", r.stdout)
	}

	// Review, then approve; without a terminal, approval needs --yes.
	r = tlineEnv(t, root, env, "plugins")
	if r.code != 0 || !strings.Contains(r.stdout, "fake\n    runs:    ./tools/fake\n    program: a program inside this timeline") || !strings.Contains(r.stdout, "Not approved") {
		t.Fatalf("plugins: %+v", r)
	}
	r = tlineEnv(t, root, env, "plugins", "trust")
	if r.code != 2 || !strings.Contains(r.stderr, "ask your user to review and approve them; do not approve plugins yourself") {
		t.Fatalf("trust without --yes: %+v", r)
	}
	if r = tlineEnv(t, root, env, "plugins", "trust", "--yes"); r.code != 0 || !strings.Contains(r.stdout, "Approved 1 external plugin") {
		t.Fatalf("trust --yes: %+v", r)
	}
	if r = tlineEnv(t, root, env, "check", "--dry-run"); r.code != 0 {
		t.Fatalf("approved check: %+v", r)
	}

	// Changing the program needs approval again.
	f, err := os.OpenFile(filepath.Join(root, "tools", "fake"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("\n")
	_ = f.Close()
	if r = tlineEnv(t, root, env, "check", "--dry-run"); r.code != 6 || !strings.Contains(r.stderr, "changed since you approved them") {
		t.Fatalf("changed program: %+v", r)
	}
	if r = tlineEnv(t, root, env, "plugins", "--json"); !strings.Contains(r.stdout, `"status": "changed"`) {
		t.Errorf("plugins --json: %s", r.stdout)
	}
	tlineEnv(t, root, env, "plugins", "trust", "--yes")
	if r = tlineEnv(t, root, env, "plugins", "untrust"); r.code != 0 || !strings.Contains(r.stdout, "Removed the approval") {
		t.Fatalf("untrust: %+v", r)
	}
	if r = tlineEnv(t, root, env, "check", "--dry-run"); r.code != 6 {
		t.Errorf("after untrust: %+v", r)
	}

	// TLINE_TRUST_PLUGINS=1 skips approval, for CI.
	if r = tlineEnv(t, root, append(env, "TLINE_TRUST_PLUGINS=1"), "check", "--dry-run"); r.code != 0 {
		t.Errorf("TLINE_TRUST_PLUGINS: %+v", r)
	}
}

func TestExamplePlugins(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is needed for the Python example plugin")
	}
	root := copyFixture(t, filepath.Join("..", "..", "..", "examples", "plugins"))
	build := exec.Command("go", "build", "-o", filepath.Join(root, "tools", "tline-incidents"), "github.com/jbadhree/timeline/examples/plugins/incidents")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the Go example plugin: %v\n%s", err, out)
	}
	for _, prog := range []string{"tline-predictions", "tline-incidents"} {
		if err := os.Chmod(filepath.Join(root, "tools", prog), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env, _ := trusted(t)

	if r := tlineEnv(t, root, env, "check", "--dry-run"); r.code != 0 || !strings.Contains(r.stdout, "checked 4 events (0 drafts, 4 recorded): all valid") {
		t.Fatalf("check: %+v", r)
	}
	if r := tlineEnv(t, root, env, "incident", "open"); r.code != 0 || !strings.Contains(r.stdout, "low   2026-10-13  Search is slow for large accounts") {
		t.Errorf("incident open: %+v", r)
	}
	if r := tlineEnv(t, root, env, "prediction", "due", "--json"); r.code != 0 || !strings.Contains(r.stdout, `"due"`) {
		t.Errorf("prediction due: %+v", r)
	}

	// Each plugin's rule that config alone cannot express.
	edit := func(rel, old, new string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, new, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	edit("predictions/q4-revenue.md", "window_start: 2026-10-01T00:00:00Z", "window_start: 2027-01-15T00:00:00Z")
	edit("predictions/launch-date.md", "status: open", "status: correct")
	edit("incident-log/2026-10-13-slow-search.md", "severity: low", "severity: high")
	r := tlineEnv(t, root, env, "check", "--dry-run")
	for _, want := range []string{
		`predictions/q4-revenue.md:9: field "window_start": must be before the prediction's event time`,
		`predictions/launch-date.md: section "Outcome": is required once a prediction is closed`,
		`incident-log/2026-10-13-slow-search.md: section "Follow-up": is required for high-severity incidents`,
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("missing %q in:\n%s", want, r.stdout)
		}
	}
}
