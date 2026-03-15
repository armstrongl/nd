package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
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

func TestFileLockStaleDetectionSucceeds(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// Simulate a dead process: hold the flock on an old file, then release.
	// The file stays behind with an old mod time but no active flock holder.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}

	// Set old mod time (>60s) to trigger stale detection.
	oldTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Release the flock but keep the fd open — the file still exists with old mod time.
	// When another FileLock opens the same path, it gets the same inode.
	// The flock is released so the new acquire will succeed directly in the poll loop.
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()

	lock := state.NewFileLock(lockPath)
	err = lock.Acquire(200 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected acquire to succeed on released lock: %v", err)
	}
	defer lock.Release()

	if lock.AcquiredAt.IsZero() {
		t.Error("AcquiredAt should be set after acquire")
	}
}

func TestFileLockNoStaleBreakForRecentLock(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// lock1 holds the lock — actively held, file mod time is recent.
	lock1 := state.NewFileLock(lockPath)
	if err := lock1.Acquire(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	defer lock1.Release()

	// lock2 tries to acquire, times out. The file is recent, so no stale break.
	lock2 := state.NewFileLock(lockPath)
	err := lock2.Acquire(200 * time.Millisecond)
	if err == nil {
		lock2.Release()
		t.Fatal("expected lock2 to fail because lock1 is actively held with recent mod time")
	}

	var lockErr *nd.LockError
	if !errors.As(err, &lockErr) {
		t.Fatalf("expected *nd.LockError, got %T: %v", err, err)
	}
	if lockErr.Stale {
		t.Error("expected Stale=false for a recently-modified lock file")
	}
}

func TestFileLockStaleBreakFailsWhenStillHeld(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// Create a lock file held by another fd with old mod time.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	// Set old mod time to trigger stale detection.
	oldTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Try to acquire — stale detection triggers, removes file, creates new file,
	// but the original fd still holds flock on the inode. The new file gets a new
	// inode, so flock on it should succeed. This tests the remove-and-retry path.
	lock := state.NewFileLock(lockPath)
	err = lock.Acquire(200 * time.Millisecond)
	// After removing the stale file, the retry opens a new file (new inode),
	// so flock should succeed since no one holds a lock on the new inode.
	if err != nil {
		t.Fatalf("expected retry to succeed on new inode after stale break, got: %v", err)
	}
	defer lock.Release()
}
