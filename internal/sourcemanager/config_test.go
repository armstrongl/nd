package sourcemanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
)

func TestDefaultConfig(t *testing.T) {
	c := sourcemanager.DefaultConfig()
	if c.Version != nd.SchemaVersion {
		t.Errorf("version: got %d, want %d", c.Version, nd.SchemaVersion)
	}
	if c.DefaultScope != nd.ScopeGlobal {
		t.Errorf("scope: got %q", c.DefaultScope)
	}
	if c.DefaultAgent != "claude-code" {
		t.Errorf("agent: got %q", c.DefaultAgent)
	}
	if c.SymlinkStrategy != nd.SymlinkAbsolute {
		t.Errorf("strategy: got %q", c.SymlinkStrategy)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := sourcemanager.LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultAgent != "claude-code" {
		t.Errorf("expected defaults, got agent %q", cfg.DefaultAgent)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: project
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: my-skills
    type: local
    path: /home/dev/skills
`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := sourcemanager.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DefaultScope != nd.ScopeProject {
		t.Errorf("scope: got %q", cfg.DefaultScope)
	}
	if len(cfg.Sources) != 1 {
		t.Fatalf("sources: got %d", len(cfg.Sources))
	}
	if cfg.Sources[0].ID != "my-skills" {
		t.Errorf("source id: got %q", cfg.Sources[0].ID)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("version: 0\ndefault_scope: bad\n"), 0o644)

	_, err := sourcemanager.LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadProjectConfigMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	pc, err := sourcemanager.LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc != nil {
		t.Error("expected nil for missing project config")
	}
}

func TestLoadProjectConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: project
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pc, err := sourcemanager.LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if pc == nil {
		t.Fatal("expected non-nil project config")
	} else if pc.DefaultScope == nil || *pc.DefaultScope != nd.ScopeProject {
		t.Errorf("scope: got %v", pc.DefaultScope)
	}
}

func TestLoadProjectConfigMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("{{not yaml"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := sourcemanager.LoadProjectConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestMergeConfigs(t *testing.T) {
	global := sourcemanager.DefaultConfig()
	global.Sources = []config.SourceEntry{
		{ID: "global-src", Type: nd.SourceLocal, Path: "/global"},
	}

	projectScope := nd.ScopeProject
	project := config.ProjectConfig{
		Version:      1,
		DefaultScope: &projectScope,
		Sources: []config.SourceEntry{
			{ID: "proj-src", Type: nd.SourceLocal, Path: "/project"},
		},
	}

	merged := sourcemanager.MergeConfigs(global, &project)
	if merged.DefaultScope != nd.ScopeProject {
		t.Errorf("scope should be overridden to project, got %q", merged.DefaultScope)
	}
	if merged.DefaultAgent != "claude-code" {
		t.Errorf("agent should be inherited, got %q", merged.DefaultAgent)
	}
	if len(merged.Sources) != 2 {
		t.Fatalf("should have 2 sources, got %d", len(merged.Sources))
	}
	if merged.Sources[0].ID != "global-src" {
		t.Errorf("global source should come first, got %q", merged.Sources[0].ID)
	}
}

func TestMergeConfigsNilProject(t *testing.T) {
	global := sourcemanager.DefaultConfig()
	merged := sourcemanager.MergeConfigs(global, nil)
	if merged.DefaultScope != global.DefaultScope {
		t.Error("nil project should return global unchanged")
	}
}

func TestMergeConfigsAgentOverrides(t *testing.T) {
	global := sourcemanager.DefaultConfig()
	global.Agents = []config.AgentOverride{
		{Name: "claude-code", GlobalDir: "/global/claude"},
		{Name: "cursor", GlobalDir: "/global/cursor"},
	}

	project := config.ProjectConfig{
		Version: 1,
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: "/project/claude"},
			{Name: "windsurf", GlobalDir: "/project/windsurf"},
		},
	}

	merged := sourcemanager.MergeConfigs(global, &project)
	if len(merged.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(merged.Agents))
	}
	// Should be sorted by name
	if merged.Agents[0].Name != "claude-code" {
		t.Errorf("first agent: got %q", merged.Agents[0].Name)
	}
	// claude-code should use project override
	if merged.Agents[0].GlobalDir != "/project/claude" {
		t.Errorf("claude-code dir: got %q, want /project/claude", merged.Agents[0].GlobalDir)
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := sourcemanager.DefaultConfig()
	cfg.Sources = []config.SourceEntry{
		{ID: "test", Type: nd.SourceLocal, Path: "/test"},
	}

	err := sourcemanager.WriteConfig(path, cfg)
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// Read it back
	loaded, err := sourcemanager.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig after write: %v", err)
	}
	if len(loaded.Sources) != 1 || loaded.Sources[0].ID != "test" {
		t.Errorf("round-trip failed: %+v", loaded.Sources)
	}
}

func TestNewAppendsBuiltinSourceLast(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: user-src
    type: local
    path: /home/dev/skills
`
	os.WriteFile(configPath, []byte(content), 0o644)

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sources := sm.Sources()
	if len(sources) < 2 {
		t.Fatalf("expected at least 2 sources, got %d", len(sources))
	}

	last := sources[len(sources)-1]
	if last.ID != nd.BuiltinSourceID {
		t.Errorf("last source should be builtin, got %q", last.ID)
	}
	if last.Type != nd.SourceBuiltin {
		t.Errorf("builtin source type: got %q, want %q", last.Type, nd.SourceBuiltin)
	}
	if last.Alias != "nd" {
		t.Errorf("builtin source alias: got %q, want %q", last.Alias, "nd")
	}
	if last.Path == "" {
		t.Error("builtin source path should not be empty")
	}

	// User source should still be first (highest priority)
	if sources[0].ID != "user-src" {
		t.Errorf("first source should be user-src, got %q", sources[0].ID)
	}
}

func TestWriteConfigStripsBuiltinSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := sourcemanager.DefaultConfig()
	cfg.Sources = []config.SourceEntry{
		{ID: "user-src", Type: nd.SourceLocal, Path: "/test"},
		{ID: nd.BuiltinSourceID, Type: nd.SourceBuiltin, Path: "/cache/builtin", Alias: "nd"},
	}

	err := sourcemanager.WriteConfig(path, cfg)
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	// Read it back — builtin should not be on disk
	loaded, err := sourcemanager.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig after write: %v", err)
	}
	if len(loaded.Sources) != 1 {
		t.Fatalf("expected 1 source on disk (builtin stripped), got %d", len(loaded.Sources))
	}
	if loaded.Sources[0].ID != "user-src" {
		t.Errorf("persisted source: got %q, want %q", loaded.Sources[0].ID, "user-src")
	}
}

func TestUserSourcesPriorityPreservedWithBuiltin(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `version: 1
default_scope: global
default_agent: claude-code
symlink_strategy: absolute
sources:
  - id: first-src
    type: local
    path: /first
  - id: second-src
    type: local
    path: /second
  - id: third-src
    type: local
    path: /third
`
	os.WriteFile(configPath, []byte(content), 0o644)

	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sources := sm.Sources()
	// 3 user sources + 1 builtin
	if len(sources) != 4 {
		t.Fatalf("expected 4 sources, got %d", len(sources))
	}

	// Verify user source order is preserved
	expectedOrder := []string{"first-src", "second-src", "third-src", nd.BuiltinSourceID}
	for i, want := range expectedOrder {
		if sources[i].ID != want {
			t.Errorf("sources[%d].ID: got %q, want %q", i, sources[i].ID, want)
		}
	}

	// Verify priority ordering (Order field = index)
	for i, s := range sources {
		if s.Order != i {
			t.Errorf("sources[%d].Order: got %d, want %d", i, s.Order, i)
		}
	}
}
