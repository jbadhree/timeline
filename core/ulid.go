package core

import (
	"errors"
	"io"
	"time"
)

// NewULID returns a canonical ULID: 48 bits of Unix milliseconds from t,
// then 80 bits read from entropy. Callers pass crypto/rand.Reader in
// production and a fixed reader in tests.
func NewULID(t time.Time, entropy io.Reader) (string, error) {
	ms := t.UnixMilli()
	if ms < 0 || ms >= 1<<48 {
		return "", errors.New("time is outside the ULID range")
	}
	var b [16]byte
	for i := 0; i < 6; i++ {
		b[i] = byte(uint64(ms) >> (40 - 8*i))
	}
	if _, err := io.ReadFull(entropy, b[6:]); err != nil {
		return "", err
	}
	var hi, lo uint64
	for i := 0; i < 8; i++ {
		hi = hi<<8 | uint64(b[i])
		lo = lo<<8 | uint64(b[8+i])
	}
	// 26 base32 characters hold 130 bits; the first carries the top 3.
	var out [26]byte
	for i := 25; i >= 0; i-- {
		out[i] = crockford[lo&31]
		lo = lo>>5 | hi<<59
		hi >>= 5
	}
	return string(out[:]), nil
}
