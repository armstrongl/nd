package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
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

	// A stale-aged lock file that is STILL actively flocked by another fd: the
	// mod time is old enough to trip stale detection, but a live holder keeps
	// the flock, so breaking the lock would corrupt state.
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

	// Acquire must NOT break the still-held lock onto a fresh inode. Because the
	// lock file is never unlinked, the second Acquire contends on the same inode
	// as the live holder, so flock keeps it out and Acquire fails with a
	// *nd.LockError (Stale=true) instead of entering the critical section.
	lock := state.NewFileLock(lockPath)
	err = lock.Acquire(200 * time.Millisecond)
	if err == nil {
		lock.Release()
		t.Fatal("expected Acquire to fail while the stale lock is still actively held, got nil")
	}

	var lockErr *nd.LockError
	if !errors.As(err, &lockErr) {
		t.Fatalf("expected *nd.LockError, got %T: %v", err, err)
	}
	if !lockErr.Stale {
		t.Error("expected Stale=true when a stale-aged lock is still actively held")
	}
}

// TestFileLockStaleBreakSerializesConcurrentBreakers is the regression test for
// the unsafe unlink-then-relock race: two holders that both observe the same
// stale (>60s) lock file and both take the stale-break path must never both be
// inside the post-Acquire critical section at once. The old implementation
// os.Remove'd the lock file and relocked a fresh inode, so both breakers
// succeeded; the fix retries on the same inode, so exactly one wins at a time.
func TestFileLockStaleBreakSerializesConcurrentBreakers(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	// A stale-aged lock file with NO active flock holder, so both goroutines
	// reach the stale-break path: whichever loses the initial flock times out
	// while the winner is still in its critical section.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	oldTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(lockPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	var (
		mu         sync.Mutex
		inCritical int
		maxInCrit  int
	)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock := state.NewFileLock(lockPath)
			// Losing the race and returning *nd.LockError is acceptable; both
			// entering the critical section together is the bug under test.
			if err := lock.Acquire(200 * time.Millisecond); err != nil {
				return
			}
			defer lock.Release()

			mu.Lock()
			inCritical++
			if inCritical > maxInCrit {
				maxInCrit = inCritical
			}
			mu.Unlock()

			// Hold long enough that the other goroutine's Acquire times out and
			// takes the stale-break path while this one is still inside.
			time.Sleep(300 * time.Millisecond)

			mu.Lock()
			inCritical--
			mu.Unlock()
		}()
	}
	wg.Wait()

	if maxInCrit != 1 {
		t.Fatalf("mutual exclusion violated: max %d holders inside the critical section at once (want exactly 1)", maxInCrit)
	}
}
