package builtin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/armstrongl/nd/internal/version"
)

// cacheBaseDir can be overridden in tests to avoid touching the real filesystem.
var cacheBaseDir string

// stampFile names the content-stamp file written into an extracted cache dir.
// It records a checksum of the embedded source tree so a rebuilt binary with
// changed builtin assets (common for unversioned "dev" builds, which all share
// one cache dir) invalidates a stale extraction instead of serving it forever.
// The leading dot keeps it out of convention-based source scanning.
const stampFile = ".nd-builtin-stamp"

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

// EnsureExtracted materializes the embedded source into dir. It re-extracts
// whenever the extracted content no longer matches the embedded tree, detected
// via a content stamp: if dir exists and its stamp matches the current embedded
// checksum, extraction is skipped; otherwise the stale dir is removed and the
// tree is re-extracted. Uses atomic rename to prevent partial state.
func EnsureExtracted(dir string) error {
	stamp, err := embeddedStamp()
	if err != nil {
		return fmt.Errorf("compute builtin stamp: %w", err)
	}

	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		// Serve the cache only when its stamp matches the embedded content.
		if existing, rerr := os.ReadFile(filepath.Join(dir, stampFile)); rerr == nil && string(existing) == stamp {
			return nil // Already extracted, content current
		}
		// Missing or mismatched stamp: the extraction is stale (or predates
		// stamping). Remove it so the fresh copy replaces it cleanly.
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove stale cache dir: %w", err)
		}
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

	// Stamp the temp dir before the rename so the finalized cache is atomically
	// tagged with the content it was extracted from.
	if err := os.WriteFile(filepath.Join(tmpDir, stampFile), []byte(stamp), 0o644); err != nil {
		return fmt.Errorf("write builtin stamp: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpDir, dir); err != nil {
		return fmt.Errorf("finalize cache directory: %w", err)
	}

	success = true
	return nil
}

// embeddedStamp computes a deterministic checksum over the embedded source
// tree. Every path (so additions, removals, and renames register) and every
// file's length and contents feed the hash, so any change to the builtin assets
// yields a different stamp. fs.WalkDir visits entries in lexical order, making
// the result stable across runs.
func embeddedStamp() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(FS, "source", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\n", path)
		if d.IsDir() {
			return nil
		}
		data, err := FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		fmt.Fprintf(h, "%d\n", len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
