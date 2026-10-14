// Command incidents is an example external tline plugin, in Go. It provides
// an "incident" type, rules that timeline.yaml alone cannot express, and an
// "incident open" command. Build it with
//
//	go build -o examples/plugins/tools/tline-incidents ./examples/plugins/incidents
//
// See docs/plugins.md for the protocol.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type section struct {
	Heading string `json:"heading"`
}

type event struct {
	Path     string         `json:"path"`
	Type     string         `json:"type"`
	Time     string         `json:"time"`
	Fields   map[string]any `json:"fields"`
	Sections []section      `json:"sections"`
}

type request struct {
	Request     string            `json:"request"`
	Events      []event           `json:"events"`
	Args        []string          `json:"args"`
	Diagnostics []json.RawMessage `json:"diagnostics"`
}

type diagnostic struct {
	Path    string `json:"path"`
	Section string `json:"section,omitempty"`
	Message string `json:"message"`
}

func has(e event, heading string) bool {
	for _, s := range e.Sections {
		if s.Heading == heading {
			return true
		}
	}
	return false
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "incidents:", err)
		os.Exit(1)
	}
	var resp any
	switch req.Request {
	case "describe":
		resp = map[string]any{
			"name":      "incidents",
			"protocols": []int{1},
			"types": []any{map[string]any{
				"name":         "incident",
				"entry_timing": "past",
				"fields": map[string]any{
					"summary":  map[string]any{"type": "string", "required": true},
					"severity": map[string]any{"type": "string", "required": true, "values": []string{"low", "high"}},
				},
				"status": map[string]any{"values": []string{"open", "resolved"}, "closed": []string{"resolved"}},
			}},
			"validates": true,
			"commands": []any{map[string]any{
				"name":    "incident",
				"summary": "list open incidents",
				"usage":   "Usage:\n  tline incident open   list open incidents, high severity first\n",
			}},
		}
	case "validate":
		diags := []diagnostic{}
		for _, e := range req.Events {
			if e.Type != "incident" {
				continue
			}
			if e.Fields["severity"] == "high" && !has(e, "Follow-up") {
				diags = append(diags, diagnostic{Path: e.Path, Section: "Follow-up", Message: "is required for high-severity incidents"})
			}
			if e.Fields["status"] == "resolved" && !has(e, "Resolution") {
				diags = append(diags, diagnostic{Path: e.Path, Section: "Resolution", Message: "is required once an incident is resolved"})
			}
		}
		resp = map[string]any{"diagnostics": diags}
	case "command":
		if len(req.Args) != 1 || req.Args[0] != "open" {
			resp = map[string]any{"usage_error": "expected 'open'"}
			break
		}
		var open []event
		for _, e := range req.Events {
			if e.Type == "incident" && e.Fields["status"] == "open" {
				open = append(open, e)
			}
		}
		sort.Slice(open, func(i, j int) bool {
			if open[i].Fields["severity"] != open[j].Fields["severity"] {
				return open[i].Fields["severity"] == "high"
			}
			return open[i].Time < open[j].Time
		})
		var b strings.Builder
		list := []map[string]any{}
		for _, e := range open {
			fmt.Fprintf(&b, "%-4s  %s  %v  (%s)\n", e.Fields["severity"], e.Time[:10], e.Fields["summary"], e.Path)
			list = append(list, map[string]any{"path": e.Path, "severity": e.Fields["severity"], "summary": e.Fields["summary"]})
		}
		if len(open) == 0 {
			b.WriteString("No open incidents.\n")
		}
		resp = map[string]any{"text": b.String(), "json": map[string]any{"open": list}, "incomplete": len(req.Diagnostics) > 0}
	default:
		fmt.Fprintln(os.Stderr, "incidents: unknown request", req.Request)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		os.Exit(1)
	}
}
