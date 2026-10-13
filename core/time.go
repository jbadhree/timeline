package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ParseTimestamp parses an RFC 3339 timestamp with an explicit offset and
// whole-second precision.
func ParseTimestamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, errors.New("must be an RFC 3339 timestamp with an explicit offset, such as 2026-10-01T13:00:00Z")
	}
	// time.Parse accepts fractional seconds even though the layout omits them.
	if i := strings.IndexByte(s, 'T'); i >= 0 && strings.Contains(s[i:], ".") {
		return time.Time{}, errors.New("must have whole-second precision")
	}
	return t.UTC(), nil
}

// CheckTiming checks event time e against entry time a under rule r.
// maxDelay applies only to RealTime.
func CheckTiming(r TimingRule, e, a time.Time, maxDelay time.Duration) error {
	switch r {
	case Past:
		if !e.Before(a) {
			return fmt.Errorf("a %s event must happen before it is entered: event time %s is not before entry time %s", r, fmtTime(e), fmtTime(a))
		}
	case Future:
		if !e.After(a) {
			return fmt.Errorf("a %s event must be entered before it happens: event time %s is not after entry time %s", r, fmtTime(e), fmtTime(a))
		}
	case RealTime:
		d := a.Sub(e)
		if d < 0 {
			return fmt.Errorf("a %s event cannot be entered before it happens: event time %s is after entry time %s", r, fmtTime(e), fmtTime(a))
		}
		if d > maxDelay {
			return fmt.Errorf("a %s event must be entered within %s of happening, but was entered %s later", r, maxDelay, d)
		}
	default:
		return fmt.Errorf("unknown timing rule %q", r)
	}
	return nil
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ValidULID reports whether s is a canonical 26-character ULID.
func ValidULID(s string) bool {
	if len(s) != 26 || s[0] > '7' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(crockford, s[i]) < 0 {
			return false
		}
	}
	return true
}
