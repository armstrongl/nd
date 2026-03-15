# Source Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the Source Manager service layer — the first business logic component that loads config, registers sources, scans for assets, and syncs Git repos.

**Architecture:** Single `SourceManager` struct in `internal/sourcemanager/` that owns the full source lifecycle. Uses existing data types from `internal/config/`, `internal/source/`, `internal/asset/`, and `internal/nd/`. Shells out to `git` for clone/pull. All file writes use atomic temp-file-then-rename pattern.

**Tech Stack:** Go 1.23+, gopkg.in/yaml.v3, os/exec for git, t.TempDir() for tests

---

## Task 1: Atomic File Write Utility

Shared utility for atomic file writes (NFR-010). Used by config writing here and later by state/profile persistence.

**Files:**

- Create: `internal/nd/atomic.go`
- Test: `internal/nd/atomic_test.go`

### Step 1: Write the failing test

Create `internal/nd/atomic_test.go`:

```go
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
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/nd/ -run TestAtomicWrite -v`
Expected: FAIL — `AtomicWrite` not defined

### Step 3: Write minimal implementation

Create `internal/nd/atomic.go`:

```go
package nd

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path atomically: write to temp file in the same
// directory, fsync, then rename. Prevents data loss from crashes mid-write (NFR-010).
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)

	f, err := os.CreateTemp(dir, ".nd-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()

	cleanup := func() {
		f.Close()
		os.Remove(tmpPath)
	}

	if _, err := f.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("fsync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp to target: %w", err)
	}

	return nil
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/nd/ -run TestAtomicWrite -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/nd/atomic.go internal/nd/atomic_test.go
git commit -m "feat(nd): add AtomicWrite utility for crash-safe file writes"
```

---

## Task 2: Config Validation

Fill in the existing `Config.Validate()` stub in `internal/config/validation.go` and `ContextMeta.Validate()` stub in `internal/asset/context.go`.

**Files:**

- Modify: `internal/config/validation.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/asset/context.go`
- Modify: `internal/asset/context_test.go`

### Step 1: Write the failing tests for Config.Validate

Add to `internal/config/config_test.go`:

```go
func TestConfigValidateValid(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources:         []config.SourceEntry{},
	}
	errs := c.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestConfigValidateInvalidVersion(t *testing.T) {
	c := config.Config{Version: 0}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for version 0")
	}
}

func TestConfigValidateInvalidScope(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    "invalid",
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for invalid scope")
	}
}

func TestConfigValidateEmptyAgent(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for empty agent")
	}
}

func TestConfigValidateInvalidSymlinkStrategy(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: "invalid",
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for invalid symlink strategy")
	}
}

func TestConfigValidateFutureVersion(t *testing.T) {
	c := config.Config{
		Version:         99,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for future schema version")
	}
}

func TestConfigValidateDuplicateSourceIDs(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "dup", Type: nd.SourceLocal, Path: "/a"},
			{ID: "dup", Type: nd.SourceLocal, Path: "/b"},
		},
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for duplicate source IDs")
	}
}

func TestConfigValidateSourceMissingPath(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "s1", Type: nd.SourceLocal, Path: ""},
		},
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for empty source path")
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/config/ -run TestConfigValidate -v`
Expected: FAIL — most tests will pass with empty slice (stub returns nil), but `TestConfigValidateValid` passes and the others fail because they expect errors from a nil-returning stub.

### Step 3: Write implementation

Replace the **entire file** `internal/config/validation.go` (the existing file contains the `ValidationError` type and the `Validate` stub — this replaces both):

```go
package config

import (
	"fmt"

	"github.com/larah/nd/internal/nd"
)

// ValidationError represents a single config validation failure.
type ValidationError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s:%d: field %s: %s", e.File, e.Line, e.Field, e.Message)
}

// Validate checks all fields for correctness.
// Returns a slice of ValidationError with file and line info (NFR-005).
func (c *Config) Validate() []ValidationError {
	var errs []ValidationError

	if c.Version < 1 {
		errs = append(errs, ValidationError{
			Field: "version", Message: "must be >= 1",
		})
	}

	if c.Version > nd.SchemaVersion {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: fmt.Sprintf("config version %d is newer than supported version %d (downgrade?)", c.Version, nd.SchemaVersion),
		})
	}

	switch c.DefaultScope {
	case nd.ScopeGlobal, nd.ScopeProject:
		// valid
	default:
		errs = append(errs, ValidationError{
			Field:   "default_scope",
			Message: fmt.Sprintf("invalid scope %q, must be %q or %q", c.DefaultScope, nd.ScopeGlobal, nd.ScopeProject),
		})
	}

	if c.DefaultAgent == "" {
		errs = append(errs, ValidationError{
			Field: "default_agent", Message: "must not be empty",
		})
	}

	switch c.SymlinkStrategy {
	case nd.SymlinkAbsolute, nd.SymlinkRelative:
		// valid
	default:
		errs = append(errs, ValidationError{
			Field:   "symlink_strategy",
			Message: fmt.Sprintf("invalid strategy %q, must be %q or %q", c.SymlinkStrategy, nd.SymlinkAbsolute, nd.SymlinkRelative),
		})
	}

	seenIDs := make(map[string]bool)
	for i, s := range c.Sources {
		field := fmt.Sprintf("sources[%d]", i)
		if s.ID == "" {
			errs = append(errs, ValidationError{
				Field: field + ".id", Message: "must not be empty",
			})
		} else if seenIDs[s.ID] {
			errs = append(errs, ValidationError{
				Field: field + ".id", Message: fmt.Sprintf("duplicate source ID %q", s.ID),
			})
		} else {
			seenIDs[s.ID] = true
		}

		if s.Path == "" {
			errs = append(errs, ValidationError{
				Field: field + ".path", Message: "must not be empty",
			})
		}

		switch s.Type {
		case nd.SourceLocal, nd.SourceGit:
			// valid
		default:
			errs = append(errs, ValidationError{
				Field:   field + ".type",
				Message: fmt.Sprintf("invalid source type %q", s.Type),
			})
		}
	}

	return errs
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/config/ -v`
Expected: PASS

### Step 5: Write the failing test for ContextMeta.Validate

Add to `internal/asset/context_test.go`:

```go
func TestContextMetaValidateEmpty(t *testing.T) {
	m := asset.ContextMeta{}
	if err := m.Validate(); err == nil {
		t.Error("expected error for empty description")
	}
}

func TestContextMetaValidateValid(t *testing.T) {
	m := asset.ContextMeta{Description: "Go project rules"}
	if err := m.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
```

### Step 6: Run test to verify it fails

Run: `go test ./internal/asset/ -run TestContextMetaValidate -v`
Expected: FAIL — `TestContextMetaValidateEmpty` fails (stub returns nil)

### Step 7: Write implementation

Update `Validate` in `internal/asset/context.go`:

```go
// Validate checks ContextMeta fields for correctness.
func (m *ContextMeta) Validate() error {
	if m.Description == "" {
		return fmt.Errorf("context meta: description must not be empty")
	}
	return nil
}
```

Add `"fmt"` to the import block in `context.go`.

### Step 8: Run tests to verify they pass

Run: `go test ./internal/asset/ -run TestContextMetaValidate -v`
Expected: PASS

### Step 9: Commit

```shell
git add internal/config/validation.go internal/config/config_test.go internal/asset/context.go internal/asset/context_test.go
git commit -m "feat(config,asset): implement Config.Validate and ContextMeta.Validate"
```

---

## Task 3: Config Loading, Defaults, and Merging

Create the config loading subsystem in `internal/sourcemanager/config.go`. Handles reading YAML, creating defaults, merging global + project configs, and writing back.

**Files:**

- Create: `internal/sourcemanager/config.go`
- Create: `internal/sourcemanager/config_test.go`

### Step 1: Write the failing tests

Create `internal/sourcemanager/config_test.go`:

```go
package sourcemanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/sourcemanager"
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
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -v`
Expected: FAIL — package does not exist

### Step 3: Write implementation

Create `internal/sourcemanager/config.go`:

```go
package sourcemanager

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
)

// DefaultConfig returns a Config with built-in defaults.
func DefaultConfig() config.Config {
	return config.Config{
		Version:         nd.SchemaVersion,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources:         []config.SourceEntry{},
	}
}

// LoadConfig reads and validates a config file. If the file does not exist,
// returns defaults (first-run experience).
func LoadConfig(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return config.Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config.Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if errs := cfg.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return config.Config{}, errors.New(strings.Join(msgs, "; "))
	}

	return cfg, nil
}

// LoadProjectConfig reads a project-level config file.
// Returns nil if the file does not exist.
func LoadProjectConfig(path string) (*config.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read project config: %w", err)
	}

	var pc config.ProjectConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parse project config %s: %w", path, err)
	}

	return &pc, nil
}

// MergeConfigs merges a global config with an optional project config.
// Project fields override global when non-nil. Sources are appended
// (global first for priority per FR-016a). Agent overrides from project
// replace global entries by agent name.
func MergeConfigs(global config.Config, project *config.ProjectConfig) config.Config {
	if project == nil {
		return global
	}

	merged := global

	if project.DefaultScope != nil {
		merged.DefaultScope = *project.DefaultScope
	}
	if project.DefaultAgent != nil {
		merged.DefaultAgent = *project.DefaultAgent
	}
	if project.SymlinkStrategy != nil {
		merged.SymlinkStrategy = *project.SymlinkStrategy
	}

	if len(project.Sources) > 0 {
		merged.Sources = append(merged.Sources, project.Sources...)
	}

	if len(project.Agents) > 0 {
		agentMap := make(map[string]config.AgentOverride)
		for _, a := range merged.Agents {
			agentMap[a.Name] = a
		}
		for _, a := range project.Agents {
			agentMap[a.Name] = a
		}
		names := make([]string, 0, len(agentMap))
		for name := range agentMap {
			names = append(names, name)
		}
		sort.Strings(names)
		merged.Agents = make([]config.AgentOverride, 0, len(agentMap))
		for _, name := range names {
			merged.Agents = append(merged.Agents, agentMap[name])
		}
	}

	return merged
}

// WriteConfig writes a config to disk using atomic writes (NFR-010).
func WriteConfig(path string, cfg config.Config) error {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return nd.AtomicWrite(path, data)
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/config.go internal/sourcemanager/config_test.go
git commit -m "feat(sourcemanager): add config loading, defaults, merging, and writing"
```

---

## Task 4: SourceManager Struct

Create the main `SourceManager` struct with `New()`, `Config()`, and `Sources()`.

**Files:**

- Create: `internal/sourcemanager/sourcemanager.go`
- Create: `internal/sourcemanager/sourcemanager_test.go`

### Step 1: Write the failing tests

Create `internal/sourcemanager/sourcemanager_test.go`:

```go
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
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestNew -v`
Expected: FAIL — `sourcemanager.New` not defined

### Step 3: Write implementation

Create `internal/sourcemanager/sourcemanager.go`:

```go
package sourcemanager

import (
	"fmt"
	"path/filepath"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/source"
)

// SourceManager owns the full source lifecycle: config, registration,
// scanning, and sync.
type SourceManager struct {
	configPath string
	sourcesDir string // derived from configPath: <configDir>/sources/
	projectDir string
	cfg        config.Config
}

// New creates a SourceManager by loading the global config and optionally
// merging a project config. If the global config file does not exist,
// defaults are used (first-run experience).
func New(configPath string, projectDir string) (*SourceManager, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if projectDir != "" {
		projectConfigPath := filepath.Join(projectDir, ".nd", "config.yaml")
		pc, err := LoadProjectConfig(projectConfigPath)
		if err != nil {
			return nil, fmt.Errorf("load project config: %w", err)
		}
		cfg = MergeConfigs(cfg, pc)
	}

	return &SourceManager{
		configPath: configPath,
		sourcesDir: filepath.Join(filepath.Dir(configPath), "sources"),
		projectDir: projectDir,
		cfg:        cfg,
	}, nil
}

// Config returns the current merged configuration.
func (sm *SourceManager) Config() *config.Config {
	return &sm.cfg
}

// Sources returns all registered sources with availability status.
func (sm *SourceManager) Sources() []source.Source {
	sources := make([]source.Source, len(sm.cfg.Sources))
	for i, entry := range sm.cfg.Sources {
		sources[i] = source.Source{
			ID:    entry.ID,
			Type:  entry.Type,
			Path:  entry.Path,
			URL:   entry.URL,
			Alias: entry.Alias,
			Order: i,
		}
	}
	return sources
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/sourcemanager.go internal/sourcemanager/sourcemanager_test.go
git commit -m "feat(sourcemanager): add SourceManager struct with New, Config, Sources"
```

---

## Task 5: Git URL Parsing

Parse GitHub shorthand, HTTPS, and SSH URLs.

**Files:**

- Create: `internal/sourcemanager/git.go`
- Create: `internal/sourcemanager/git_test.go`

### Step 1: Write the failing tests

Create `internal/sourcemanager/git_test.go`:

```go
package sourcemanager_test

import (
	"testing"

	"github.com/larah/nd/internal/sourcemanager"
)

func TestExpandGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"owner/repo", "https://github.com/owner/repo.git"},
		{"my-org/my-skills", "https://github.com/my-org/my-skills.git"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"https://gitlab.com/org/repo.git", "https://gitlab.com/org/repo.git"},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git"},
		{"git@gitlab.com:org/repo.git", "git@gitlab.com:org/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "ssh://git@github.com/owner/repo.git"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sourcemanager.ExpandGitURL(tt.input)
			if got != tt.want {
				t.Errorf("ExpandGitURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/owner/my-skills.git", "my-skills"},
		{"https://github.com/owner/repo", "repo"},
		{"git@github.com:owner/repo.git", "repo"},
		{"owner/repo", "repo"},
		{"owner/my-cool-skills", "my-cool-skills"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sourcemanager.RepoNameFromURL(tt.input)
			if got != tt.want {
				t.Errorf("RepoNameFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestExpand -v`
Expected: FAIL — functions not defined

### Step 3: Write implementation

Create `internal/sourcemanager/git.go`:

```go
package sourcemanager

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExpandGitURL expands GitHub shorthand (owner/repo) to a full HTTPS URL.
// Full URLs (HTTPS, SSH, ssh://) are returned as-is.
func ExpandGitURL(input string) string {
	if strings.Contains(input, "://") || strings.HasPrefix(input, "git@") {
		return input
	}
	// GitHub shorthand: owner/repo
	parts := strings.SplitN(input, "/", 2)
	if len(parts) == 2 && !strings.Contains(parts[0], ".") {
		return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1])
	}
	return input
}

// RepoNameFromURL extracts the repository name from a Git URL or shorthand.
func RepoNameFromURL(url string) string {
	// Handle shorthand: owner/repo
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}

	// Handle git@host:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		if idx := strings.LastIndex(url, "/"); idx >= 0 {
			return strings.TrimSuffix(url[idx+1:], ".git")
		}
		if idx := strings.LastIndex(url, ":"); idx >= 0 {
			return strings.TrimSuffix(url[idx+1:], ".git")
		}
	}

	// Handle https://host/owner/repo.git
	base := filepath.Base(url)
	return strings.TrimSuffix(base, ".git")
}

// gitClone clones a repository to the target directory.
func gitClone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %s: %w", url, string(output), err)
	}
	return nil
}

// gitPull runs git pull --ff-only in the given directory.
func gitPull(repoDir string) error {
	cmd := exec.Command("git", "-C", repoDir, "pull", "--ff-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull in %s: %s: %w", repoDir, string(output), err)
	}
	return nil
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run "TestExpand|TestRepoName" -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/git.go internal/sourcemanager/git_test.go
git commit -m "feat(sourcemanager): add Git URL parsing and repo name extraction"
```

---

## Task 6: Source ID Generation and AddLocal

Register local directories as asset sources.

**Files:**

- Create: `internal/sourcemanager/register.go`
- Create: `internal/sourcemanager/register_test.go`

### Step 1: Write the failing tests

Create `internal/sourcemanager/register_test.go`:

```go
package sourcemanager_test

import (
	"os"
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
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run "TestGenerate|TestAddLocal" -v`
Expected: FAIL — functions not defined

### Step 3: Write implementation

Create `internal/sourcemanager/register.go`:

```go
package sourcemanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/source"
)

// GenerateSourceID creates a source ID from a path's base name,
// deduplicating with a numeric suffix if needed.
func GenerateSourceID(path string, existingIDs map[string]bool) string {
	base := filepath.Base(path)
	if existingIDs == nil || !existingIDs[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !existingIDs[candidate] {
			return candidate
		}
	}
}

// AddLocal registers a local directory as an asset source.
func (sm *SourceManager) AddLocal(path string, alias string) (*source.Source, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", absPath)
	}

	// Check for duplicate
	for _, s := range sm.cfg.Sources {
		if s.Path == absPath {
			return nil, fmt.Errorf("source at %q is already registered as %q", absPath, s.ID)
		}
	}

	existingIDs := make(map[string]bool)
	for _, s := range sm.cfg.Sources {
		existingIDs[s.ID] = true
	}
	id := GenerateSourceID(absPath, existingIDs)

	entry := config.SourceEntry{
		ID:    id,
		Type:  nd.SourceLocal,
		Path:  absPath,
		Alias: alias,
	}

	sm.cfg.Sources = append(sm.cfg.Sources, entry)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		// Roll back in-memory change
		sm.cfg.Sources = sm.cfg.Sources[:len(sm.cfg.Sources)-1]
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &source.Source{
		ID:    id,
		Type:  nd.SourceLocal,
		Path:  absPath,
		Alias: alias,
		Order: len(sm.cfg.Sources) - 1,
	}, nil
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run "TestGenerate|TestAddLocal" -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/register.go internal/sourcemanager/register_test.go
git commit -m "feat(sourcemanager): add source ID generation and AddLocal"
```

---

## Task 7: AddGit

Register Git repositories as asset sources by cloning them. This task also adds a `URL` field to `config.SourceEntry` to persist the original Git URL for duplicate detection after config reload.

**Files:**

- Modify: `internal/config/config.go` (add URL field to SourceEntry)
- Modify: `internal/sourcemanager/sourcemanager.go` (add sourcesDir field)
- Modify: `internal/sourcemanager/register.go`
- Modify: `internal/sourcemanager/register_test.go`

### Step 1: Add URL field to SourceEntry

Add a `URL` field to `config.SourceEntry` in `internal/config/config.go` so Git source URLs are persisted and survive config reloads:

```go
// SourceEntry represents a source registration in the config file.
// Sources are listed in registration order (first registered = highest priority per FR-016a).
type SourceEntry struct {
	ID    string        `yaml:"id"              json:"id"`
	Type  nd.SourceType `yaml:"type"            json:"type"`
	Path  string        `yaml:"path"            json:"path"`
	URL   string        `yaml:"url,omitempty"   json:"url,omitempty"`
	Alias string        `yaml:"alias,omitempty" json:"alias,omitempty"`
}
```

Also add `sourcesDir` to `SourceManager` in `internal/sourcemanager/sourcemanager.go`:

```go
type SourceManager struct {
	configPath string
	sourcesDir string
	projectDir string
	cfg        config.Config
}
```

And update `New()` to derive `sourcesDir` from `configPath`:

```go
return &SourceManager{
	configPath: configPath,
	sourcesDir: filepath.Join(filepath.Dir(configPath), "sources"),
	projectDir: projectDir,
	cfg:        cfg,
}, nil
```

Also update `Sources()` to include the URL field:

```go
func (sm *SourceManager) Sources() []source.Source {
	sources := make([]source.Source, len(sm.cfg.Sources))
	for i, entry := range sm.cfg.Sources {
		sources[i] = source.Source{
			ID:    entry.ID,
			Type:  entry.Type,
			Path:  entry.Path,
			URL:   entry.URL,
			Alias: entry.Alias,
			Order: i,
		}
	}
	return sources
}
```

### Step 2: Write the failing tests

Add to `internal/sourcemanager/register_test.go`:

```go
func TestAddGitWithBareRepo(t *testing.T) {
	sm, _ := newTestManager(t)

	bareRepo := t.TempDir()
	exec_git(t, "init", "--bare", bareRepo)

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
	exec_git(t, "init", "--bare", bareRepo)

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
	exec_git(t, "init", "--bare", bareRepo)

	sm.AddGit(bareRepo, "")

	// Reload from disk — URL should be persisted
	sm2, _ := sourcemanager.New(configPath, "")
	_, err := sm2.AddGit(bareRepo, "")
	if err == nil {
		t.Fatal("expected error for duplicate URL after reload")
	}
}

func exec_git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}
```

Add `"os/exec"` to the import block.

### Step 3: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestAddGit -v`
Expected: FAIL — `AddGit` not defined

### Step 4: Write implementation

Add to `internal/sourcemanager/register.go`:

```go
// AddGit registers a Git repository as an asset source by cloning it.
// Clone target is derived from sm.sourcesDir (e.g., ~/.config/nd/sources/).
func (sm *SourceManager) AddGit(url string, alias string) (*source.Source, error) {
	expandedURL := ExpandGitURL(url)

	// Check for duplicate URL (persisted in config via SourceEntry.URL)
	for _, s := range sm.cfg.Sources {
		if s.Type == nd.SourceGit && s.URL == expandedURL {
			return nil, fmt.Errorf("git source %q is already registered as %q", url, s.ID)
		}
	}

	existingIDs := make(map[string]bool)
	for _, s := range sm.cfg.Sources {
		existingIDs[s.ID] = true
	}
	repoName := RepoNameFromURL(url)
	id := GenerateSourceID(filepath.Join(sm.sourcesDir, repoName), existingIDs)

	cloneTarget := filepath.Join(sm.sourcesDir, id)

	if err := os.MkdirAll(sm.sourcesDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sources dir: %w", err)
	}

	if err := gitClone(expandedURL, cloneTarget); err != nil {
		os.RemoveAll(cloneTarget)
		return nil, err
	}

	entry := config.SourceEntry{
		ID:    id,
		Type:  nd.SourceGit,
		Path:  cloneTarget,
		URL:   expandedURL,
		Alias: alias,
	}

	sm.cfg.Sources = append(sm.cfg.Sources, entry)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		sm.cfg.Sources = sm.cfg.Sources[:len(sm.cfg.Sources)-1]
		os.RemoveAll(cloneTarget)
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &source.Source{
		ID:    id,
		Type:  nd.SourceGit,
		Path:  cloneTarget,
		URL:   expandedURL,
		Alias: alias,
		Order: len(sm.cfg.Sources) - 1,
	}, nil
}
```

### Step 5: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run TestAddGit -v`
Expected: PASS

### Step 6: Commit

```shell
git add internal/config/config.go internal/sourcemanager/sourcemanager.go internal/sourcemanager/register.go internal/sourcemanager/register_test.go
git commit -m "feat(sourcemanager): add URL to SourceEntry and implement AddGit"
```

---

## Task 8: Remove Source

Remove a registered source from config.

**Files:**

- Modify: `internal/sourcemanager/register.go`
- Modify: `internal/sourcemanager/register_test.go`

### Step 1: Write the failing tests

Add to `internal/sourcemanager/register_test.go`:

```go
func TestRemove(t *testing.T) {
	sm, _ := newTestManager(t)

	sourceDir := t.TempDir()
	sm.AddLocal(sourceDir, "")
	if len(sm.Sources()) != 1 {
		t.Fatal("setup: expected 1 source")
	}

	id := sm.Sources()[0].ID
	err := sm.Remove(id)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(sm.Sources()) != 0 {
		t.Errorf("expected 0 sources after remove, got %d", len(sm.Sources()))
	}
}

func TestRemoveNotFound(t *testing.T) {
	sm, _ := newTestManager(t)
	err := sm.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestRemovePersists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, _ := sourcemanager.New(configPath, "")

	sourceDir := t.TempDir()
	sm.AddLocal(sourceDir, "")
	id := sm.Sources()[0].ID
	sm.Remove(id)

	// Reload from disk
	sm2, _ := sourcemanager.New(configPath, "")
	if len(sm2.Sources()) != 0 {
		t.Errorf("expected 0 sources after reload, got %d", len(sm2.Sources()))
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestRemove -v`
Expected: FAIL — `Remove` not defined

### Step 3: Write implementation

Add to `internal/sourcemanager/register.go`:

```go
// Remove unregisters a source by ID. Does not delete deployed assets
// or cloned directories — that is the caller's responsibility.
func (sm *SourceManager) Remove(sourceID string) error {
	idx := -1
	for i, s := range sm.cfg.Sources {
		if s.ID == sourceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("source %q not found", sourceID)
	}

	removed := sm.cfg.Sources[idx]
	sm.cfg.Sources = append(sm.cfg.Sources[:idx], sm.cfg.Sources[idx+1:]...)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		// Roll back
		sm.cfg.Sources = append(sm.cfg.Sources[:idx], append([]config.SourceEntry{removed}, sm.cfg.Sources[idx:]...)...)
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run TestRemove -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/register.go internal/sourcemanager/register_test.go
git commit -m "feat(sourcemanager): add Remove for source unregistration"
```

---

## Task 9: Convention-Based Asset Scanner

Scan source directories for assets using the conventional directory layout.

**Files:**

- Create: `internal/sourcemanager/scanner.go`
- Create: `internal/sourcemanager/scanner_test.go`

### Step 1: Write the failing tests

Create `internal/sourcemanager/scanner_test.go`:

```go
package sourcemanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/sourcemanager"
)

// makeSourceTree creates a source directory with assets in conventional layout.
func makeSourceTree(t *testing.T, assets map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	for dir, entries := range assets {
		dirPath := filepath.Join(root, dir)
		os.MkdirAll(dirPath, 0o755)
		for _, entry := range entries {
			entryPath := filepath.Join(dirPath, entry)
			if entry[len(entry)-1] == '/' {
				// Directory entry
				os.MkdirAll(entryPath[:len(entryPath)-1], 0o755)
				// Create a placeholder file inside
				os.WriteFile(filepath.Join(entryPath[:len(entryPath)-1], "SKILL.md"), []byte("# skill"), 0o644)
			} else {
				os.WriteFile(entryPath, []byte("# "+entry), 0o644)
			}
		}
	}
	return root
}

func TestScanConventionBasic(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills":  {"review/", "deploy/"},
		"agents":  {"go-dev/"},
		"rules":   {"no-emojis.md"},
		"commands": {"build-project.md"},
	})

	result := sourcemanager.ScanSource("test-source", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Assets) != 5 {
		t.Errorf("expected 5 assets, got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}

	// Check types
	typeCount := make(map[nd.AssetType]int)
	for _, a := range result.Assets {
		typeCount[a.Type]++
	}
	if typeCount[nd.AssetSkill] != 2 {
		t.Errorf("skills: got %d, want 2", typeCount[nd.AssetSkill])
	}
	if typeCount[nd.AssetAgent] != 1 {
		t.Errorf("agents: got %d, want 1", typeCount[nd.AssetAgent])
	}
	if typeCount[nd.AssetRule] != 1 {
		t.Errorf("rules: got %d, want 1", typeCount[nd.AssetRule])
	}
}

func TestScanConventionSkipsExcluded(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})
	// Add excluded dirs that should be ignored
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	os.MkdirAll(filepath.Join(root, "node_modules"), 0o755)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (only skills/review), got %d", len(result.Assets))
	}
}

func TestScanConventionEmptySource(t *testing.T) {
	root := t.TempDir()
	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 0 {
		t.Errorf("expected 0 assets, got %d", len(result.Assets))
	}
	if len(result.Errors) > 0 {
		t.Errorf("empty source should not produce errors: %v", result.Errors)
	}
}

func TestScanConventionUnavailableSource(t *testing.T) {
	result := sourcemanager.ScanSource("test", "/nonexistent/source")
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unavailable source")
	}
}

func TestScanConventionAssetIdentity(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})

	result := sourcemanager.ScanSource("my-source", root)
	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Assets))
	}
	a := result.Assets[0]
	if a.SourceID != "my-source" {
		t.Errorf("source id: got %q", a.SourceID)
	}
	if a.Name != "review" {
		t.Errorf("name: got %q", a.Name)
	}
	if a.Type != nd.AssetSkill {
		t.Errorf("type: got %q", a.Type)
	}
	if !a.IsDir {
		t.Error("skills should be directories")
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestScanConvention -v`
Expected: FAIL — `ScanSource` not defined

### Step 3: Write implementation

Create `internal/sourcemanager/scanner.go`:

```go
package sourcemanager

import (
	"os"
	"path/filepath"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/source"
)

// excludedDirs are directories that source scanning always skips (NFR-017).
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
}

// dirToAssetType maps conventional directory names to asset types.
var dirToAssetType = map[string]nd.AssetType{
	"skills":        nd.AssetSkill,
	"agents":        nd.AssetAgent,
	"commands":      nd.AssetCommand,
	"output-styles": nd.AssetOutputStyle,
	"rules":         nd.AssetRule,
	"context":       nd.AssetContext,
	"plugins":       nd.AssetPlugin,
	"hooks":         nd.AssetHook,
}

// ScanSource scans a single source directory for assets using convention-based
// discovery. Returns a ScanResult with discovered assets, warnings, and errors.
func ScanSource(sourceID string, rootPath string) source.ScanResult {
	result := source.ScanResult{SourceID: sourceID}

	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		result.Warnings = append(result.Warnings,
			"source "+sourceID+" at "+rootPath+" is unavailable")
		return result
	}

	for dirName, assetType := range dirToAssetType {
		dirPath := filepath.Join(rootPath, dirName)
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			continue
		}

		if assetType == nd.AssetContext {
			scanContextDir(&result, sourceID, dirPath)
			continue
		}

		scanAssetDir(&result, sourceID, assetType, dirPath)
	}

	return result
}

// scanAssetDir scans a single asset type directory for entries.
func scanAssetDir(result *source.ScanResult, sourceID string, assetType nd.AssetType, dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if excludedDirs[name] || name[0] == '.' {
			continue
		}

		result.Assets = append(result.Assets, asset.Asset{
			Identity: asset.Identity{
				SourceID: sourceID,
				Type:     assetType,
				Name:     name,
			},
			SourcePath: filepath.Join(dirPath, name),
			IsDir:      entry.IsDir(),
		})
	}
}
```

Note: `scanContextDir` is a placeholder call — it will be implemented in Task 10. For now, add a stub at the bottom of `scanner.go`:

```go
// scanContextDir scans the context/ directory for context assets.
// Context assets use a folder-per-asset layout with optional _meta.yaml.
func scanContextDir(result *source.ScanResult, sourceID string, dirPath string) {
	// Implemented in Task 10
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run TestScanConvention -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/scanner.go internal/sourcemanager/scanner_test.go
git commit -m "feat(sourcemanager): add convention-based asset scanner"
```

---

## Task 10: Context Asset Scanner

Scan the `context/` directory with its special folder-per-asset layout and optional `_meta.yaml`.

**Files:**

- Modify: `internal/sourcemanager/scanner.go` (replace `scanContextDir` stub)
- Modify: `internal/sourcemanager/scanner_test.go`

### Step 1: Write the failing tests

Add to `internal/sourcemanager/scanner_test.go`:

```go
func TestScanContextAssets(t *testing.T) {
	root := t.TempDir()

	// Create context folder structure
	ctx1 := filepath.Join(root, "context", "go-project-rules")
	os.MkdirAll(ctx1, 0o755)
	os.WriteFile(filepath.Join(ctx1, "CLAUDE.md"), []byte("# Go rules"), 0o644)
	os.WriteFile(filepath.Join(ctx1, "_meta.yaml"), []byte("description: Go project rules\ntags:\n  - go\n"), 0o644)

	ctx2 := filepath.Join(root, "context", "web-frontend")
	os.MkdirAll(ctx2, 0o755)
	os.WriteFile(filepath.Join(ctx2, "CLAUDE.md"), []byte("# Web rules"), 0o644)
	// No _meta.yaml for this one

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 context assets, got %d", len(result.Assets))
	}

	// Find the one with metadata
	var withMeta, withoutMeta *asset.Asset
	for i := range result.Assets {
		if result.Assets[i].Name == "go-project-rules" {
			withMeta = &result.Assets[i]
		}
		if result.Assets[i].Name == "web-frontend" {
			withoutMeta = &result.Assets[i]
		}
	}

	if withMeta == nil {
		t.Fatal("go-project-rules not found")
	}
	if withMeta.Type != nd.AssetContext {
		t.Errorf("type: got %q", withMeta.Type)
	}
	if withMeta.ContextFile == nil {
		t.Fatal("ContextFile should be set")
	}
	if withMeta.ContextFile.FolderName != "go-project-rules" {
		t.Errorf("folder: got %q", withMeta.ContextFile.FolderName)
	}
	if withMeta.ContextFile.FileName != "CLAUDE.md" {
		t.Errorf("file: got %q", withMeta.ContextFile.FileName)
	}
	if withMeta.Meta == nil {
		t.Fatal("Meta should be set for asset with _meta.yaml")
	}
	if withMeta.Meta.Description != "Go project rules" {
		t.Errorf("description: got %q", withMeta.Meta.Description)
	}

	if withoutMeta == nil {
		t.Fatal("web-frontend not found")
	}
	if withoutMeta.Meta != nil {
		t.Error("Meta should be nil for asset without _meta.yaml")
	}
	if withoutMeta.ContextFile == nil {
		t.Fatal("ContextFile should still be set")
	}
}

func TestScanContextLocalOnly(t *testing.T) {
	root := t.TempDir()

	ctx := filepath.Join(root, "context", "local-rules")
	os.MkdirAll(ctx, 0o755)
	os.WriteFile(filepath.Join(ctx, "CLAUDE.local.md"), []byte("# Local"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Assets))
	}
	if result.Assets[0].ContextFile.FileName != "CLAUDE.local.md" {
		t.Errorf("file: got %q", result.Assets[0].ContextFile.FileName)
	}
}
```

Add `"github.com/larah/nd/internal/asset"` to the test file imports.

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestScanContext -v`
Expected: FAIL — context scanning returns 0 assets (stub is empty)

### Step 3: Write implementation

Replace the `scanContextDir` stub in `internal/sourcemanager/scanner.go`:

```go
// scanContextDir scans the context/ directory for context assets.
// Context assets use a folder-per-asset layout (FR-016b):
//
//	context/
//	  go-project-rules/
//	    CLAUDE.md
//	    _meta.yaml
func scanContextDir(result *source.ScanResult, sourceID string, dirPath string) {
	folders, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, folder := range folders {
		if !folder.IsDir() || folder.Name()[0] == '.' {
			continue
		}

		folderPath := filepath.Join(dirPath, folder.Name())
		contextFile := findContextFile(folderPath)
		if contextFile == "" {
			result.Warnings = append(result.Warnings,
				"context folder "+folder.Name()+" has no context file")
			continue
		}

		a := asset.Asset{
			Identity: asset.Identity{
				SourceID: sourceID,
				Type:     nd.AssetContext,
				Name:     folder.Name(),
			},
			SourcePath: filepath.Join(folderPath, contextFile),
			IsDir:      false,
			ContextFile: &asset.ContextInfo{
				FolderName: folder.Name(),
				FileName:   contextFile,
			},
		}

		// Load optional _meta.yaml
		metaPath := filepath.Join(folderPath, "_meta.yaml")
		if meta, err := loadContextMeta(metaPath); err == nil && meta != nil {
			a.Meta = meta
		}

		result.Assets = append(result.Assets, a)
	}
}

// findContextFile looks for a recognized context file in a folder.
func findContextFile(folderPath string) string {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '_' {
			continue
		}
		// Accept any .md file that isn't _meta.yaml
		if filepath.Ext(e.Name()) == ".md" {
			return e.Name()
		}
	}
	return ""
}

// loadContextMeta loads and validates a _meta.yaml file.
func loadContextMeta(path string) (*asset.ContextMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta asset.ContextMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
```

Add `"gopkg.in/yaml.v3"` to the import block in `scanner.go`.

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run TestScanContext -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/sourcemanager/scanner.go internal/sourcemanager/scanner_test.go
git commit -m "feat(sourcemanager): add context asset scanning with _meta.yaml support"
```

---

## Task 11: Manifest-Based Scanning and Full Scan()

Add manifest-based scanning and the top-level `Scan()` method on `SourceManager`.

**Files:**

- Modify: `internal/sourcemanager/scanner.go`
- Modify: `internal/sourcemanager/scanner_test.go`
- Modify: `internal/sourcemanager/sourcemanager.go`
- Modify: `internal/sourcemanager/sourcemanager_test.go`

### Step 1: Write the failing tests for manifest scanning

Add to `internal/sourcemanager/scanner_test.go`:

```go
func TestScanWithManifest(t *testing.T) {
	root := t.TempDir()

	// Non-conventional layout with manifest
	os.MkdirAll(filepath.Join(root, "go-skills", "skills"), 0o755)
	os.MkdirAll(filepath.Join(root, "go-skills", "skills", "review"), 0o755)
	os.WriteFile(filepath.Join(root, "go-skills", "skills", "review", "SKILL.md"), []byte("# review"), 0o644)
	os.MkdirAll(filepath.Join(root, "custom-agents"), 0o755)
	os.MkdirAll(filepath.Join(root, "custom-agents", "builder"), 0o755)

	// Also has conventional skills/ that should be IGNORED when manifest exists
	os.MkdirAll(filepath.Join(root, "skills"), 0o755)
	os.MkdirAll(filepath.Join(root, "skills", "ignored"), 0o755)

	manifest := `version: 1
paths:
  skills:
    - go-skills/skills
  agents:
    - custom-agents
`
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), []byte(manifest), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("errors: %v", result.Errors)
	}
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 assets (1 skill + 1 agent), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}

	// The conventional skills/ignored should NOT be discovered
	for _, a := range result.Assets {
		if a.Name == "ignored" {
			t.Error("conventional skills/ignored should not be discovered when manifest exists")
		}
	}
}

func TestScanManifestExclude(t *testing.T) {
	root := t.TempDir()

	// Create skills
	os.MkdirAll(filepath.Join(root, "skills", "keep"), 0o755)
	os.MkdirAll(filepath.Join(root, "skills", "experimental"), 0o755)

	manifest := `version: 1
paths:
  skills:
    - skills
exclude:
  - experimental
`
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), []byte(manifest), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (excluded experimental), got %d", len(result.Assets))
	}
	for _, a := range result.Assets {
		if a.Name == "experimental" {
			t.Error("excluded asset should not be discovered")
		}
	}
}

func TestScanManifestSizeLimit(t *testing.T) {
	root := t.TempDir()
	// Create a manifest larger than 1MB
	data := make([]byte, 1024*1024+1)
	for i := range data {
		data[i] = 'x'
	}
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), data, 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) == 0 {
		t.Error("expected error for oversized manifest")
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run "TestScanWithManifest|TestScanManifestSize" -v`
Expected: FAIL — manifest scanning not implemented

### Step 3: Write implementation for manifest scanning

Add to `internal/sourcemanager/scanner.go`, modifying `ScanSource`:

```go
const maxManifestSize = 1024 * 1024 // 1MB (NFR-013)

// ScanSource scans a single source directory for assets.
// If nd-source.yaml exists, uses manifest paths. Otherwise uses convention-based discovery.
func ScanSource(sourceID string, rootPath string) source.ScanResult {
	result := source.ScanResult{SourceID: sourceID}

	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		result.Warnings = append(result.Warnings,
			"source "+sourceID+" at "+rootPath+" is unavailable")
		return result
	}

	// Check for manifest
	manifestPath := filepath.Join(rootPath, "nd-source.yaml")
	if manifest, err := loadManifest(manifestPath, rootPath); err != nil {
		result.Errors = append(result.Errors, err)
		return result
	} else if manifest != nil {
		scanWithManifest(&result, sourceID, rootPath, manifest)
		return result
	}

	// Convention-based discovery
	for dirName, assetType := range dirToAssetType {
		dirPath := filepath.Join(rootPath, dirName)
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			continue
		}

		if assetType == nd.AssetContext {
			scanContextDir(&result, sourceID, dirPath)
			continue
		}

		scanAssetDir(&result, sourceID, assetType, dirPath)
	}

	return result
}

// loadManifest reads and validates an nd-source.yaml file.
// Returns nil, nil if the file does not exist.
func loadManifest(path string, sourceRoot string) (*source.Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if info.Size() > maxManifestSize {
		return nil, fmt.Errorf("manifest %s is %d bytes, maximum is %d (NFR-013)", path, info.Size(), maxManifestSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m source.Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	if errs := m.Validate(sourceRoot); len(errs) > 0 {
		return nil, errs[0]
	}

	return &m, nil
}

// scanWithManifest scans using manifest-defined paths instead of conventions.
// Respects the manifest's Exclude list (FR-008).
func scanWithManifest(result *source.ScanResult, sourceID string, rootPath string, m *source.Manifest) {
	excludeSet := make(map[string]bool)
	for _, e := range m.Exclude {
		excludeSet[strings.TrimSuffix(e, "/")] = true
	}

	for assetType, paths := range m.Paths {
		for _, p := range paths {
			dirPath := filepath.Join(rootPath, p)
			info, err := os.Stat(dirPath)
			if err != nil || !info.IsDir() {
				result.Warnings = append(result.Warnings,
					"manifest path "+p+" for "+string(assetType)+" not found")
				continue
			}

			if assetType == nd.AssetContext {
				scanContextDir(result, sourceID, dirPath)
			} else {
				scanAssetDirExcluding(result, sourceID, assetType, dirPath, excludeSet)
			}
		}
	}
}

// scanAssetDirExcluding is like scanAssetDir but skips entries matching the exclude set.
func scanAssetDirExcluding(result *source.ScanResult, sourceID string, assetType nd.AssetType, dirPath string, excludeSet map[string]bool) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if excludedDirs[name] || name[0] == '.' || excludeSet[name] {
			continue
		}

		result.Assets = append(result.Assets, asset.Asset{
			Identity: asset.Identity{
				SourceID: sourceID,
				Type:     assetType,
				Name:     name,
			},
			SourcePath: filepath.Join(dirPath, name),
			IsDir:      entry.IsDir(),
		})
	}
}
```

Add `"fmt"` and `"strings"` to the import block if not already present.

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run "TestScan" -v`
Expected: PASS (all scanner tests)

### Step 5: Write the failing test for full Scan()

Add to `internal/sourcemanager/sourcemanager_test.go`:

```go
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
```

Move `makeSourceTree` from `scanner_test.go` to a shared test helper, or duplicate it in `sourcemanager_test.go`. Since both are in `sourcemanager_test` package, just keep it in `scanner_test.go` — it is accessible from `sourcemanager_test.go`.

### Step 6: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run "TestScan$|TestScanConflict" -v`
Expected: FAIL — `Scan` method not defined on SourceManager

### Step 7: Write implementation

Add to `internal/sourcemanager/sourcemanager.go`:

```go
import (
	"github.com/larah/nd/internal/asset"
)
```

Add method:

```go
// ScanSummary holds the result of a full scan across all sources.
type ScanSummary struct {
	Index    *asset.Index
	Warnings []string
}

// Scan discovers all assets across all registered sources and builds an index.
// Unavailable sources produce warnings but do not fail the scan (NFR-006).
func (sm *SourceManager) Scan() (*ScanSummary, error) {
	var allAssets []asset.Asset
	var allWarnings []string

	for _, entry := range sm.cfg.Sources {
		result := ScanSource(entry.ID, entry.Path)
		allAssets = append(allAssets, result.Assets...)
		allWarnings = append(allWarnings, result.Warnings...)
	}

	return &ScanSummary{
		Index:    asset.NewIndex(allAssets),
		Warnings: allWarnings,
	}, nil
}
```

### Step 8: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -v`
Expected: PASS (all tests)

### Step 9: Commit

```shell
git add internal/sourcemanager/scanner.go internal/sourcemanager/scanner_test.go internal/sourcemanager/sourcemanager.go internal/sourcemanager/sourcemanager_test.go
git commit -m "feat(sourcemanager): add manifest scanning and full Scan() method"
```

---

## Task 12: Git Sync

Implement `SyncSource()` to pull updates for Git sources.

**Files:**

- Modify: `internal/sourcemanager/sourcemanager.go`
- Modify: `internal/sourcemanager/sourcemanager_test.go`

### Step 1: Write the failing tests

Add to `internal/sourcemanager/sourcemanager_test.go`:

```go
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
	// Create a bare repo and clone it to simulate a git source
	bareRepo := t.TempDir()
	exec_git(t, "init", "--bare", bareRepo)

	cloneDir := t.TempDir()
	exec_git(t, "clone", bareRepo, cloneDir)

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
```

Add `"fmt"` to the imports if not already there.

### Step 2: Run tests to verify they fail

Run: `go test ./internal/sourcemanager/ -run TestSyncSource -v`
Expected: FAIL — `SyncSource` not defined

### Step 3: Write implementation

Add to `internal/sourcemanager/sourcemanager.go`:

```go
// SyncSource pulls updates for a Git source. Returns an error if the source
// is not found or is not a Git source. Uses --ff-only to avoid merge commits.
func (sm *SourceManager) SyncSource(sourceID string) error {
	var entry *config.SourceEntry
	for i := range sm.cfg.Sources {
		if sm.cfg.Sources[i].ID == sourceID {
			entry = &sm.cfg.Sources[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("source %q not found", sourceID)
	}

	if entry.Type != nd.SourceGit {
		return fmt.Errorf("source %q is type %q, not git", sourceID, entry.Type)
	}

	return gitPull(entry.Path)
}
```

Add `"github.com/larah/nd/internal/nd"` to the imports in `sourcemanager.go`.

### Step 4: Run tests to verify they pass

Run: `go test ./internal/sourcemanager/ -run TestSyncSource -v`
Expected: PASS

### Step 5: Run all tests with race detector

Run: `go test ./internal/sourcemanager/ -race -v`
Expected: PASS

### Step 6: Run full test suite

Run: `go test ./... -race`
Expected: PASS — no regressions in existing packages

### Step 7: Commit

```shell
git add internal/sourcemanager/sourcemanager.go internal/sourcemanager/sourcemanager_test.go
git commit -m "feat(sourcemanager): add SyncSource for Git source updates"
```

---

## Final Verification

After all 12 tasks are complete:

1. Run: `go test ./... -race -count=1`
   Expected: All tests pass with race detector

2. Run: `go vet ./...`
   Expected: No issues

3. Run: `gofumpt -l .` (check formatting)
   Expected: No output (all files formatted)

4. Verify coverage: `go test ./internal/sourcemanager/ -coverprofile=cover.out && go tool cover -func=cover.out`
   Expected: >80% coverage (NFR-009)
