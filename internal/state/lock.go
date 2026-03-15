package state

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/larah/nd/internal/nd"
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

// Acquire attempts to acquire an exclusive flock within the given timeout.
// If the timeout expires and the lock file's modification time is older than
// 60 seconds, the lock is considered stale: the file is removed and acquisition
// is retried once. Returns *nd.LockError if the lock cannot be acquired.
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
			f.Close()
			// Check for stale lock before giving up.
			if l.isStale() {
				return l.breakAndRetry(timeout)
			}
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

// breakAndRetry removes the stale lock file and attempts one more acquisition.
// Returns *nd.LockError with Stale=true if the retry also fails.
func (l *FileLock) breakAndRetry(timeout time.Duration) error {
	os.Remove(l.Path)

	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return &nd.LockError{
			Path:    l.Path,
			Timeout: timeout.String(),
			Stale:   true,
		}
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return &nd.LockError{
			Path:    l.Path,
			Timeout: timeout.String(),
			Stale:   true,
		}
	}

	l.file = f
	l.AcquiredAt = time.Now()
	return nil
}

// Release releases the file lock. Safe to call multiple times.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	l.AcquiredAt = time.Time{}
	return err
}
