package sourcemanager_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/sourcemanager"
)

func TestGenerateSourceID(t *testing.T) {
	got := sourcemanager.GenerateSourceID("/Users/dev/my-awesome-skills", nil)
	if got != "my-awesome-skills" {
		t.Errorf("got %q, want %q", got, "my-awesome-skills")
	}
}

func TestGenerateSourceIDDedup(t *testing.T) {
	existing := map[string]bool{"my-skills": true}
	got := sourcemanager.GenerateSourceID("/Users/dev/my-skills", existing)
	if got != "my-skills-2" {
		t.Errorf("got %q, want %q", got, "my-skills-2")
	}
}

func newTestManager(t *testing.T) (*sourcemanager.SourceManager, string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return sm, dir
}

func TestAddLocal(t *testing.T) {
	sm, _ := newTestManager(t)

	sourceDir := t.TempDir()
	src, err := sm.AddLocal(sourceDir, "")
	if err != nil {
		t.Fatalf("AddLocal: %v", err)
	}
	if src.Type != nd.SourceLocal {
		t.Errorf("type: got %q", src.Type)
	}
	if src.Path != sourceDir {
		t.Errorf("path: got %q, want %q", src.Path, sourceDir)
	}

	sources := sm.Sources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}

func TestAddLocalWithAlias(t *testing.T) {
	sm, _ := newTestManager(t)

	sourceDir := t.TempDir()
	src, err := sm.AddLocal(sourceDir, "my-alias")
	if err != nil {
		t.Fatalf("AddLocal: %v", err)
	}
	if src.Alias != "my-alias" {
		t.Errorf("alias: got %q", src.Alias)
	}
}

func TestAddLocalNonexistent(t *testing.T) {
	sm, _ := newTestManager(t)
	_, err := sm.AddLocal("/nonexistent/path", "")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestAddLocalNotDirectory(t *testing.T) {
	sm, _ := newTestManager(t)

	f, _ := os.CreateTemp(t.TempDir(), "file")
	f.Close()

	_, err := sm.AddLocal(f.Name(), "")
	if err == nil {
		t.Fatal("expected error for file path")
	}
}

func TestAddLocalDuplicate(t *testing.T) {
	sm, _ := newTestManager(t)

	sourceDir := t.TempDir()
	sm.AddLocal(sourceDir, "")

	_, err := sm.AddLocal(sourceDir, "")
	if err == nil {
		t.Fatal("expected error for duplicate path")
	}
}

func TestAddLocalPersistsToConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, _ := sourcemanager.New(configPath, "")

	sourceDir := t.TempDir()
	sm.AddLocal(sourceDir, "")

	// Load config from disk to verify persistence
	sm2, _ := sourcemanager.New(configPath, "")
	sources := sm2.Sources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source after reload, got %d", len(sources))
	}
	if sources[0].Path != sourceDir {
		t.Errorf("persisted path: got %q, want %q", sources[0].Path, sourceDir)
	}
}

func execGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

func TestAddGitWithBareRepo(t *testing.T) {
	sm, _ := newTestManager(t)

	bareRepo := t.TempDir()
	execGit(t, "init", "--bare", bareRepo)

	src, err := sm.AddGit(bareRepo, "test-alias")
	if err != nil {
		t.Fatalf("AddGit: %v", err)
	}
	if src.Type != nd.SourceGit {
		t.Errorf("type: got %q", src.Type)
	}
	if src.Alias != "test-alias" {
		t.Errorf("alias: got %q", src.Alias)
	}
	if src.URL != bareRepo {
		t.Errorf("url: got %q", src.URL)
	}
}

func TestAddGitDuplicateURL(t *testing.T) {
	sm, _ := newTestManager(t)

	bareRepo := t.TempDir()
	execGit(t, "init", "--bare", bareRepo)

	sm.AddGit(bareRepo, "")

	_, err := sm.AddGit(bareRepo, "")
	if err == nil {
		t.Fatal("expected error for duplicate URL")
	}
}

func TestAddGitDuplicateAfterReload(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, _ := sourcemanager.New(configPath, "")

	bareRepo := t.TempDir()
	execGit(t, "init", "--bare", bareRepo)

	sm.AddGit(bareRepo, "")

	// Reload from disk — URL should be persisted
	sm2, _ := sourcemanager.New(configPath, "")
	_, err := sm2.AddGit(bareRepo, "")
	if err == nil {
		t.Fatal("expected error for duplicate URL after reload")
	}
}
