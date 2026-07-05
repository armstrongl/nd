package oplog

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRotateIfNeededSurfacesNonNotExistStatError is the white-box regression
// test for the swallowed-rotation bug: rotateIfNeeded must return any os.Stat
// error that is not fs.ErrNotExist instead of treating it as "no file yet".
//
// A regular file is used as a directory component in the log path, so os.Stat
// on the "log file" fails with ENOTDIR — an error that is not fs.ErrNotExist.
// Before the fix this branch returned nil and rotation was silently disabled.
func TestRotateIfNeededSurfacesNonNotExistStatError(t *testing.T) {
	dir := t.TempDir()

	// A regular file used as a path component makes os.Stat on any path below
	// it fail with ENOTDIR (deterministic and distinct from fs.ErrNotExist).
	regularFile := filepath.Join(dir, "notadir")
	if err := os.WriteFile(regularFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := NewWriter(regularFile) // path becomes <regularFile>/operations.log
	if err := w.rotateIfNeeded(); err == nil {
		t.Fatal("rotateIfNeeded() = nil; want the non-not-exist stat error to be returned")
	}
}

// TestRotateIfNeededNoFileYet guards the preserved fast path: when the log file
// does not exist yet, rotateIfNeeded returns nil (nothing to rotate).
func TestRotateIfNeededNoFileYet(t *testing.T) {
	w := NewWriter(t.TempDir())
	if err := w.rotateIfNeeded(); err != nil {
		t.Fatalf("rotateIfNeeded() = %v; want nil when the log file does not exist", err)
	}
}
