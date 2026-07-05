package state

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/armstrongl/nd/internal/nd"
)

// FileLock provides advisory file locking on deployments.yaml.
// Acquired before read-modify-write cycles, released after the atomic rename.
type FileLock struct {
	Path       string
	AcquiredAt time.Time
	file       *os.File
}

// NewFileLock creates a FileLock for the given path.
func NewFileLock(path string) *FileLock {
	return &FileLock{Path: path}
}

// staleLockThreshold is the age beyond which a lock file is considered stale.
const staleLockThreshold = 60 * time.Second

// staleBreakTimeout bounds how long Acquire keeps retrying on the existing lock
// inode after detecting a stale lock, before giving up with a *nd.LockError.
// A dead holder's flock is released by the kernel when the process exits, so the
// existing inode becomes lockable again without unlinking. We must never remove
// and recreate the lock file to "break" a stale lock: flock(2) locks are tied to
// the inode, so a fresh inode is independently lockable and two contenders that
// both broke the lock would each succeed and lose mutual exclusion.
const staleBreakTimeout = 2 * time.Second

// Acquire attempts to acquire an exclusive flock within the given timeout.
// If the timeout expires and the lock file's modification time is older than
// staleLockThreshold, the lock is considered stale: acquisition keeps retrying
// on the same lock-file inode for a further staleBreakTimeout (a dead holder's
// flock is already released by the kernel). The lock file is never unlinked, so
// every contender opens the same path, same inode, and same flock, and mutual
// exclusion holds. Returns *nd.LockError if the lock cannot be acquired.
func (l *FileLock) Acquire(timeout time.Duration) error {
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file %s: %w", l.Path, err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			l.file = f
			l.AcquiredAt = time.Now()
			return nil
		}

		if time.Now().After(deadline) {
			// Check for a stale lock before giving up. Keep f open: the stale
			// recovery retries on this same inode rather than unlinking it.
			if l.isStale() {
				return l.breakAndRetry(f, timeout)
			}
			f.Close()
			return &nd.LockError{
				Path:    l.Path,
				Timeout: timeout.String(),
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// isStale checks whether the lock file's modification time exceeds the
// stale threshold (60s).
func (l *FileLock) isStale() bool {
	info, err := os.Stat(l.Path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > staleLockThreshold
}

// breakAndRetry recovers from a stale lock without unlinking the lock file. It
// keeps retrying LOCK_EX|LOCK_NB on the already-open fd f (the same inode every
// contender holds) for up to staleBreakTimeout. A lock left by a dead process is
// released by the kernel on exit, so the retry succeeds once that inode becomes
// free; a lock still actively held by a live holder keeps failing and we return
// *nd.LockError with Stale=true. Because the inode is never recreated, two
// contenders that both take this path contend on the same flock and exactly one
// wins — mutual exclusion is preserved. On failure f is closed.
func (l *FileLock) breakAndRetry(f *os.File, timeout time.Duration) error {
	deadline := time.Now().Add(staleBreakTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			l.file = f
			l.AcquiredAt = time.Now()
			return nil
		}

		if time.Now().After(deadline) {
			f.Close()
			return &nd.LockError{
				Path:    l.Path,
				Timeout: timeout.String(),
				Stale:   true,
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Release releases the file lock. Safe to call multiple times.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	l.AcquiredAt = time.Time{}
	return errors.Join(unlockErr, closeErr)
}
