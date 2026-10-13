// Package config finds the Timeline root and decodes timeline.yaml into a
// validated core.Config.
//
// Decoding is strict: unknown keys, wrong value types, conflicting folders,
// and redefined shared or built-in fields are errors, reported together as
// core.Diagnostics. Root discovery never falls back to another root after a
// configuration error, and never creates configuration.
//
// Owning goal: goals/events-and-types.md.
package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbadhree/timeline/core"
	"go.yaml.in/yaml/v3"
)

// FileName is the configuration file that marks a Timeline root.
const FileName = "timeline.yaml"

// ErrNoRoot means no timeline.yaml was found.
var ErrNoRoot = errors.New("no " + FileName + " found")

// FindRoot returns the absolute Timeline root. With explicit set, the root is
// that directory (relative to cwd) and must contain timeline.yaml. Otherwise
// cwd and then its parents are searched for the nearest timeline.yaml.
func FindRoot(explicit, cwd string) (string, error) {
	if explicit != "" {
		dir := explicit
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
		dir = filepath.Clean(dir)
		if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("%w in --root %s", ErrNoRoot, dir)
			}
			return "", err
		}
		return dir, nil
	}
	dir := filepath.Clean(cwd)
	for {
		_, err := os.Stat(filepath.Join(dir, FileName))
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w in %s or any parent directory; run tline inside a Timeline root or pass --root", ErrNoRoot, cwd)
		}
		dir = parent
	}
}

// Error reports an invalid timeline.yaml.
type Error struct {
	Diagnostics []core.Diagnostic
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("invalid " + FileName)
	for _, d := range e.Diagnostics {
		b.WriteString("\n  " + d.String())
	}
	return b.String()
}

// Load reads and decodes root/timeline.yaml. An invalid file returns *Error.
func Load(root string) (*core.Config, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, err
	}
	return Decode(data)
}

// Decode decodes timeline.yaml content. An invalid file returns *Error.
func Decode(data []byte) (*core.Config, error) {
	d := &decoder{}
	cfg := d.decode(data)
	if len(d.diags) > 0 {
		core.SortDiagnostics(d.diags)
		return nil, &Error{Diagnostics: d.diags}
	}
	return cfg, nil
}

type decoder struct {
	diags []core.Diagnostic
}

func (d *decoder) errf(n *yaml.Node, format string, args ...any) {
	line := 0
	if n != nil {
		line = n.Line
	}
	d.diags = append(d.diags, core.Diagnostic{Path: FileName, Line: line, Message: fmt.Sprintf(format, args...)})
}

// pairs returns a mapping's key/value pairs, reporting duplicate and unknown
// keys. allowed nil means any key.
func (d *decoder) pairs(n *yaml.Node, what string, allowed ...string) [][2]*yaml.Node {
	if n.Kind != yaml.MappingNode {
		d.errf(n, "%s must be a mapping", what)
		return nil
	}
	var out [][2]*yaml.Node
	seen := map[string]bool{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if seen[k.Value] {
			d.errf(k, "%s: key %q appears more than once", what, k.Value)
			continue
		}
		seen[k.Value] = true
		if allowed != nil && !contains(allowed, k.Value) {
			d.errf(k, "%s: unknown key %q (allowed: %s)", what, k.Value, strings.Join(allowed, ", "))
			continue
		}
		out = append(out, [2]*yaml.Node{k, v})
	}
	return out
}

func (d *decoder) str(n *yaml.Node, what string) (string, bool) {
	if n.Kind != yaml.ScalarNode || n.Tag != "!!str" || strings.TrimSpace(n.Value) == "" {
		d.errf(n, "%s must be a non-empty string", what)
		return "", false
	}
	return n.Value, true
}

func (d *decoder) strList(n *yaml.Node, what string) ([]string, bool) {
	if n.Kind != yaml.SequenceNode {
		d.errf(n, "%s must be a list", what)
		return nil, false
	}
	var out []string
	ok := true
	for _, item := range n.Content {
		s, good := d.str(item, what+" item")
		if !good {
			ok = false
			continue
		}
		if contains(out, s) {
			d.errf(item, "%s: %q is listed twice", what, s)
			ok = false
			continue
		}
		out = append(out, s)
	}
	return out, ok
}

func (d *decoder) decode(data []byte) *core.Config {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		d.errf(nil, "%v", err)
		return nil
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		d.errf(nil, "file is empty")
		return nil
	}
	cfg := &core.Config{}
	var typesNode *yaml.Node
	hasVersion := false
	for _, kv := range d.pairs(doc.Content[0], FileName, "version", "name", "types") {
		k, v := kv[0], kv[1]
		switch k.Value {
		case "version":
			hasVersion = true
			if v.Tag != "!!int" || v.Value != "1" {
				d.errf(v, "version must be 1")
			}
			cfg.Version = 1
		case "name":
			cfg.Name, _ = d.str(v, "name")
		case "types":
			typesNode = v
		}
	}
	if !hasVersion {
		d.errf(doc.Content[0], "version is required")
	}
	if typesNode == nil {
		d.errf(doc.Content[0], "types is required")
		return cfg
	}
	for _, kv := range d.pairs(typesNode, "types") {
		if t, ok := d.decodeType(kv[0], kv[1]); ok {
			cfg.Types = append(cfg.Types, t)
		}
	}
	d.checkFolders(cfg, typesNode)
	return cfg
}

func (d *decoder) decodeType(k, v *yaml.Node) (core.TypeDef, bool) {
	name := k.Value
	what := "type " + name
	if !validName(name) {
		d.errf(k, "type name %q must use lowercase letters, digits, and underscores", name)
		return core.TypeDef{}, false
	}
	builtin := core.IsBuiltinName(name)
	allowed := []string{"folder", "entry_timing", "max_delay", "fields", "sections", "status"}
	if builtin {
		allowed = []string{"folder", "fields", "sections"}
	}
	before := len(d.diags)
	var (
		folder   string
		timing   core.TimingRule
		maxDelay time.Duration
		hasDelay bool
		hasKey   = map[string]bool{}
		fields   []core.FieldDef
		sections []string
		status   *core.StatusDef
	)
	fieldsNode := (*yaml.Node)(nil)
	for _, kv := range d.pairs(v, what, allowed...) {
		key, val := kv[0], kv[1]
		hasKey[key.Value] = true
		switch key.Value {
		case "folder":
			if s, ok := d.str(val, what+": folder"); ok {
				folder = d.cleanFolder(val, s, what)
			}
		case "entry_timing":
			if s, ok := d.str(val, what+": entry_timing"); ok {
				timing = core.TimingRule(s)
				if !timing.Valid() {
					d.errf(val, "%s: entry_timing must be past, future, or real_time", what)
				}
			}
		case "max_delay":
			hasDelay = true
			if s, ok := d.str(val, what+": max_delay"); ok {
				dur, err := time.ParseDuration(s)
				if err != nil || dur <= 0 {
					d.errf(val, "%s: max_delay must be a positive Go duration such as 15m", what)
				}
				maxDelay = dur
			}
		case "fields":
			fieldsNode = val
		case "sections":
			sections, _ = d.strList(val, what+": sections")
		case "status":
			status = d.decodeStatus(val, what)
		}
	}
	if !hasKey["folder"] {
		d.errf(v, "%s: folder is required", what)
	}
	if !builtin {
		switch {
		case !hasKey["entry_timing"]:
			d.errf(v, "%s: entry_timing is required", what)
		case timing == core.RealTime && !hasDelay:
			d.errf(v, "%s: max_delay is required for real_time types", what)
		case timing != core.RealTime && hasDelay:
			d.errf(v, "%s: max_delay only applies to real_time types", what)
		}
	}
	if fieldsNode != nil {
		fields = d.decodeFields(fieldsNode, what, name, builtin, status != nil)
	}
	if len(d.diags) > before {
		return core.TypeDef{}, false
	}
	if builtin {
		return core.NewBuiltinType(core.Builtin(name), folder, fields, sections), true
	}
	return core.TypeDef{
		Name: name, Folder: folder, Timing: timing, MaxDelay: maxDelay,
		Fields: fields, Sections: sections, Status: status,
	}, true
}

func (d *decoder) cleanFolder(n *yaml.Node, s, what string) string {
	if strings.Contains(s, `\`) || path.IsAbs(s) {
		d.errf(n, "%s: folder must be a relative path using /", what)
		return ""
	}
	c := path.Clean(s)
	if c == "." || c == ".." || strings.HasPrefix(c, "../") {
		d.errf(n, "%s: folder must be a subfolder of the root", what)
		return ""
	}
	if c == ".tline" || strings.HasPrefix(c, ".tline/") || c == ".git" || strings.HasPrefix(c, ".git/") {
		d.errf(n, "%s: folder %q is reserved", what, c)
		return ""
	}
	return c
}

func (d *decoder) decodeStatus(n *yaml.Node, what string) *core.StatusDef {
	st := &core.StatusDef{}
	for _, kv := range d.pairs(n, what+": status", "values", "closed") {
		switch kv[0].Value {
		case "values":
			st.Values, _ = d.strList(kv[1], what+": status values")
		case "closed":
			st.Closed, _ = d.strList(kv[1], what+": status closed")
		}
	}
	if len(st.Values) == 0 {
		d.errf(n, "%s: status values must list at least one value", what)
	}
	for _, c := range st.Closed {
		if !contains(st.Values, c) {
			d.errf(n, "%s: closed status %q is not one of the status values", what, c)
		}
	}
	return st
}

func (d *decoder) decodeFields(n *yaml.Node, what, typeName string, builtin, hasStatus bool) []core.FieldDef {
	var out []core.FieldDef
	for _, kv := range d.pairs(n, what+": fields") {
		k, v := kv[0], kv[1]
		name := k.Value
		fwhat := fmt.Sprintf("%s: field %s", what, name)
		switch {
		case !validName(name):
			d.errf(k, "%s: field name %q must use lowercase letters, digits, and underscores", what, name)
			continue
		case core.IsSharedField(name):
			d.errf(k, "%s: cannot redefine the shared field %q", what, name)
			continue
		case builtin && contains(core.BuiltinFieldNames(core.Builtin(typeName)), name):
			d.errf(k, "%s: cannot redefine the built-in field %q", what, name)
			continue
		case name == core.FieldStatus && hasStatus:
			d.errf(k, "%s: status is defined by the type's status setting", what)
			continue
		}
		fd := core.FieldDef{Name: name}
		hasType := false
		for _, fkv := range d.pairs(v, fwhat, "type", "required", "values", "items") {
			fk, fv := fkv[0], fkv[1]
			switch fk.Value {
			case "type":
				hasType = true
				if s, ok := d.str(fv, fwhat+": type"); ok {
					fd.Kind = core.FieldKind(s)
					if !fd.Kind.Valid() {
						d.errf(fv, "%s: type must be string, number, bool, time, or list", fwhat)
					}
				}
			case "required":
				if fv.Tag != "!!bool" {
					d.errf(fv, "%s: required must be true or false", fwhat)
				}
				fd.Required = fv.Value == "true"
			case "values":
				fd.Values, _ = d.strList(fv, fwhat+": values")
			case "items":
				if s, ok := d.str(fv, fwhat+": items"); ok {
					fd.Items = core.FieldKind(s)
				}
			}
		}
		switch {
		case !hasType:
			d.errf(v, "%s: type is required", fwhat)
		case fd.Kind == core.KindList && (fd.Items == "" || fd.Items == core.KindList || !fd.Items.Valid()):
			d.errf(v, "%s: a list needs items of type string, number, bool, or time", fwhat)
		case fd.Kind != core.KindList && fd.Items != "":
			d.errf(v, "%s: items only applies to lists", fwhat)
		}
		out = append(out, fd)
	}
	return out
}

// checkFolders rejects folders shared by two types or nested in one another.
func (d *decoder) checkFolders(cfg *core.Config, n *yaml.Node) {
	for i, a := range cfg.Types {
		for _, b := range cfg.Types[i+1:] {
			switch {
			case a.Folder == b.Folder:
				d.errf(n, "types %s and %s both use folder %q", a.Name, b.Name, a.Folder)
			case strings.HasPrefix(a.Folder, b.Folder+"/") || strings.HasPrefix(b.Folder, a.Folder+"/"):
				d.errf(n, "folders of types %s (%s) and %s (%s) must not be nested", a.Name, a.Folder, b.Name, b.Folder)
			}
		}
	}
}

func validName(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
