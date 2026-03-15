package state

import "time"

// FileLock provides advisory file locking on deployments.yaml.
// Acquired before read-modify-write cycles, released after the atomic rename.
type FileLock struct {
	Path       string    // Path to the lock file (deployments.yaml.lock)
	AcquiredAt time.Time // When the lock was acquired
	fd         int       // File descriptor (internal)
}

// Acquire attempts to acquire the lock within the given timeout.
// Returns *nd.LockError if the lock cannot be acquired.
func (l *FileLock) Acquire(timeout time.Duration) error {
	return nil
}

// Release releases the file lock.
func (l *FileLock) Release() error {
	return nil
}
