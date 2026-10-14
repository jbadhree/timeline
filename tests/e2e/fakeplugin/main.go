// Command fakeplugin is an external tline plugin for end-to-end tests. Its
// behavior is chosen by environment variables, which tline passes through:
//
//	FAKE_MODE      normal (default), crash, hang, badjson, version, misname, foreign
//	FAKE_NAME      the plugin name it describes (default "fake")
//	FAKE_COMMAND   its command name (default "widget")
//	FAKE_TYPE      its type name (default "widget")
//	FAKE_LOG       a file to append each request kind to
//	FAKE_FOREIGN   with FAKE_MODE=foreign, a path to report a problem in
//
// It provides a type with a required number "size", rejects sizes above 10
// in its own type and titles containing "TODO" in types that list it as a
// validator, and has a command whose "list" subcommand lists its events.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type event struct {
	Path       string         `json:"path"`
	Type       string         `json:"type"`
	Fields     map[string]any `json:"fields"`
	FieldLines map[string]int `json:"field_lines"`
}

type request struct {
	Protocol    int              `json:"protocol"`
	Request     string           `json:"request"`
	Now         string           `json:"now"`
	Events      []event          `json:"events"`
	Args        []string         `json:"args"`
	JSON        bool             `json:"json"`
	Diagnostics []map[string]any `json:"diagnostics"`
}

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "fakeplugin: bad request:", err)
		os.Exit(2)
	}
	if log := os.Getenv("FAKE_LOG"); log != "" {
		f, err := os.OpenFile(log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintln(f, req.Request)
			_ = f.Close()
		}
	}
	fmt.Fprintln(os.Stderr, "fakeplugin: handling", req.Request)

	mode := env("FAKE_MODE", "normal")
	switch mode {
	case "crash":
		fmt.Fprintln(os.Stderr, "fakeplugin: crashing on purpose")
		os.Exit(3)
	case "hang":
		time.Sleep(time.Minute)
	case "badjson":
		fmt.Println("{not json")
		return
	}

	typ := env("FAKE_TYPE", "widget")
	var resp any
	switch req.Request {
	case "describe":
		name := env("FAKE_NAME", "fake")
		if mode == "misname" {
			name = "someone-else"
		}
		protocols := []int{1}
		if mode == "version" {
			protocols = []int{2}
		}
		resp = map[string]any{
			"name":      name,
			"protocols": protocols,
			"types": []any{map[string]any{
				"name": typ, "entry_timing": "future",
				"fields": map[string]any{"size": map[string]any{"type": "number", "required": true}},
				"status": map[string]any{"values": []string{"open", "done"}, "closed": []string{"done"}},
			}},
			"validates": true,
			"commands": []any{map[string]any{
				"name": env("FAKE_COMMAND", "widget"), "summary": "list widgets",
				"usage": "Usage:\n  tline " + env("FAKE_COMMAND", "widget") + " list\n",
			}},
		}
	case "validate":
		diags := []map[string]any{}
		for _, e := range req.Events {
			if e.Type == typ {
				if size, ok := e.Fields["size"].(float64); ok && size > 10 {
					diags = append(diags, map[string]any{"path": e.Path, "line": e.FieldLines["size"], "field": "size", "message": "size must be at most 10"})
				}
				continue
			}
			if title, ok := e.Fields["title"].(string); ok && strings.Contains(title, "TODO") {
				diags = append(diags, map[string]any{"path": e.Path, "line": e.FieldLines["title"], "field": "title", "message": "title still says TODO"})
			}
		}
		if mode == "foreign" {
			diags = append(diags, map[string]any{"path": os.Getenv("FAKE_FOREIGN"), "message": "I dislike this file"})
		}
		resp = map[string]any{"diagnostics": diags}
	case "command":
		if len(req.Args) == 0 || req.Args[0] != "list" {
			resp = map[string]any{"usage_error": "expected list"}
			break
		}
		var paths []string
		for _, e := range req.Events {
			if e.Type == typ {
				paths = append(paths, e.Path)
			}
		}
		sort.Strings(paths)
		resp = map[string]any{
			"text":       strings.Join(paths, "\n") + "\n",
			"json":       map[string]any{"widgets": paths, "now": req.Now},
			"incomplete": len(req.Diagnostics) > 0,
		}
	default:
		fmt.Fprintln(os.Stderr, "fakeplugin: unknown request", req.Request)
		os.Exit(4)
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		os.Exit(5)
	}
}
