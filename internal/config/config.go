// Package config finds the Timeline root and decodes timeline.yaml into a
// validated core.Config.
//
// Decoding is strict: unknown keys, wrong value types, conflicting folders,
// and redefined shared or plugin-template fields are errors, reported together as
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
	"github.com/jbadhree/timeline/internal/history"
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
// plugins provide type templates; see Decode.
func Load(root string, plugins []core.Plugin) (*core.Config, error) {
	return LoadWith(root, plugins, nil)
}

// LoadWith is Load with a Resolver for external plugins; see DecodeWith.
func LoadWith(root string, plugins []core.Plugin, resolve Resolver) (*core.Config, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, err
	}
	return DecodeWith(data, plugins, resolve)
}

// ExternalPlugin is an external plugin entry in timeline.yaml:
// "- name: { run: ./tools/plugin, timeout: 10s }".
type ExternalPlugin struct {
	Name    string
	Run     string        // a path relative to the root, or a program on PATH
	Timeout time.Duration // zero means the default
}

// Resolver turns a timeline's external plugin entries into plugins, in the
// same order. It may refuse, for example because the user has not approved
// them; its error is returned by DecodeWith unchanged.
type Resolver func(specs []ExternalPlugin) ([]core.Plugin, error)

// ExternalPlugins returns the external plugin entries of timeline.yaml
// content, as written, without running anything or checking the rest of the
// file. Malformed entries are skipped; Decode reports them.
func ExternalPlugins(data []byte) ([]ExternalPlugin, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	top := doc.Content[0]
	var list *yaml.Node
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value == "plugins" {
			list = top.Content[i+1]
		}
	}
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil, nil
	}
	d := &decoder{available: map[string]core.Plugin{}}
	var out []ExternalPlugin
	for _, item := range list.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		if spec, ok := d.decodeExternal(item); ok {
			out = append(out, spec)
		}
	}
	return out, nil
}

// Decode decodes timeline.yaml content. An invalid file returns *Error.
//
// available are the plugins tline can run. The file's "plugins" list chooses
// which of them this timeline uses; without the list, all of them are used.
// A type with "from: plugin/type", or named like a type of a used plugin, is
// built from that plugin's template: the configuration chooses its folder and
// may add fields, sections, and validators, but cannot set its timing,
// status, or the template's own fields.
func Decode(data []byte, available []core.Plugin) (*core.Config, error) {
	return DecodeWith(data, available, nil)
}

// DecodeWith is Decode for a timeline whose "plugins" list may name external
// plugins. resolve is called once with all of them, before types are built; a
// nil resolve rejects external plugins.
func DecodeWith(data []byte, available []core.Plugin, resolve Resolver) (*core.Config, error) {
	d := &decoder{available: map[string]core.Plugin{}, resolve: resolve}
	for _, p := range available {
		d.available[p.Name()] = p
		d.availableOrder = append(d.availableOrder, p.Name())
	}
	cfg := d.decode(data)
	if d.resolveErr != nil {
		return nil, d.resolveErr
	}
	if len(d.diags) > 0 {
		core.SortDiagnostics(d.diags)
		return nil, &Error{Diagnostics: d.diags}
	}
	return cfg, nil
}

type decoder struct {
	resolve    Resolver
	resolveErr error // a Resolver's refusal, returned as is

	diags          []core.Diagnostic
	available      map[string]core.Plugin
	availableOrder []string
	enabled        []string
	// templates are the types of enabled plugins, by "plugin/type".
	templates map[string]core.TypeDef
	// byName lists the "plugin/type" keys providing each type name, for the
	// short form "goal: { folder: goals }".
	byName map[string][]string
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
	var typesNode, pluginsNode *yaml.Node
	hasVersion := false
	for _, kv := range d.pairs(doc.Content[0], FileName, "version", "name", "plugins", "types") {
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
		case "plugins":
			pluginsNode = v
		case "types":
			typesNode = v
		}
	}
	d.decodePlugins(pluginsNode)
	if d.resolveErr != nil {
		return cfg
	}
	cfg.Plugins = d.enabled
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
	template, fromPlugin, ok := d.typeTemplate(name, v)
	if !ok {
		return core.TypeDef{}, false
	}
	allowed := []string{"folder", "entry_timing", "max_delay", "fields", "sections", "status", "validators", "history", "frozen"}
	if fromPlugin {
		allowed = []string{"from", "folder", "fields", "sections", "validators", "history", "frozen"}
	}
	before := len(d.diags)
	var (
		folder     string
		timing     core.TimingRule
		maxDelay   time.Duration
		hasDelay   bool
		hasKey     = map[string]bool{}
		fields     []core.FieldDef
		sections   []string
		status     *core.StatusDef
		validators []string
		history    bool
		frozen     []string
		frozenNode *yaml.Node
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
		case "validators":
			validators = d.decodeValidators(val, what, template.Plugin)
		case "history":
			if val.Tag != "!!bool" {
				d.errf(val, "%s: history must be true or false", what)
			}
			history = val.Value == "true"
		case "frozen":
			frozenNode = val
			frozen, _ = d.strList(val, what+": frozen")
		}
	}
	if !hasKey["folder"] {
		d.errf(v, "%s: folder is required", what)
	}
	if !fromPlugin {
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
		var reserved []string
		if fromPlugin {
			reserved = append(reserved, core.FieldStatus)
			for _, f := range template.Fields {
				reserved = append(reserved, f.Name)
			}
		}
		fields = d.decodeFields(fieldsNode, what, template.Plugin, reserved, status != nil)
	}
	if len(d.diags) > before {
		return core.TypeDef{}, false
	}
	var t core.TypeDef
	if fromPlugin {
		t = template
		t.Name, t.Base, t.Folder = name, template.Name, folder
		t.Fields = append(append([]core.FieldDef{}, template.Fields...), fields...)
		t.Sections = append(append([]string{}, template.Sections...), sections...)
		t.Validators = validators
	} else {
		t = core.TypeDef{
			Name: name, Folder: folder, Timing: timing, MaxDelay: maxDelay,
			Fields: fields, Sections: sections, Status: status, Validators: validators,
		}
	}
	t.History, t.Frozen = history, frozen
	if frozenNode != nil {
		d.checkFrozen(frozenNode, what, &t)
	}
	if len(d.diags) > before {
		return core.TypeDef{}, false
	}
	return t, true
}

// checkFrozen checks a type's frozen list: it needs history, and each entry
// is one of the type's fields, a shared field other than the stamps, or a
// "## Heading" section.
func (d *decoder) checkFrozen(n *yaml.Node, what string, t *core.TypeDef) {
	if !t.History {
		d.errf(n, "%s: frozen requires history: true", what)
		return
	}
	for _, name := range t.Frozen {
		if heading, ok := strings.CutPrefix(name, "## "); ok {
			if strings.TrimSpace(heading) == "" {
				d.errf(n, "%s: frozen section %q needs a heading", what, name)
			}
			continue
		}
		_, declared := t.Field(name)
		switch {
		case core.IsStamp(name):
			d.errf(n, "%s: %s cannot be listed in frozen; history types already reject changes to stamps", what, name)
		case declared, core.IsSharedField(name), name == core.FieldStatus && t.Status != nil:
		default:
			d.errf(n, "%s: frozen %q is not a field of this type; write sections as \"## Heading\"", what, name)
		}
	}
}

// decodePlugins reads the "plugins" list, or enables every available plugin
// when there is none, then collects the enabled plugins' type templates.
func (d *decoder) decodePlugins(n *yaml.Node) {
	if n == nil {
		d.enabled = append([]string{}, d.availableOrder...)
	} else if n.Kind != yaml.SequenceNode {
		d.errf(n, "plugins must be a list of plugin names")
	} else {
		var external []ExternalPlugin
		for _, item := range n.Content {
			if item.Kind == yaml.MappingNode {
				if spec, ok := d.decodeExternal(item); ok {
					external = append(external, spec)
					d.enabled = append(d.enabled, spec.Name)
				}
				continue
			}
			name, ok := d.str(item, "plugins item")
			switch {
			case !ok:
			case d.available[name] == nil:
				d.errf(item, "unknown plugin %q (available: %s)", name, strings.Join(d.availableOrder, ", "))
			case contains(d.enabled, name):
				d.errf(item, "plugin %q is listed twice", name)
			default:
				d.enabled = append(d.enabled, name)
			}
		}
		if len(external) > 0 && len(d.diags) == 0 {
			if d.resolve == nil {
				d.errf(n, "external plugins are not available here")
				return
			}
			plugins, err := d.resolve(external)
			if err != nil {
				d.resolveErr = err
				return
			}
			for i, p := range plugins {
				d.available[external[i].Name] = p
			}
		}
	}

	d.templates, d.byName = map[string]core.TypeDef{}, map[string][]string{}
	for _, name := range d.enabled {
		tp, ok := d.available[name].(core.TypeProvider)
		if !ok {
			continue
		}
		for _, t := range tp.Types() {
			if !validName(t.Name) || !t.Timing.Valid() {
				d.errf(n, "plugin %s provides type %q without a valid name and timing rule", name, t.Name)
				continue
			}
			if keys := d.byName[t.Name]; len(keys) > 0 {
				d.errf(n, "type %q is provided by both plugin %s and plugin %s; use only one of them", t.Name, d.templates[keys[0]].Plugin, name)
				continue
			}
			t.Plugin = name
			key := name + "/" + t.Name
			d.templates[key] = t
			d.byName[t.Name] = append(d.byName[t.Name], key)
		}
	}
}

// typeTemplate finds the plugin template for a type: the one named by its
// "from" key, or, without one, the enabled plugin type with the same name.
// fromPlugin is false for an ordinary configured type.
func (d *decoder) typeTemplate(name string, v *yaml.Node) (t core.TypeDef, fromPlugin, ok bool) {
	what := "type " + name
	if v.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(v.Content); i += 2 {
			if v.Content[i].Value != "from" {
				continue
			}
			from, good := d.str(v.Content[i+1], what+": from")
			if !good {
				return core.TypeDef{}, false, false
			}
			plugin, _, _ := strings.Cut(from, "/")
			switch t, found := d.templates[from]; {
			case found:
				return t, true, true
			case d.available[plugin] != nil && !contains(d.enabled, plugin):
				d.errf(v.Content[i+1], "%s: plugin %s is not enabled; add it to plugins", what, plugin)
			default:
				d.errf(v.Content[i+1], "%s: from %q does not name a plugin type (use plugin/type, such as goals/goal)", what, from)
			}
			return core.TypeDef{}, false, false
		}
	}
	switch keys := d.byName[name]; len(keys) {
	case 0:
		return core.TypeDef{}, false, true
	case 1:
		return d.templates[keys[0]], true, true
	default:
		var plugins []string
		for _, k := range keys {
			plugins = append(plugins, d.templates[k].Plugin)
		}
		d.errf(v, "%s: type %q is provided by both plugin %s; use from: to choose one", what, name, strings.Join(plugins, " and plugin "))
		return core.TypeDef{}, false, false
	}
}

// decodeValidators reads a type's "validators" list: enabled plugins that
// validate, other than the type's own plugin.
func (d *decoder) decodeValidators(n *yaml.Node, what, own string) []string {
	names, _ := d.strList(n, what+": validators")
	var out []string
	for _, name := range names {
		p := d.available[name]
		switch validates := validatesRules(p); {
		case p == nil:
			d.errf(n, "%s: unknown validator plugin %q", what, name)
		case !contains(d.enabled, name):
			d.errf(n, "%s: plugin %s is not enabled; add it to plugins", what, name)
		case !validates:
			d.errf(n, "%s: plugin %s has no validation rules", what, name)
		case name == own:
			d.errf(n, "%s: plugin %s already validates its own types", what, name)
		default:
			out = append(out, name)
		}
	}
	return out
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
	if c == ".tline" || strings.HasPrefix(c, ".tline/") || c == ".git" || strings.HasPrefix(c, ".git/") || c == history.Folder || strings.HasPrefix(c, history.Folder+"/") {
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

// decodeFields decodes a type's fields. reserved lists the fields of a plugin
// template, which the configuration cannot redefine.
func (d *decoder) decodeFields(n *yaml.Node, what, plugin string, reserved []string, hasStatus bool) []core.FieldDef {
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
		case contains(reserved, name):
			d.errf(k, "%s: cannot redefine the field %q from plugin %s", what, name, plugin)
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

// decodeExternal reads "name: { run: ..., timeout: ... }".
func (d *decoder) decodeExternal(item *yaml.Node) (ExternalPlugin, bool) {
	if len(item.Content) != 2 {
		d.errf(item, "an external plugin is written as - name: { run: ./path/to/program }")
		return ExternalPlugin{}, false
	}
	k, v := item.Content[0], item.Content[1]
	spec := ExternalPlugin{Name: k.Value}
	what := "plugin " + spec.Name
	switch {
	case !validName(spec.Name):
		d.errf(k, "plugin name %q must use lowercase letters, digits, and underscores", spec.Name)
		return spec, false
	case d.available[spec.Name] != nil:
		d.errf(k, "plugin %q is a built-in plugin; give the external plugin another name", spec.Name)
		return spec, false
	case contains(d.enabled, spec.Name):
		d.errf(k, "plugin %q is listed twice", spec.Name)
		return spec, false
	}
	hasRun := false
	for _, kv := range d.pairs(v, what, "run", "timeout") {
		switch kv[0].Value {
		case "run":
			spec.Run, hasRun = d.str(kv[1], what+": run")
		case "timeout":
			if s, ok := d.str(kv[1], what+": timeout"); ok {
				t, err := time.ParseDuration(s)
				if err != nil || t <= 0 {
					d.errf(kv[1], "%s: timeout must be a positive duration such as 10s", what)
				}
				spec.Timeout = t
			}
		}
	}
	if !hasRun {
		d.errf(v, "%s: run is required", what)
		return spec, false
	}
	return spec, true
}

// validatesRules reports whether a plugin has validation rules. External
// plugins say so when described; built-in ones by implementing Validator.
func validatesRules(p core.Plugin) bool {
	if v, ok := p.(interface{ Validates() bool }); ok {
		return v.Validates()
	}
	_, ok := p.(core.Validator)
	return ok
}
