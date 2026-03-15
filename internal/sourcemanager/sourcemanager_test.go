package sourcemanager_test

import (
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
