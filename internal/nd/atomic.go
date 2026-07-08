package nd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// AtomicWrite writes data to path atomically: write to temp file in the same
// directory, fsync, then rename. Prevents data loss from crashes mid-write (NFR-010).
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)

	f, err := os.CreateTemp(dir, ".nd-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()

	cleanup := func() {
		f.Close()
		os.Remove(tmpPath)
	}

	if _, err := f.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("fsync temp file: %w", err)
	}

	if err := f.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp to target: %w", err)
	}

	// fsync the parent directory so the rename is durable: on POSIX filesystems
	// the renamed directory entry is not guaranteed persistent until the
	// directory itself is synced (NFR-010). Some non-POSIX/network filesystems
	// do not support directory fsync and report EINVAL/ENOTSUP; tolerate that as
	// a no-op. The rename already succeeded, so on failure we only report the
	// durability error and leave the in-place target untouched.
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("fsync parent directory: %w", err)
	}
	if err := d.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) {
		d.Close()
		return fmt.Errorf("fsync parent directory: %w", err)
	}
	if err := d.Close(); err != nil {
		return fmt.Errorf("fsync parent directory: %w", err)
	}

	return nil
}
