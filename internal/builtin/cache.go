package builtin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/armstrongl/nd/internal/version"
)

// cacheBaseDir can be overridden in tests to avoid touching the real filesystem.
var cacheBaseDir string

// CacheDir returns the directory where built-in source files are extracted.
// Format: $XDG_CACHE_HOME/nd/builtin/<version>/ (default ~/.cache/nd/builtin/<version>/).
func CacheDir() string {
	base := cacheBaseDir
	if base == "" {
		base = xdgCacheHome()
	}
	return filepath.Join(base, "nd", "builtin", sanitizeVersion(version.Version))
}

// Path returns the filesystem path to the extracted built-in source.
// It calls EnsureExtracted to materialize the embedded files if needed.
// Returns the path to the source root (the directory containing skills/, commands/, agents/).
func Path() (string, error) {
	dir := CacheDir()
	if err := EnsureExtracted(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// EnsureExtracted checks if the cache directory exists and extracts
// embedded files if it does not. Uses atomic rename to prevent partial state.
func EnsureExtracted(dir string) error {
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return nil // Already extracted
	}

	// Extract to a temp directory in the same parent, then rename atomically.
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create cache parent: %w", err)
	}

	tmpDir, err := os.MkdirTemp(parent, ".tmp-builtin-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	// Clean up temp dir on failure
	success := false
	defer func() {
		if !success {
			os.RemoveAll(tmpDir)
		}
	}()

	// Walk the embedded FS and extract files.
	// The embedded tree is rooted at "source/", so we strip that prefix
	// to produce a standard nd source layout at the top level.
	err = fs.WalkDir(FS, "source", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip the "source/" prefix to get the relative path
		relPath := strings.TrimPrefix(path, "source/")
		if relPath == "" || relPath == "source" {
			return nil // Skip root
		}

		target := filepath.Join(tmpDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("extract embedded source: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpDir, dir); err != nil {
		return fmt.Errorf("finalize cache directory: %w", err)
	}

	success = true
	return nil
}

// xdgCacheHome returns $XDG_CACHE_HOME or ~/.cache as fallback.
func xdgCacheHome() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "nd-cache")
	}
	return filepath.Join(home, ".cache")
}

// sanitizeVersion makes a version string safe for use as a directory name.
func sanitizeVersion(v string) string {
	// Replace characters that are problematic in paths
	r := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		" ", "_",
	)
	return r.Replace(v)
}
