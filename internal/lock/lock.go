// Package lock provides the per-root writer lock that lets only one
// tline check stamp files at a time, across processes.
//
// The lock is a file created with O_EXCL, which works on every supported OS
// and filesystem. A check that crashes leaves the file behind; the error
// names it so a person can remove it once no check is running.
//
// Owning goal: goals/record-events.md.
package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrHeld means another process held the lock for the whole wait.
var ErrHeld = errors.New("lock is held by another tline check")

// pollInterval is how often a waiting process retries.
const pollInterval = 50 * time.Millisecond

// Acquire creates the lock file at path, creating its directory if needed,
// and waits up to wait while another process holds it. The returned release
// function removes the lock.
func Acquire(ctx context.Context, path string, wait time.Duration) (release func() error, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, werr := fmt.Fprintf(f, "pid %d since %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
			cerr := f.Close()
			if werr != nil || cerr != nil {
				_ = os.Remove(path)
				return nil, errors.Join(werr, cerr)
			}
			return func() error { return os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			holder, _ := os.ReadFile(path)
			return nil, fmt.Errorf("%w (%s): %s; if no check is running, delete %s", ErrHeld, path, strings.TrimSpace(string(holder)), path)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
