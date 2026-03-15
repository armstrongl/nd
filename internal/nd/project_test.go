package nd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot_GitMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindProjectRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
}

func TestFindProjectRoot_ClaudeMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindProjectRoot(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
}

func TestFindProjectRoot_NoMarker(t *testing.T) {
	dir := t.TempDir()
	_, err := FindProjectRoot(dir)
	if err == nil {
		t.Fatal("expected error when no marker found")
	}
}

func TestFindProjectRoot_PrefersClosest(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "sub")
	if err := os.MkdirAll(filepath.Join(inner, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindProjectRoot(inner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != inner {
		t.Errorf("got %q, want %q (should find closest marker)", got, inner)
	}
}
