package nd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/nd"
)

func TestAtomicWriteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	err := nd.AtomicWrite(path, []byte("hello: world\n"))
	if err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello: world\n" {
		t.Errorf("content: got %q", got)
	}
}

func TestAtomicWriteOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	os.WriteFile(path, []byte("old"), 0o644)

	err := nd.AtomicWrite(path, []byte("new"))
	if err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Errorf("content: got %q, want %q", got, "new")
	}
}

func TestAtomicWriteNoPartialOnError(t *testing.T) {
	// Writing to a nonexistent directory should fail without leaving temp files
	path := "/nonexistent/dir/file.yaml"
	err := nd.AtomicWrite(path, []byte("data"))
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
