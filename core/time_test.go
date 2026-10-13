package core

import (
	"testing"
	"time"
)

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := ParseTimestamp(s)
	if err != nil {
		t.Fatalf("ParseTimestamp(%q): %v", s, err)
	}
	return v
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"2026-10-01T13:00:00Z", false},
		{"2026-10-01T15:00:00+02:00", false},
		{"2026-10-01T13:00:00.5Z", true}, // fractional seconds
		{"2026-10-01T13:00:00", true},    // no offset
		{"2026-10-01", true},             // date only
		{"yesterday", true},
	}
	for _, tt := range tests {
		_, err := ParseTimestamp(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTimestamp(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
	}
	if got := ts(t, "2026-10-01T15:00:00+02:00"); !got.Equal(ts(t, "2026-10-01T13:00:00Z")) || got.Location() != time.UTC {
		t.Errorf("offset not normalized to UTC: %v", got)
	}
}

func TestCheckTimingBoundaries(t *testing.T) {
	a := ts(t, "2026-10-01T12:00:00Z")
	sec := time.Second
	tests := []struct {
		name  string
		rule  TimingRule
		event time.Time
		delay time.Duration
		ok    bool
	}{
		{"past before entry", Past, a.Add(-sec), 0, true},
		{"past equal to entry", Past, a, 0, false},
		{"past after entry", Past, a.Add(sec), 0, false},
		{"future after entry", Future, a.Add(sec), 0, true},
		{"future equal to entry", Future, a, 0, false},
		{"future before entry", Future, a.Add(-sec), 0, false},
		{"real-time at entry", RealTime, a, 15 * time.Minute, true},
		{"real-time at max delay", RealTime, a.Add(-15 * time.Minute), 15 * time.Minute, true},
		{"real-time past max delay", RealTime, a.Add(-15*time.Minute - sec), 15 * time.Minute, false},
		{"real-time in the future", RealTime, a.Add(sec), 15 * time.Minute, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTiming(tt.rule, tt.event, a, tt.delay)
			if (err == nil) != tt.ok {
				t.Errorf("CheckTiming = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestValidULID(t *testing.T) {
	for s, want := range map[string]bool{
		"01M3VW7KR0FFWVGBE1D5GYN31R": true,
		"01m3vw7kr0ffwvgbe1d5gyn31r": false, // lowercase is not canonical
		"81M3VW7KR0FFWVGBE1D5GYN31R": false, // overflows 48-bit time
		"01M3VW7KR0FFWVGBE1D5GYN31":  false, // too short
		"01M3VW7KR0FFWVGBE1D5GYN3IU": false, // I and U are not Crockford
	} {
		if got := ValidULID(s); got != want {
			t.Errorf("ValidULID(%q) = %v, want %v", s, got, want)
		}
	}
}
