// Package export provides utilities for exporting nd-managed assets
// into the Claude Code plugin format for distribution.
package export

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyFile copies a single file from src to dst, resolving symlinks.
// The destination is always a regular file, never a symlink.
// File permissions are preserved from the source.
// Parent directories of dst are created as needed.
func CopyFile(src, dst string) error {
	// Use os.Stat (not os.Lstat) to resolve symlinks.
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	if srcInfo.IsDir() {
		return fmt.Errorf("copy file: %s is a directory, not a file", src)
	}

	// Create parent directories for destination.
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("copy file: create parent dirs: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy file: open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("copy file: create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy file: write: %w", err)
	}

	return nil
}

// CopyDir recursively copies a directory from src to dst, resolving symlinks.
// All files are copied preserving their relative structure.
// Directories are created with 0o755 permissions.
// Parent directories of dst are created as needed.
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy dir: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("copy dir: %s is not a directory", src)
	}

	// Create the destination directory (and parents).
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("copy dir: create destination: %w", err)
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from source root.
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("copy dir: relative path: %w", err)
		}

		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		// Skip symlinks to prevent following links outside the source tree.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		return CopyFile(path, dstPath)
	})
}
