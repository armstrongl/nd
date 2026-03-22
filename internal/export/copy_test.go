// internal/export/copy_test.go
package export_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/export"
)

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.md")
	os.WriteFile(src, []byte("# Hello"), 0o644)
	dst := filepath.Join(t.TempDir(), "dest.md")

	err := export.CopyFile(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "# Hello" {
		t.Fatalf("got %q", got)
	}
	// Verify permissions preserved
	info, _ := os.Stat(dst)
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions: got %o, want 0644", info.Mode().Perm())
	}
}

func TestCopyDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	os.MkdirAll(filepath.Join(src, "nested", "deep"), 0o755)
	os.WriteFile(filepath.Join(src, "file1.md"), []byte("root"), 0o644)
	os.WriteFile(filepath.Join(src, "nested", "file2.md"), []byte("nested"), 0o644)
	os.WriteFile(filepath.Join(src, "nested", "deep", "file3.md"), []byte("deep"), 0o644)

	dst := filepath.Join(t.TempDir(), "dest")
	err := export.CopyDir(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all files copied with correct structure
	for _, rel := range []string{"file1.md", "nested/file2.md", "nested/deep/file3.md"} {
		p := filepath.Join(dst, rel)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Fatalf("missing %s", rel)
		}
	}
	got, _ := os.ReadFile(filepath.Join(dst, "nested", "deep", "file3.md"))
	if string(got) != "deep" {
		t.Fatalf("file3.md content: got %q, want %q", got, "deep")
	}
}

func TestCopyFileResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	os.WriteFile(target, []byte("target content"), 0o644)
	link := filepath.Join(dir, "link.md")
	os.Symlink(target, link)
	dst := filepath.Join(t.TempDir(), "dest.md")

	// CopyFile should resolve the symlink and copy target content as a regular file
	err := export.CopyFile(link, dst)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "target content" {
		t.Fatalf("got %q", got)
	}
	info, _ := os.Lstat(dst)
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("dest should be a regular file, not a symlink")
	}
}

func TestCopyDirCreatesParents(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "file.md"), []byte("data"), 0o644)

	dst := filepath.Join(t.TempDir(), "a", "b", "c", "dest")
	err := export.CopyDir(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "file.md")); os.IsNotExist(err) {
		t.Fatal("file not copied")
	}
}

func TestCopyFileSourceMissing(t *testing.T) {
	err := export.CopyFile("/nonexistent/file.md", filepath.Join(t.TempDir(), "dest.md"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestCopyDirSkipsSymlinks(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "real.md"), []byte("real content"), 0o644)

	// Create a symlink to an external file
	externalFile := filepath.Join(t.TempDir(), "external.md")
	os.WriteFile(externalFile, []byte("external content"), 0o644)
	os.Symlink(externalFile, filepath.Join(src, "link.md"))

	dst := filepath.Join(t.TempDir(), "dest")
	err := export.CopyDir(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	// Real file should be copied
	if _, err := os.Stat(filepath.Join(dst, "real.md")); os.IsNotExist(err) {
		t.Fatal("real.md should be copied")
	}

	// Symlink should be skipped
	if _, err := os.Stat(filepath.Join(dst, "link.md")); !os.IsNotExist(err) {
		t.Fatal("link.md (symlink) should be skipped, not copied")
	}
}
