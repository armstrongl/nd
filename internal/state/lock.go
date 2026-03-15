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

// Acquire attempts to acquire an exclusive flock within the given timeout.
// Returns *nd.LockError if the lock cannot be acquired.
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
			return &nd.LockError{
				Path:    l.Path,
				Timeout: timeout.String(),
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
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	l.AcquiredAt = time.Time{}
	return err
}
