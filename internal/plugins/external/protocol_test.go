package external

import (
	"strings"
	"testing"

	"github.com/jbadhree/timeline/core"
)

func TestDecodeTemplate(t *testing.T) {
	tj := templateJSON{Name: "w", EntryTiming: "future", Fields: map[string]fieldJSON{
		"b": {Type: "number", Required: true}, "a": {Type: "list", Items: "string"},
	}, Status: &statusJSON{Values: []string{"open", "done"}, Closed: []string{"done"}}}
	typ, err := decodeTemplate(tj)
	if err != nil || len(typ.Fields) != 2 || typ.Fields[0].Name != "a" || !typ.Fields[1].Required || !typ.Status.IsClosed("done") {
		t.Fatalf("decodeTemplate = %+v, %v", typ, err)
	}
	for _, bad := range []struct {
		tj   templateJSON
		want string
	}{
		{templateJSON{Name: "w", EntryTiming: "future", Fields: map[string]fieldJSON{"x": {Type: "colour"}}}, "unknown type"},
		{templateJSON{Name: "w", EntryTiming: "future", Fields: map[string]fieldJSON{"x": {Type: "list"}}}, "needs items"},
		{templateJSON{Name: "w", EntryTiming: "future", Fields: map[string]fieldJSON{"timeline_at": {Type: "time"}}}, "shared field"},
		{templateJSON{Name: "w", EntryTiming: "real_time"}, "needs max_delay"},
		{templateJSON{Name: "w", EntryTiming: "future", Status: &statusJSON{}}, "status needs values"},
	} {
		if _, err := decodeTemplate(bad.tj); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%+v: error %v, want %q", bad.tj, err, bad.want)
		}
	}
}

func TestEncodeValue(t *testing.T) {
	list := core.RawValue{Kind: core.ValueList, Items: []core.RawValue{{Kind: core.ValueInt, Text: "3"}, {Kind: core.ValueString, Text: "x"}}}
	for _, tc := range []struct {
		in   core.RawValue
		want any
	}{
		{core.RawValue{Kind: core.ValueString, Text: "hi"}, "hi"},
		{core.RawValue{Kind: core.ValueInt, Text: "42"}, int64(42)},
		{core.RawValue{Kind: core.ValueFloat, Text: "0.5"}, 0.5},
		{core.RawValue{Kind: core.ValueBool, Text: "true"}, true},
		{core.RawValue{Kind: core.ValueNull}, nil},
		{core.RawValue{Kind: core.ValueTimestamp, Text: "2026-10-14T00:00:00Z"}, "2026-10-14T00:00:00Z"},
	} {
		if got := encodeValue(tc.in); got != tc.want {
			t.Errorf("encodeValue(%+v) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
	if got := encodeValue(list).([]any); len(got) != 2 || got[0] != int64(3) || got[1] != "x" {
		t.Errorf("list = %#v", got)
	}
}
