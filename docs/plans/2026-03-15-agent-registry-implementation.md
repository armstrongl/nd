# Agent Registry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the Agent Registry service layer that detects installed coding agents, applies config overrides, and provides agent lookup.

**Architecture:** A `Registry` struct in `internal/agent/registry.go` with injected filesystem/PATH stubs for testability. Hardcoded Claude Code definition in v1. Config overrides applied during construction.

**Tech Stack:** Go 1.25, standard library (`os`, `os/exec`, `os/user`), `config.Config`

---

## Task 1: Registry struct and New constructor

**Files:**

- Create: `internal/agent/registry.go`
- Test: `internal/agent/registry_test.go`

### Step 1: Write the failing test

In `internal/agent/registry_test.go`:

```go
package agent_test

import (
	"os"
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/config"
)

func TestNewRegistryHasClaudeCode(t *testing.T) {
	cfg := config.Config{}
	r := agent.New(cfg)
	agents := r.All()
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	if agents[0].Name != "claude-code" {
		t.Errorf("got name %q, want %q", agents[0].Name, "claude-code")
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/agent/ -run TestNewRegistryHasClaudeCode -v`
Expected: FAIL — `agent.New` undefined

### Step 3: Write minimal implementation

In `internal/agent/registry.go`:

```go
package agent

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/armstrongl/nd/internal/config"
)

// Registry manages agent detection, lookup, and config override application.
type Registry struct {
	agents     []Agent
	defaultCfg string // config.DefaultAgent value
	detected   bool
	lookPath   func(string) (string, error)
	stat       func(string) (os.FileInfo, error)
}

// New creates a Registry with known agent definitions and applies config overrides.
func New(cfg config.Config) *Registry {
	homeDir := "~"
	if u, err := user.Current(); err == nil {
		homeDir = u.HomeDir
	}

	agents := []Agent{
		{
			Name:       "claude-code",
			GlobalDir:  filepath.Join(homeDir, ".claude"),
			ProjectDir: ".claude",
		},
	}

	for i := range agents {
		for _, override := range cfg.Agents {
			if override.Name == agents[i].Name {
				if override.GlobalDir != "" {
					agents[i].GlobalDir = expandHome(override.GlobalDir, homeDir)
				}
				if override.ProjectDir != "" {
					agents[i].ProjectDir = override.ProjectDir
				}
			}
		}
	}

	return &Registry{
		agents:     agents,
		defaultCfg: cfg.DefaultAgent,
		lookPath:   exec.LookPath,
		stat:       os.Stat,
	}
}

// All returns all known agents (detected or not).
func (r *Registry) All() []Agent {
	result := make([]Agent, len(r.agents))
	copy(result, r.agents)
	return result
}

func expandHome(path, homeDir string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	if path == "~" {
		return homeDir
	}
	return path
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/agent/ -run TestNewRegistryHasClaudeCode -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/agent/registry.go internal/agent/registry_test.go
git commit -m "feat(agent): add Registry struct and New constructor"
```

---

## Task 2: Config override application

**Files:**

- Modify: `internal/agent/registry_test.go`
- (No changes to `registry.go` — override logic is already in `New`)

### Step 1: Write the failing test for GlobalDir override

Add to `internal/agent/registry_test.go`:

```go
func TestNewRegistryAppliesGlobalDirOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: "/custom/global"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].GlobalDir != "/custom/global" {
		t.Errorf("got GlobalDir %q, want %q", agents[0].GlobalDir, "/custom/global")
	}
}

func TestNewRegistryAppliesProjectDirOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", ProjectDir: ".custom-claude"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].ProjectDir != ".custom-claude" {
		t.Errorf("got ProjectDir %q, want %q", agents[0].ProjectDir, ".custom-claude")
	}
}

func TestNewRegistryExpandsTildeInOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: "~/custom-claude"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if strings.HasPrefix(agents[0].GlobalDir, "~") {
		t.Errorf("tilde not expanded: got %q", agents[0].GlobalDir)
	}
	if !strings.HasSuffix(agents[0].GlobalDir, "/custom-claude") {
		t.Errorf("got GlobalDir %q, want suffix %q", agents[0].GlobalDir, "/custom-claude")
	}
}

func TestNewRegistryIgnoresUnknownAgentOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "unknown-agent", GlobalDir: "/somewhere"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	// Claude Code should keep its default GlobalDir (contains ".claude")
	if !strings.HasSuffix(agents[0].GlobalDir, ".claude") {
		t.Errorf("expected default GlobalDir, got %q", agents[0].GlobalDir)
	}
}
```

Add `"strings"` to the import block.

### Step 2: Run tests to verify they pass

Run: `go test ./internal/agent/ -run TestNewRegistry -v`
Expected: All 5 `TestNewRegistry*` tests PASS (override logic is already in `New` from Task 1)

### Step 3: Commit

```shell
git add internal/agent/registry_test.go
git commit -m "test(agent): add config override tests for Registry"
```

---

## Task 3: Detect method — detection logic

**Files:**

- Modify: `internal/agent/registry.go`
- Modify: `internal/agent/registry_test.go`

### Step 1: Write the failing tests

Add to `internal/agent/registry_test.go`:

```go
func stubRegistry(cfg config.Config, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error)) *agent.Registry {
	r := agent.New(cfg)
	r.SetLookPath(lookPath)
	r.SetStat(stat)
	return r
}

func lookPathFound(file string) (string, error) {
	return "/usr/local/bin/" + file, nil
}

func lookPathNotFound(file string) (string, error) {
	return "", exec.ErrNotFound
}

func statFound(path string) (os.FileInfo, error) {
	return nil, nil // non-nil FileInfo not needed for detection
}

func statNotFound(path string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func TestDetectPathAndDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	result := r.Detect()
	if len(result.Agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(result.Agents))
	}
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true")
	}
	if !result.Agents[0].InPath {
		t.Error("expected InPath=true")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestDetectPathNoDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statNotFound)
	result := r.Detect()
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true when in PATH")
	}
	if !result.Agents[0].InPath {
		t.Error("expected InPath=true")
	}
}

func TestDetectNoPATHWithDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathNotFound, statFound)
	result := r.Detect()
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true when dir exists")
	}
	if result.Agents[0].InPath {
		t.Error("expected InPath=false")
	}
}

func TestDetectNoPATHNoDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathNotFound, statNotFound)
	result := r.Detect()
	if result.Agents[0].Detected {
		t.Error("expected Detected=false")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning when no agents detected")
	}
}
```

Add `"os/exec"` to the import block.

### Step 2: Run tests to verify they fail

Run: `go test ./internal/agent/ -run TestDetect -v`
Expected: FAIL — `SetLookPath` and `SetStat` undefined, `Detect` not implemented

### Step 3: Write implementation

Add to `internal/agent/registry.go`:

```go
// SetLookPath replaces the PATH lookup function (for testing).
func (r *Registry) SetLookPath(fn func(string) (string, error)) {
	r.lookPath = fn
}

// SetStat replaces the filesystem stat function (for testing).
func (r *Registry) SetStat(fn func(string) (os.FileInfo, error)) {
	r.stat = fn
}

// agentBinaries maps agent names to their expected binary names in PATH.
var agentBinaries = map[string]string{
	"claude-code": "claude",
}

// Detect probes the system for installed agents (PATH + config dir).
// Populates Agent.Detected and Agent.InPath fields. Safe to call multiple
// times (subsequent calls are no-ops). Returns DetectionResult.
func (r *Registry) Detect() DetectionResult {
	if r.detected {
		return DetectionResult{Agents: r.All()}
	}

	var warnings []string

	anyDetected := false
	for i := range r.agents {
		binary := agentBinaries[r.agents[i].Name]

		// PATH check
		if binary != "" {
			if _, err := r.lookPath(binary); err == nil {
				r.agents[i].InPath = true
			}
		}

		// Config dir check
		dirExists := false
		if _, err := r.stat(r.agents[i].GlobalDir); err == nil {
			dirExists = true
		}

		// Detected if either PATH or dir exists
		r.agents[i].Detected = r.agents[i].InPath || dirExists

		if r.agents[i].Detected {
			anyDetected = true
		}
	}

	if !anyDetected {
		warnings = append(warnings,
			"No coding agents detected. Install Claude Code or configure a custom agent path in config.yaml (see: nd settings edit).")
	}

	r.detected = true

	return DetectionResult{
		Agents:   r.All(),
		Warnings: warnings,
	}
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/agent/ -run TestDetect -v`
Expected: All 4 `TestDetect*` tests PASS

### Step 5: Commit

```shell
git add internal/agent/registry.go internal/agent/registry_test.go
git commit -m "feat(agent): add Detect method with PATH and config dir checks"
```

---

## Task 4: Detect idempotency

**Files:**

- Modify: `internal/agent/registry_test.go`

### Step 1: Write the test

Add to `internal/agent/registry_test.go`:

```go
func TestDetectIsIdempotent(t *testing.T) {
	callCount := 0
	countingLookPath := func(file string) (string, error) {
		callCount++
		return "/usr/local/bin/" + file, nil
	}
	r := stubRegistry(config.Config{}, countingLookPath, statFound)

	r.Detect()
	r.Detect()
	r.Detect()

	if callCount != 1 {
		t.Errorf("lookPath called %d times, want 1 (idempotent)", callCount)
	}
}
```

### Step 2: Run test to verify it passes

Run: `go test ./internal/agent/ -run TestDetectIsIdempotent -v`
Expected: PASS (idempotency is already implemented via the `r.detected` guard)

### Step 3: Commit

```shell
git add internal/agent/registry_test.go
git commit -m "test(agent): add Detect idempotency test"
```

---

## Task 5: Get method

**Files:**

- Modify: `internal/agent/registry.go`
- Modify: `internal/agent/registry_test.go`

### Step 1: Write the failing tests

Add to `internal/agent/registry_test.go`:

```go
func TestGetFoundAgent(t *testing.T) {
	r := agent.New(config.Config{})
	a, err := r.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got name %q, want %q", a.Name, "claude-code")
	}
}

func TestGetUnknownAgent(t *testing.T) {
	r := agent.New(config.Config{})
	_, err := r.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/agent/ -run TestGet -v`
Expected: FAIL — `Get` undefined

### Step 3: Write implementation

Add to `internal/agent/registry.go`:

```go
// Get returns the agent with the given name, or an error if not found.
func (r *Registry) Get(name string) (*Agent, error) {
	for i := range r.agents {
		if r.agents[i].Name == name {
			return &r.agents[i], nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", name)
}
```

Add `"fmt"` to the import block.

### Step 4: Run tests to verify they pass

Run: `go test ./internal/agent/ -run TestGet -v`
Expected: PASS

### Step 5: Commit

```shell
git add internal/agent/registry.go internal/agent/registry_test.go
git commit -m "feat(agent): add Get method for agent lookup by name"
```

---

## Task 6: Default method

**Files:**

- Modify: `internal/agent/registry.go`
- Modify: `internal/agent/registry_test.go`

### Step 1: Write the failing tests

Add to `internal/agent/registry_test.go`:

```go
func TestDefaultReturnsConfiguredAgent(t *testing.T) {
	cfg := config.Config{DefaultAgent: "claude-code"}
	r := stubRegistry(cfg, lookPathFound, statFound)
	r.Detect()

	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}

func TestDefaultFallsBackToFirstDetected(t *testing.T) {
	cfg := config.Config{} // no DefaultAgent set
	r := stubRegistry(cfg, lookPathFound, statFound)
	r.Detect()

	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}

func TestDefaultErrorsWhenNoneDetected(t *testing.T) {
	cfg := config.Config{}
	r := stubRegistry(cfg, lookPathNotFound, statNotFound)
	r.Detect()

	_, err := r.Default()
	if err == nil {
		t.Error("expected error when no agents detected")
	}
}

func TestDefaultErrorsWhenConfiguredAgentNotDetected(t *testing.T) {
	cfg := config.Config{DefaultAgent: "claude-code"}
	r := stubRegistry(cfg, lookPathNotFound, statNotFound)
	r.Detect()

	_, err := r.Default()
	if err == nil {
		t.Error("expected error when configured default agent is not detected")
	}
}

func TestDefaultAutoDetectsIfNotCalled(t *testing.T) {
	cfg := config.Config{}
	r := stubRegistry(cfg, lookPathFound, statFound)
	// Don't call Detect() explicitly — Default() should trigger it

	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}
```

### Step 2: Run tests to verify they fail

Run: `go test ./internal/agent/ -run TestDefault -v`
Expected: FAIL — `Default` undefined

### Step 3: Write implementation

Add to `internal/agent/registry.go`:

```go
// Default returns the default agent: the one named in config.DefaultAgent if
// detected, otherwise the first detected agent, otherwise an error.
// Calls Detect() automatically if not already called.
func (r *Registry) Default() (*Agent, error) {
	if !r.detected {
		r.Detect()
	}

	// Try configured default first
	if r.defaultCfg != "" {
		for i := range r.agents {
			if r.agents[i].Name == r.defaultCfg && r.agents[i].Detected {
				return &r.agents[i], nil
			}
		}
	}

	// Fall back to first detected
	for i := range r.agents {
		if r.agents[i].Detected {
			return &r.agents[i], nil
		}
	}

	return nil, fmt.Errorf("no coding agents detected; install Claude Code or configure a custom agent path in config.yaml")
}
```

### Step 4: Run tests to verify they pass

Run: `go test ./internal/agent/ -run TestDefault -v`
Expected: All 5 `TestDefault*` tests PASS

### Step 5: Commit

```shell
git add internal/agent/registry.go internal/agent/registry_test.go
git commit -m "feat(agent): add Default method with config-named and fallback logic"
```

---

## Task 7: Run full test suite and verify coverage

**Files:**

- No changes

### Step 1: Run all agent package tests

Run: `go test ./internal/agent/ -v -count=1`
Expected: ALL tests pass (both existing `agent_test.go` and new `registry_test.go`)

### Step 2: Check coverage

Run: `go test ./internal/agent/ -coverprofile=coverage.out && go tool cover -func=coverage.out`
Expected: >90% coverage on `registry.go`

### Step 3: Run full project test suite

Run: `go test ./... -count=1`
Expected: All packages pass — no regressions

### Step 4: Clean up coverage file

Run: `rm coverage.out`

### Step 5: Commit (if any fixes were needed)

Only commit if fixes were made. Otherwise proceed.

---

## Task 8: Lint and final commit

**Files:**

- Possibly: `internal/agent/registry.go` (formatting fixes)

### Step 1: Run gofumpt

Run: `gofumpt -w internal/agent/registry.go`

### Step 2: Run golangci-lint

Run: `golangci-lint run ./internal/agent/`

### Step 3: Fix any lint issues and re-run

If issues found, fix and re-run until clean.

### Step 4: Run rumdl on the design doc

Run: `rumdl check docs/plans/2026-03-15-agent-registry-design.md`
Expected: No issues found

### Step 5: Final commit if any formatting changes

```shell
git add internal/agent/registry.go
git commit -m "style(agent): apply gofumpt formatting"
```
