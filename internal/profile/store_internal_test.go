package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAutoSnapshotNoOverwrite is a regression test for the bug where
// AutoSnapshot performed no existence check and silently overwrote any file at
// its generated path (e.g. a same-nanosecond name collision). Two calls seeded
// with an identical clock generate an identical name; the second must fail with
// an "already exists" error and must not clobber the first file's contents.
func TestAutoSnapshotNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "profiles"), filepath.Join(dir, "snapshots"))

	fixed := time.Date(2026, 5, 17, 12, 0, 0, 123456789, time.UTC)

	snap, err := store.autoSnapshotAt(fixed, nil)
	if err != nil {
		t.Fatalf("first autoSnapshotAt: %v", err)
	}

	path := store.snapshotPath(snap.Name, true)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first snapshot: %v", err)
	}

	// Same clock -> same generated name -> the target already exists.
	if _, err := store.autoSnapshotAt(fixed, nil); err == nil {
		t.Fatal("expected error when auto snapshot target already exists, got nil")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot after collision: %v", err)
	}
	if string(before) != string(after) {
		t.Error("auto snapshot file was overwritten despite the collision")
	}
}
