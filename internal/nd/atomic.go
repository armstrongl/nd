package nd

import (
	"fmt"
	"os"
	"path/filepath"
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

	return nil
}
