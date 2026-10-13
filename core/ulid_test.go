package core

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewULID(t *testing.T) {
	zero := bytes.NewReader(make([]byte, 10))
	got, err := NewULID(time.UnixMilli(0), zero)
	if err != nil || got != strings.Repeat("0", 26) {
		t.Errorf("NewULID(epoch, zeros) = %q, %v", got, err)
	}

	ones := bytes.NewReader(bytes.Repeat([]byte{0xff}, 10))
	got, err = NewULID(time.UnixMilli(1<<48-1), ones)
	if err != nil || got != "7"+strings.Repeat("Z", 25) {
		t.Errorf("NewULID(max, ones) = %q, %v", got, err)
	}

	// The spec's example time prefix: 1469918176385 ms encodes as 01ARYZ6S41.
	got, err = NewULID(time.UnixMilli(1469918176385), bytes.NewReader(make([]byte, 10)))
	if err != nil || !strings.HasPrefix(got, "01ARYZ6S41") || !ValidULID(got) {
		t.Errorf("NewULID(spec time) = %q, %v", got, err)
	}

	early, _ := NewULID(time.UnixMilli(1000), bytes.NewReader(bytes.Repeat([]byte{0xff}, 10)))
	late, _ := NewULID(time.UnixMilli(1001), bytes.NewReader(make([]byte, 10)))
	if early >= late {
		t.Errorf("ULIDs do not sort by time: %s >= %s", early, late)
	}

	if _, err := NewULID(time.UnixMilli(0), bytes.NewReader(nil)); err == nil {
		t.Error("want an error when entropy runs out")
	}
}
