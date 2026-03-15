package state_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

func TestFileLockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock := state.NewFileLock(lockPath)
	if err := lock.Acquire(5 * time.Second); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if lock.AcquiredAt.IsZero() {
		t.Error("AcquiredAt should be set after Acquire")
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

func TestFileLockBlocksConcurrent(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock1 := state.NewFileLock(lockPath)
	if err := lock1.Acquire(5 * time.Second); err != nil {
		t.Fatalf("lock1 Acquire: %v", err)
	}
	defer lock1.Release()

	lock2 := state.NewFileLock(lockPath)
	err := lock2.Acquire(200 * time.Millisecond)
	if err == nil {
		lock2.Release()
		t.Fatal("expected lock2 to fail, got nil")
	}

	var lockErr *nd.LockError
	if !errors.As(err, &lockErr) {
		t.Errorf("expected *nd.LockError, got %T: %v", err, err)
	}
}

func TestFileLockReleaseUnlocks(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock1 := state.NewFileLock(lockPath)
	if err := lock1.Acquire(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	lock1.Release()

	lock2 := state.NewFileLock(lockPath)
	if err := lock2.Acquire(1 * time.Second); err != nil {
		t.Fatalf("lock2 should succeed after release: %v", err)
	}
	lock2.Release()
}

func TestFileLockDoubleRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock := state.NewFileLock(lockPath)
	lock.Acquire(5 * time.Second)
	lock.Release()
	// Second release should not panic or error
	if err := lock.Release(); err != nil {
		t.Errorf("double release should be safe: %v", err)
	}
}
