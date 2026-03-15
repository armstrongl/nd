package sourcemanager_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/sourcemanager"
)

func TestNewWithMissingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg := sm.Config()
	if cfg.DefaultAgent != "claude-code" {
		t.Errorf("expected defaults, got agent %q", cfg.DefaultAgent)
	}
}

func TestNewWithExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: project
default_agent: claude-code
symlink_strategy: absolute
sources: []
`
	os.WriteFile(configPath, []byte(content), 0o644)

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if sm.Config().DefaultScope != nd.ScopeProject {
		t.Errorf("scope: got %q", sm.Config().DefaultScope)
	}
}

func TestNewWithProjectConfig(t *testing.T) {
	globalDir := t.TempDir()
	globalPath := filepath.Join(globalDir, "config.yaml")

	projectDir := t.TempDir()
	projectConfigDir := filepath.Join(projectDir, ".nd")
	os.MkdirAll(projectConfigDir, 0o755)
	projectContent := `version: 1
default_scope: project
`
	os.WriteFile(filepath.Join(projectConfigDir, "config.yaml"), []byte(projectContent), 0o644)

	sm, err := sourcemanager.New(globalPath, projectDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if sm.Config().DefaultScope != nd.ScopeProject {
		t.Errorf("project override should set scope to project, got %q", sm.Config().DefaultScope)
	}
}

func TestSourcesEmpty(t *testing.T) {
	dir := t.TempDir()
	sm, _ := sourcemanager.New(filepath.Join(dir, "config.yaml"), "")
	sources := sm.Sources()
	if len(sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(sources))
	}
}

func TestSourcesPopulated(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: skills-repo
    type: local
    path: /home/dev/skills
    alias: my-skills
  - id: shared-rules
    type: local
    path: /home/dev/rules
`
	os.WriteFile(configPath, []byte(content), 0o644)

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sources := sm.Sources()
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	if sources[0].ID != "skills-repo" {
		t.Errorf("source[0].ID: got %q", sources[0].ID)
	}
	if sources[0].Type != nd.SourceLocal {
		t.Errorf("source[0].Type: got %q", sources[0].Type)
	}
	if sources[0].Path != "/home/dev/skills" {
		t.Errorf("source[0].Path: got %q", sources[0].Path)
	}
	if sources[0].Alias != "my-skills" {
		t.Errorf("source[0].Alias: got %q", sources[0].Alias)
	}
	if sources[0].Order != 0 {
		t.Errorf("source[0].Order: got %d", sources[0].Order)
	}
	if sources[1].Order != 1 {
		t.Errorf("source[1].Order: got %d", sources[1].Order)
	}
}

func TestScan(t *testing.T) {
	// Create two source directories with assets
	src1 := makeSourceTree(t, map[string][]string{
		"skills": {"review/", "deploy/"},
		"agents": {"go-dev/"},
	})
	src2 := makeSourceTree(t, map[string][]string{
		"skills": {"test/"},
		"rules":  {"no-emojis.md"},
	})

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, _ := sourcemanager.New(configPath, "")

	sm.AddLocal(src1, "")
	sm.AddLocal(src2, "")

	summary, err := sm.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	all := summary.Index.All()
	if len(all) != 5 {
		t.Errorf("expected 5 assets, got %d", len(all))
		for _, a := range all {
			t.Logf("  %s", a.Identity)
		}
	}
}

func TestScanConflictDetection(t *testing.T) {
	// Two sources with same skill name — first registered wins
	src1 := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})
	src2 := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, _ := sourcemanager.New(configPath, "")

	sm.AddLocal(src1, "")
	sm.AddLocal(src2, "")

	summary, _ := sm.Scan()
	conflicts := summary.Index.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
}

func TestScanUnavailableSourceWarning(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: gone
    type: local
    path: /nonexistent/source
`
	os.WriteFile(configPath, []byte(content), 0o644)
	sm, _ := sourcemanager.New(configPath, "")

	summary, err := sm.Scan()
	if err != nil {
		t.Fatalf("Scan should not error for unavailable sources: %v", err)
	}
	if len(summary.Warnings) == 0 {
		t.Error("expected warning for unavailable source")
	}
}

func TestSyncSourceNotFound(t *testing.T) {
	sm, _ := newTestManager(t)
	err := sm.SyncSource("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestSyncSourceNotGit(t *testing.T) {
	sm, _ := newTestManager(t)
	sourceDir := t.TempDir()
	sm.AddLocal(sourceDir, "")
	id := sm.Sources()[0].ID

	err := sm.SyncSource(id)
	if err == nil {
		t.Fatal("expected error for non-git source")
	}
}

func TestSyncSourceGit(t *testing.T) {
	// Create a bare repo, seed it with a commit, then clone it
	bareRepo := t.TempDir()
	execGit(t, "init", "--bare", bareRepo)

	// Seed the bare repo with an initial commit via a temporary clone
	seedDir := t.TempDir()
	execGit(t, "clone", bareRepo, seedDir)
	os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("init"), 0o644)
	execGit(t, "-C", seedDir, "add", ".")
	execGit(t, "-C", seedDir, "commit", "-m", "initial")
	execGit(t, "-C", seedDir, "push")

	cloneDir := t.TempDir()
	execGit(t, "clone", bareRepo, cloneDir)

	// Set up manager with the clone as a git source (manual config)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf(`version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: test-repo
    type: git
    path: %s
`, cloneDir)
	os.WriteFile(configPath, []byte(content), 0o644)

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Sync should succeed (nothing to pull, but git pull should work)
	err = sm.SyncSource("test-repo")
	if err != nil {
		t.Fatalf("SyncSource: %v", err)
	}
}
