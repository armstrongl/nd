package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/version"
)

func TestEnsureExtracted_CreatesDirectoryTree(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache", sanitizeVersion(version.Version))

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("EnsureExtracted: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("cache dir is not a directory")
	}

	// Verify skills directory exists
	if _, err := os.Stat(filepath.Join(dir, "skills")); err != nil {
		t.Errorf("skills/ not extracted: %v", err)
	}

	// Verify commands directory exists
	if _, err := os.Stat(filepath.Join(dir, "commands")); err != nil {
		t.Errorf("commands/ not extracted: %v", err)
	}

	// Verify agents directory exists
	if _, err := os.Stat(filepath.Join(dir, "agents")); err != nil {
		t.Errorf("agents/ not extracted: %v", err)
	}
}

func TestEnsureExtracted_Idempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache", sanitizeVersion(version.Version))

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call should be a no-op
	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestEnsureExtracted_ReextractsOnStampMismatch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache", sanitizeVersion(version.Version))

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("first extract: %v", err)
	}

	// A sentinel file that survives only if the dir is NOT re-extracted.
	sentinel := filepath.Join(dir, "SENTINEL")
	if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	// Matching stamp: EnsureExtracted must be a no-op (sentinel preserved).
	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("no-op extract: %v", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Errorf("matching stamp should not re-extract (sentinel gone): %v", err)
	}

	// Mismatching stamp (simulated nd upgrade with changed embedded assets):
	// EnsureExtracted must wipe and re-extract.
	if err := os.WriteFile(filepath.Join(dir, stampFile), []byte("stale-stamp"), 0o644); err != nil {
		t.Fatalf("corrupt stamp: %v", err)
	}
	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Errorf("expected re-extract to remove sentinel, got stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "skills")); err != nil {
		t.Errorf("skills/ missing after re-extract: %v", err)
	}
	// A fresh, matching stamp must have been rewritten.
	if _, err := os.Stat(filepath.Join(dir, stampFile)); err != nil {
		t.Errorf("stamp not rewritten after re-extract: %v", err)
	}
}

func TestEnsureExtracted_ReextractsWhenStampMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache", sanitizeVersion(version.Version))

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("first extract: %v", err)
	}

	sentinel := filepath.Join(dir, "SENTINEL")
	if err := os.WriteFile(sentinel, []byte("x"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	// Simulate a cache directory extracted by a pre-stamp binary version.
	if err := os.Remove(filepath.Join(dir, stampFile)); err != nil {
		t.Fatalf("remove stamp: %v", err)
	}

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("re-extract: %v", err)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Errorf("expected re-extract to remove sentinel, got stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, stampFile)); err != nil {
		t.Errorf("stamp not written after re-extract: %v", err)
	}
}

func TestEnsureExtracted_SkillsHaveSKILLmd(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache", sanitizeVersion(version.Version))

	if err := EnsureExtracted(dir); err != nil {
		t.Fatalf("EnsureExtracted: %v", err)
	}

	skillsDir := filepath.Join(dir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("read skills dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no skills found in extracted cache")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Errorf("skill %s missing SKILL.md: %v", entry.Name(), err)
		}
	}
}

func TestCacheDir_RespectsXDGOverride(t *testing.T) {
	tmp := t.TempDir()
	old := cacheBaseDir
	cacheBaseDir = tmp
	defer func() { cacheBaseDir = old }()

	got := CacheDir()
	want := filepath.Join(tmp, "nd", "builtin", sanitizeVersion(version.Version))
	if got != want {
		t.Errorf("CacheDir() = %q, want %q", got, want)
	}
}

func TestCacheDir_IncludesVersion(t *testing.T) {
	old := cacheBaseDir
	cacheBaseDir = "/tmp/test-cache"
	defer func() { cacheBaseDir = old }()

	got := CacheDir()
	if !filepath.IsAbs(got) {
		t.Errorf("CacheDir() returned relative path: %q", got)
	}
	if got != filepath.Join("/tmp/test-cache", "nd", "builtin", sanitizeVersion(version.Version)) {
		t.Errorf("CacheDir() = %q, does not include version", got)
	}
}

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v0.5.0", "v0.5.0"},
		{"dev", "dev"},
		{"v1.0.0-rc.1", "v1.0.0-rc.1"},
		{"path/with/slashes", "path_with_slashes"},
	}

	for _, tt := range tests {
		got := sanitizeVersion(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPath_ReturnsExtractedDir(t *testing.T) {
	tmp := t.TempDir()
	old := cacheBaseDir
	cacheBaseDir = tmp
	defer func() { cacheBaseDir = old }()

	got, err := Path()
	if err != nil {
		t.Fatalf("Path(): %v", err)
	}

	// Verify the returned path contains extracted files
	if _, err := os.Stat(filepath.Join(got, "skills")); err != nil {
		t.Errorf("Path() dir missing skills/: %v", err)
	}
}
