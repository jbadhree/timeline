package lock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireExcludesAndReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".tline", "lock")
	release, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), path, 120*time.Millisecond); !errors.Is(err, ErrHeld) {
		t.Fatalf("second acquire: want ErrHeld, got %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("lock file still present after release: %v", err)
	}
	release2, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	_ = release2()
}

func TestAcquireWaitsForRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	release, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		r, err := Acquire(context.Background(), path, 5*time.Second)
		if err == nil {
			err = r()
		}
		done <- err
	}()
	time.Sleep(2 * pollInterval) // let the waiter find the lock held
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("waiter: %v", err)
	}
}

func TestAcquireCancelled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	release, err := Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = release() }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Acquire(ctx, path, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
