<!-- rumdl-disable MD036 -->
# Data Types & Schemas Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all 48 Go types across 12 packages from the data types and schemas design, with full test coverage, validation methods, and YAML/JSON round-trip tests.

**Architecture:** Types are implemented bottom-up following the dependency graph: `internal/nd/` first (shared enums, constants, error types), then Level 1 packages (`config`, `asset`, `agent`, `backup`) in parallel, then Level 2 (`source`, `state`, `profile`, `oplog`), then Level 3 (`deploy`, `doctor`, `output`). Each package gets its types, methods, and validation implemented in TDD style.

**Tech Stack:** Go 1.23+, gopkg.in/yaml.v3, standard library only (no external validation or testing frameworks).

**Design doc:** `docs/plans/2026-03-14-data-types-and-schemas-design.md`

**Repo management design:** `docs/plans/2026-03-14-repo-management-design.md`

---

## Implementation Status

**COMPLETED** -- 2026-03-14

All 48 Go types across 12 packages implemented with 91.2% test coverage. 15 commits on `initial-setup` branch (fa8ddfa..717e3a4). All tests pass with race detector. Executed via 5-agent team (foundation + 4 workers).

---

## Task 1: Initialize Go module and directory structure

**Files:**

- Create: `go.mod`
- Create: `main.go`
- Create: `internal/nd/`, `internal/config/`, `internal/asset/`, `internal/source/`, `internal/state/`, `internal/profile/`, `internal/agent/`, `internal/deploy/`, `internal/backup/`, `internal/oplog/`, `internal/doctor/`, `internal/output/`

**Step 1: Initialize module and install yaml dependency**

```shell
cd /Users/larah/Repos/nd
go mod init github.com/armstrongl/nd
go get gopkg.in/yaml.v3
```

**Step 2: Create main.Go placeholder**

```go
// main.go
package main

import "fmt"

func main() {
	fmt.Println("nd - Napoleon Dynamite asset manager")
}
```

**Step 3: Create all package directories with .Go placeholder files**

Create a single `doc.go` in each package directory so Go recognizes them. Each file is just `package <name>` with a package doc comment.

Directories to create:

- `internal/nd/doc.go`
- `internal/config/doc.go`
- `internal/asset/doc.go`
- `internal/source/doc.go`
- `internal/state/doc.go`
- `internal/profile/doc.go`
- `internal/agent/doc.go`
- `internal/deploy/doc.go`
- `internal/backup/doc.go`
- `internal/oplog/doc.go`
- `internal/doctor/doc.go`
- `internal/output/doc.go`

**Step 4: Verify it compiles**

```shell
go build ./...
```

Expected: success, no errors.

**Step 5: Commit**

```shell
git add go.mod go.sum main.go internal/
git commit -m "feat: initialize Go module and package structure"
```

---

### Task 2: Implement `internal/nd/` — shared enums and constants

**Files:**

- Create: `internal/nd/asset_type.go`
- Create: `internal/nd/scope.go`
- Create: `internal/nd/origin.go`
- Create: `internal/nd/symlink.go`
- Create: `internal/nd/source_type.go`
- Create: `internal/nd/context.go`
- Create: `internal/nd/exit_codes.go`
- Create: `internal/nd/schema.go`
- Create: `internal/nd/file_kind.go`
- Test: `internal/nd/asset_type_test.go`
- Test: `internal/nd/origin_test.go`
- Test: `internal/nd/context_test.go`

This task implements all enums and constants from design section 1, minus the error types (Task 3).

**Step 1: Write failing tests for AssetType**

```go
// internal/nd/asset_type_test.go
package nd_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestAllAssetTypes(t *testing.T) {
	types := nd.AllAssetTypes()
	if len(types) != 8 {
		t.Fatalf("expected 8 asset types, got %d", len(types))
	}
}

func TestDeployableAssetTypes(t *testing.T) {
	types := nd.DeployableAssetTypes()
	for _, at := range types {
		if at == nd.AssetPlugin {
			t.Fatal("plugins should not be in deployable types")
		}
	}
	if len(types) != 7 {
		t.Fatalf("expected 7 deployable types, got %d", len(types))
	}
}

func TestAssetTypeIsDirectory(t *testing.T) {
	tests := []struct {
		at   nd.AssetType
		want bool
	}{
		{nd.AssetSkill, true},
		{nd.AssetPlugin, true},
		{nd.AssetHook, true},
		{nd.AssetAgent, false},
		{nd.AssetCommand, false},
		{nd.AssetOutputStyle, false},
		{nd.AssetRule, false},
		{nd.AssetContext, false},
	}
	for _, tt := range tests {
		if got := tt.at.IsDirectory(); got != tt.want {
			t.Errorf("%s.IsDirectory() = %v, want %v", tt.at, got, tt.want)
		}
	}
}

func TestAssetTypeDeploySubdir(t *testing.T) {
	if nd.AssetContext.DeploySubdir() != "" {
		t.Error("context should return empty deploy subdir")
	}
	if nd.AssetSkill.DeploySubdir() != "skills" {
		t.Errorf("skills should return 'skills', got %q", nd.AssetSkill.DeploySubdir())
	}
}

func TestAssetTypeIsDeployable(t *testing.T) {
	if nd.AssetPlugin.IsDeployable() {
		t.Error("plugins should not be deployable")
	}
	if !nd.AssetSkill.IsDeployable() {
		t.Error("skills should be deployable")
	}
}

func TestAssetTypeRequiresSettingsRegistration(t *testing.T) {
	if !nd.AssetHook.RequiresSettingsRegistration() {
		t.Error("hooks require settings registration")
	}
	if !nd.AssetOutputStyle.RequiresSettingsRegistration() {
		t.Error("output-styles require settings registration")
	}
	if nd.AssetSkill.RequiresSettingsRegistration() {
		t.Error("skills do not require settings registration")
	}
}
```

**Step 2: Run tests to verify they fail**

```shell
go test ./internal/nd/... -v
```

Expected: FAIL (types not defined yet)

**Step 3: Implement all enum files**

Implement `asset_type.go`, `scope.go`, `origin.go`, `symlink.go`, `source_type.go`, `context.go`, `exit_codes.go`, `schema.go`, `file_kind.go` exactly as defined in design section 1. Delete `doc.go` since the package now has real files.

**Step 4: Write and run tests for DeployOrigin**

```go
// internal/nd/origin_test.go
package nd_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestOriginProfile(t *testing.T) {
	o := nd.OriginProfile("go-backend")
	if o != "profile:go-backend" {
		t.Errorf("got %q", o)
	}
	if !o.IsProfile() {
		t.Error("should be a profile origin")
	}
	if o.ProfileName() != "go-backend" {
		t.Errorf("got %q", o.ProfileName())
	}
}

func TestOriginManualIsNotProfile(t *testing.T) {
	if nd.OriginManual.IsProfile() {
		t.Error("manual should not be a profile origin")
	}
	if nd.OriginManual.ProfileName() != "" {
		t.Error("manual profile name should be empty")
	}
}
```

**Step 5: Write and run tests for context helpers**

```go
// internal/nd/context_test.go
package nd_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestBuiltinContextFileNames(t *testing.T) {
	names := nd.BuiltinContextFileNames()
	if len(names) != 4 {
		t.Fatalf("expected 4 built-in context file names, got %d", len(names))
	}
}

func TestIsLocalOnlyContext(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"CLAUDE.md", false},
		{"AGENTS.md", false},
		{"CLAUDE.local.md", true},
		{"AGENTS.local.md", true},
		{"CUSTOM.local.md", true},
		{"short.md", false},
	}
	for _, tt := range tests {
		if got := nd.IsLocalOnlyContext(tt.name); got != tt.want {
			t.Errorf("IsLocalOnlyContext(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
```

**Step 6: Run all tests**

```shell
go test ./internal/nd/... -v
```

Expected: all PASS.

**Step 7: Commit**

```shell
git add internal/nd/
git commit -m "feat(nd): add shared enums, constants, and type methods"
```

---

### Task 3: Implement `internal/nd/` — domain error types

**Files:**

- Create: `internal/nd/errors.go`
- Test: `internal/nd/errors_test.go`

**Step 1: Write failing tests**

```go
// internal/nd/errors_test.go
package nd_test

import (
	"errors"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestPathTraversalError(t *testing.T) {
	err := &nd.PathTraversalError{
		Path:     "../../../etc/passwd",
		Root:     "/Users/dev/source",
		SourceID: "my-source",
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
	var pte *nd.PathTraversalError
	if !errors.As(err, &pte) {
		t.Error("should be assertable as PathTraversalError")
	}
}

func TestLockError(t *testing.T) {
	err := &nd.LockError{Path: "/path/to/lock", Timeout: "5s", Stale: false}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	stale := &nd.LockError{Path: "/path/to/lock", Timeout: "5s", Stale: true}
	if stale.Error() == err.Error() {
		t.Error("stale and non-stale messages should differ")
	}
}

func TestConflictError(t *testing.T) {
	err := &nd.ConflictError{
		TargetPath:   "/Users/dev/.claude/CLAUDE.md",
		ExistingKind: nd.FileKindPlainFile,
		AssetName:    "go-project-rules",
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}
```

**Step 2: Run tests to verify they fail**

```shell
go test ./internal/nd/... -v -run "Error|Lock|Conflict|Traversal"
```

Expected: FAIL

**Step 3: Implement errors.Go**

Implement `PathTraversalError`, `LockError`, and `ConflictError` exactly as defined in design section 1.

**Step 4: Run tests**

```shell
go test ./internal/nd/... -v
```

Expected: all PASS.

**Step 5: Commit**

```shell
git add internal/nd/errors.go internal/nd/errors_test.go
git commit -m "feat(nd): add domain error types (PathTraversal, Lock, Conflict)"
```

---

### Task 4: Implement `internal/config/` — configuration types

**Files:**

- Create: `internal/config/config.go`
- Create: `internal/config/validation.go`
- Test: `internal/config/config_test.go`

**Step 1: Write failing tests for YAML round-trip**

```go
// internal/config/config_test.go
package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
)

func TestConfigYAMLRoundTrip(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "my-assets", Type: nd.SourceLocal, Path: "/Users/dev/assets"},
		},
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got config.Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DefaultScope != nd.ScopeGlobal {
		t.Errorf("scope: got %q, want %q", got.DefaultScope, nd.ScopeGlobal)
	}
	if got.DefaultAgent != "claude-code" {
		t.Errorf("agent: got %q", got.DefaultAgent)
	}
	if len(got.Sources) != 1 || got.Sources[0].ID != "my-assets" {
		t.Errorf("sources: got %+v", got.Sources)
	}
}

func TestProjectConfigPointerSemantics(t *testing.T) {
	// Unset fields should not appear in YAML
	pc := config.ProjectConfig{Version: 1}
	data, err := yaml.Marshal(&pc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if contains(s, "default_scope") {
		t.Error("unset default_scope should not appear in YAML")
	}
	if contains(s, "default_agent") {
		t.Error("unset default_agent should not appear in YAML")
	}
}

func TestValidationErrorImplementsError(t *testing.T) {
	ve := config.ValidationError{
		File: "config.yaml", Line: 5, Field: "sources[0].path", Message: "path does not exist",
	}
	if ve.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run tests to verify they fail**

```shell
go test ./internal/config/... -v
```

Expected: FAIL

**Step 3: Implement config.Go and validation.Go**

`config.go`: all struct types (`Config`, `ProjectConfig`, `SourceEntry`, `AgentOverride`) with yaml/json tags exactly from design section 2.

`validation.go`: `ValidationError` struct and its `Error()` method. `Config.Validate()` as a stub that returns nil (full validation logic is a separate implementation task beyond types).

**Step 4: Run tests**

```shell
go test ./internal/config/... -v
```

Expected: all PASS.

**Step 5: Commit**

```shell
git add internal/config/
git commit -m "feat(config): add Config, ProjectConfig, and ValidationError types"
```

---

### Task 5: Implement `internal/asset/` — core asset types

**Files:**

- Create: `internal/asset/identity.go`
- Create: `internal/asset/asset.go`
- Create: `internal/asset/context.go`
- Create: `internal/asset/index.go`
- Create: `internal/asset/cache.go`
- Test: `internal/asset/identity_test.go`
- Test: `internal/asset/index_test.go`
- Test: `internal/asset/context_test.go`
- Test: `internal/asset/cache_test.go`

**Step 1: Write failing tests for Identity**

```go
// internal/asset/identity_test.go
package asset_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

func TestIdentityString(t *testing.T) {
	id := asset.Identity{SourceID: "my-src", Type: nd.AssetSkill, Name: "review"}
	got := id.String()
	want := "my-src:skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIdentityAsMapKey(t *testing.T) {
	id1 := asset.Identity{SourceID: "a", Type: nd.AssetSkill, Name: "x"}
	id2 := asset.Identity{SourceID: "a", Type: nd.AssetSkill, Name: "x"}
	id3 := asset.Identity{SourceID: "b", Type: nd.AssetSkill, Name: "x"}

	m := map[asset.Identity]bool{id1: true}
	if !m[id2] {
		t.Error("identical identities should match as map keys")
	}
	if m[id3] {
		t.Error("different identities should not match")
	}
}
```

**Step 2: Write failing tests for Index**

```go
// internal/asset/index_test.go
package asset_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

func makeAsset(source string, at nd.AssetType, name string) asset.Asset {
	return asset.Asset{
		Identity:   asset.Identity{SourceID: source, Type: at, Name: name},
		SourcePath: "/fake/" + name,
		IsDir:      at.IsDirectory(),
	}
}

func TestNewIndex(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "review"),
		makeAsset("src1", nd.AssetAgent, "go-dev"),
		makeAsset("src2", nd.AssetSkill, "deploy"),
	}
	idx := asset.NewIndex(assets)
	if len(idx.All()) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(idx.All()))
	}
}

func TestIndexLookup(t *testing.T) {
	assets := []asset.Asset{makeAsset("src1", nd.AssetSkill, "review")}
	idx := asset.NewIndex(assets)
	got := idx.Lookup(asset.Identity{SourceID: "src1", Type: nd.AssetSkill, Name: "review"})
	if got == nil {
		t.Fatal("expected to find asset")
	}
	if got.Name != "review" {
		t.Errorf("got name %q", got.Name)
	}

	missing := idx.Lookup(asset.Identity{SourceID: "nope", Type: nd.AssetSkill, Name: "nope"})
	if missing != nil {
		t.Error("expected nil for missing asset")
	}
}

func TestIndexByType(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "a"),
		makeAsset("src1", nd.AssetSkill, "b"),
		makeAsset("src1", nd.AssetAgent, "c"),
	}
	idx := asset.NewIndex(assets)
	skills := idx.ByType(nd.AssetSkill)
	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
}

func TestIndexBySource(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "a"),
		makeAsset("src2", nd.AssetSkill, "b"),
	}
	idx := asset.NewIndex(assets)
	src1 := idx.BySource("src1")
	if len(src1) != 1 {
		t.Errorf("expected 1, got %d", len(src1))
	}
}

func TestIndexConflictDetection(t *testing.T) {
	// Same (type, name) from two sources: first source wins
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "review"),
		makeAsset("src2", nd.AssetSkill, "review"),
	}
	idx := asset.NewIndex(assets)
	conflicts := idx.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Winner != "src1" {
		t.Errorf("winner should be src1, got %q", conflicts[0].Winner)
	}
	if conflicts[0].Loser != "src2" {
		t.Errorf("loser should be src2, got %q", conflicts[0].Loser)
	}
	// Only the winner should be in the index
	all := idx.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 asset after conflict, got %d", len(all))
	}
}
```

**Step 3: Write failing tests for CachedIndex**

```go
// internal/asset/cache_test.go
package asset_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/asset"
)

func TestCachedIndexYAMLRoundTrip(t *testing.T) {
	ci := asset.CachedIndex{
		Version:   1,
		SourceID:  "my-src",
		BuiltAt:   time.Now().Truncate(time.Second),
		SourceMod: time.Now().Add(-time.Hour).Truncate(time.Second),
	}
	data, err := yaml.Marshal(&ci)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got asset.CachedIndex
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SourceID != "my-src" {
		t.Errorf("source_id: got %q", got.SourceID)
	}
}

func TestCachedIndexIsStale(t *testing.T) {
	ci := asset.CachedIndex{
		SourceMod: time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC),
	}
	newer := time.Date(2026, 3, 14, 11, 0, 0, 0, time.UTC)
	if !ci.IsStale(newer) {
		t.Error("should be stale when source is newer")
	}
	older := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	if ci.IsStale(older) {
		t.Error("should not be stale when source is older")
	}
}
```

**Step 4: Write tests for ContextMeta YAML round-trip**

```go
// internal/asset/context_test.go
package asset_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/asset"
)

func TestContextMetaYAMLRoundTrip(t *testing.T) {
	meta := asset.ContextMeta{
		Description:    "Go project rules",
		Tags:           []string{"go", "backend"},
		TargetLanguage: "go",
	}
	data, err := yaml.Marshal(&meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got asset.ContextMeta
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Description != "Go project rules" {
		t.Errorf("description: got %q", got.Description)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags: got %d", len(got.Tags))
	}
}
```

**Step 5: Run all tests to verify they fail**

```shell
go test ./internal/asset/... -v
```

Expected: FAIL

**Step 6: Implement all asset files**

- `identity.go`: `Identity` struct and `String()` method
- `asset.go`: `Asset` struct
- `context.go`: `ContextInfo`, `ContextMeta` struct and `Validate()` stub
- `index.go`: `Index` struct, `Conflict`, `NewIndex()`, `Lookup`, `ByType`, `BySource`, `All`, `Conflicts`
- `cache.go`: `CachedIndex` struct and `IsStale()` method

Key implementation detail for `NewIndex`: use a `map[conflictKey]` where `conflictKey` is `(AssetType, lowercase Name)` to detect cross-source conflicts. The first asset encountered for a given key wins (assets are passed in source-registration order).

**Step 7: Run tests**

```shell
go test ./internal/asset/... -v
```

Expected: all PASS.

**Step 8: Commit**

```shell
git add internal/asset/
git commit -m "feat(asset): add Identity, Asset, Index, CachedIndex, and ContextMeta types"
```

---

### Task 6: Implement `internal/agent/` — agent registry types

**Files:**

- Create: `internal/agent/agent.go`
- Test: `internal/agent/agent_test.go`

**Step 1: Write failing tests**

```go
// internal/agent/agent_test.go
package agent_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/nd"
)

func claudeCode() agent.Agent {
	return agent.Agent{
		Name:       "claude-code",
		GlobalDir:  "/Users/dev/.claude",
		ProjectDir: ".claude",
		Detected:   true,
		InPath:     true,
	}
}

func TestDeployPathSkillGlobal(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeGlobal, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathSkillProject(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeProject, "/Users/dev/myapp", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/myapp/.claude/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathContextGlobal(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetContext, "go-rules", nd.ScopeGlobal, "", nd.ContextCLAUDE)
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/CLAUDE.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathContextProjectRoot(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetContext, "go-rules", nd.ScopeProject, "/Users/dev/myapp", nd.ContextCLAUDE)
	if err != nil {
		t.Fatal(err)
	}
	// Context files deploy to project root, not inside .claude/
	want := "/Users/dev/myapp/CLAUDE.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathLocalOnlyContextRejectsGlobal(t *testing.T) {
	a := claudeCode()
	_, err := a.DeployPath(nd.AssetContext, "local-rules", nd.ScopeGlobal, "", nd.ContextCLAUDELocal)
	if err == nil {
		t.Error("should reject global scope for .local.md context files")
	}
}

func TestDeployPathAgentFile(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetAgent, "go-specialist.md", nd.ScopeGlobal, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/agents/go-specialist.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

**Step 2: Run to verify failure, implement, run to verify pass**

```shell
go test ./internal/agent/... -v
```

**Step 3: Implement agent.Go**

`Agent` struct, `DetectionResult` struct, `DeployPath` method with all the context file special cases.

**Step 4: Run tests**

```shell
go test ./internal/agent/... -v
```

Expected: all PASS.

**Step 5: Commit**

```shell
git add internal/agent/
git commit -m "feat(agent): add Agent type with DeployPath and DetectionResult"
```

---

### Task 7: Implement `internal/backup/` — backup types

**Files:**

- Create: `internal/backup/backup.go`
- Test: `internal/backup/backup_test.go`

**Step 1: Write failing tests**

```go
// internal/backup/backup_test.go
package backup_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/backup"
	"github.com/armstrongl/nd/internal/nd"
)

func TestBackupJSONRoundTrip(t *testing.T) {
	b := backup.Backup{
		OriginalPath: "/Users/dev/.claude/CLAUDE.md",
		BackupPath:   "/Users/dev/.config/nd/backups/CLAUDE.md.2026-03-14T10-30-00.bak",
		CreatedAt:    time.Now().Truncate(time.Second),
		OriginalKind: nd.FileKindPlainFile,
	}
	data, err := json.Marshal(&b)
	if err != nil {
		t.Fatal(err)
	}
	var got backup.Backup
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.OriginalKind != nd.FileKindPlainFile {
		t.Errorf("kind: got %q", got.OriginalKind)
	}
}
```

**Step 2: Implement, test, commit**

```shell
go test ./internal/backup/... -v
git add internal/backup/
git commit -m "feat(backup): add Backup type with OriginalFileKind"
```

---

### Task 8: Implement `internal/source/` — source and manifest types

**Files:**

- Create: `internal/source/source.go`
- Create: `internal/source/manifest.go`
- Create: `internal/source/scan.go`
- Test: `internal/source/manifest_test.go`
- Test: `internal/source/source_test.go`

**Step 1: Write failing tests for Manifest YAML round-trip**

```go
// internal/source/manifest_test.go
package source_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/source"
)

func TestManifestYAMLRoundTrip(t *testing.T) {
	m := source.Manifest{
		Version: 1,
		Paths: map[nd.AssetType][]string{
			nd.AssetSkill: {"skills/", "go-skills/skills/"},
			nd.AssetAgent: {"agents/"},
		},
		Exclude: []string{"experimental/"},
	}
	data, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	var got source.Manifest
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Paths[nd.AssetSkill]) != 2 {
		t.Errorf("skills paths: got %d", len(got.Paths[nd.AssetSkill]))
	}
}

func TestManifestValidatePathTraversal(t *testing.T) {
	m := source.Manifest{
		Version: 1,
		Paths: map[nd.AssetType][]string{
			nd.AssetSkill: {"../../../etc/"},
		},
	}
	errs := m.Validate("/Users/dev/source")
	if len(errs) == 0 {
		t.Error("should reject path traversal")
	}
}

func TestManifestValidateTooManyPaths(t *testing.T) {
	paths := make([]string, 1001)
	for i := range paths {
		paths[i] = "dir/"
	}
	m := source.Manifest{
		Version: 1,
		Paths:   map[nd.AssetType][]string{nd.AssetSkill: paths},
	}
	errs := m.Validate("/Users/dev/source")
	if len(errs) == 0 {
		t.Error("should reject >1000 path entries")
	}
}
```

**Step 2: Run to verify failure, implement all source files, run to verify pass**

- `source.go`: `Source` struct
- `manifest.go`: `Manifest`, `ManifestMetadata` structs, `Validate()` method with path traversal and limit checks
- `scan.go`: `ScanResult` struct

**Step 3: Commit**

```shell
go test ./internal/source/... -v
git add internal/source/
git commit -m "feat(source): add Source, Manifest, and ScanResult types with validation"
```

---

### Task 9: Implement `internal/state/` — deployment state types

**Files:**

- Create: `internal/state/state.go`
- Create: `internal/state/health.go`
- Create: `internal/state/queries.go`
- Create: `internal/state/lock.go`
- Test: `internal/state/state_test.go`
- Test: `internal/state/health_test.go`
- Test: `internal/state/queries_test.go`

**Step 1: Write failing tests for DeploymentState YAML round-trip**

```go
// internal/state/state_test.go
package state_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

func TestDeploymentStateYAMLRoundTrip(t *testing.T) {
	s := state.DeploymentState{
		Version:       1,
		ActiveProfile: "go-backend",
		Deployments: []state.Deployment{
			{
				SourceID:   "my-assets",
				AssetType:  nd.AssetSkill,
				AssetName:  "code-review",
				SourcePath: "/Users/dev/assets/skills/code-review",
				LinkPath:   "/Users/dev/.claude/skills/code-review",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
				DeployedAt: time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC),
			},
		},
	}

	data, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var got state.DeploymentState
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ActiveProfile != "go-backend" {
		t.Errorf("active_profile: got %q", got.ActiveProfile)
	}
	if len(got.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(got.Deployments))
	}
	d := got.Deployments[0]
	if d.AssetName != "code-review" {
		t.Errorf("asset_name: got %q", d.AssetName)
	}
	if d.Scope != nd.ScopeGlobal {
		t.Errorf("scope: got %q", d.Scope)
	}
}

func TestDeploymentIdentity(t *testing.T) {
	d := state.Deployment{
		SourceID:  "src",
		AssetType: nd.AssetSkill,
		AssetName: "review",
	}
	id := d.Identity()
	if id.SourceID != "src" || id.Name != "review" {
		t.Errorf("unexpected identity: %+v", id)
	}
}
```

**Step 2: Write failing tests for HealthStatus and query methods**

```go
// internal/state/health_test.go
package state_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/state"
)

func TestHealthStatusString(t *testing.T) {
	tests := []struct {
		h    state.HealthStatus
		want string
	}{
		{state.HealthOK, "ok"},
		{state.HealthBroken, "broken"},
		{state.HealthDrifted, "drifted"},
		{state.HealthOrphaned, "orphaned"},
		{state.HealthMissing, "missing"},
		{state.HealthStatus(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.h.String(); got != tt.want {
			t.Errorf("HealthStatus(%d).String() = %q, want %q", tt.h, got, tt.want)
		}
	}
}
```

```go
// internal/state/queries_test.go
package state_test

import (
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/state"
)

func sampleState() state.DeploymentState {
	return state.DeploymentState{
		Version: 1,
		Deployments: []state.Deployment{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a",
				Scope: nd.ScopeGlobal, Origin: nd.OriginManual, DeployedAt: time.Now()},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "b",
				Scope: nd.ScopeProject, ProjectPath: "/proj", Origin: nd.OriginProfile("go"), DeployedAt: time.Now()},
			{SourceID: "s2", AssetType: nd.AssetSkill, AssetName: "c",
				Scope: nd.ScopeGlobal, Origin: nd.OriginPinned, DeployedAt: time.Now()},
		},
	}
}

func TestFindByIdentity(t *testing.T) {
	s := sampleState()
	got := s.FindByIdentity(asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "a"})
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
}

func TestFindByScope(t *testing.T) {
	s := sampleState()
	globals := s.FindByScope(nd.ScopeGlobal)
	if len(globals) != 2 {
		t.Errorf("expected 2 global, got %d", len(globals))
	}
}

func TestFindByOrigin(t *testing.T) {
	s := sampleState()
	pinned := s.FindByOrigin(nd.OriginPinned)
	if len(pinned) != 1 {
		t.Errorf("expected 1 pinned, got %d", len(pinned))
	}
}

func TestFindByProject(t *testing.T) {
	s := sampleState()
	proj := s.FindByProject("/proj")
	if len(proj) != 1 {
		t.Errorf("expected 1, got %d", len(proj))
	}
}
```

**Step 3: Run to verify failure, implement, run to verify pass**

- `state.go`: `DeploymentState`, `Deployment` structs, `Identity()` method, `Validate()` stub
- `health.go`: `HealthStatus` type, constants, `String()`, `HealthCheck` struct
- `queries.go`: `FindByIdentity`, `FindByScope`, `FindByOrigin`, `FindByProject`
- `lock.go`: `FileLock` struct, `Acquire()` and `Release()` stubs (actual flock implementation is beyond types-only scope, but the type and method signatures are defined)

**Step 4: Run tests**

```shell
go test ./internal/state/... -v
```

Expected: all PASS.

**Step 5: Commit**

```shell
git add internal/state/
git commit -m "feat(state): add DeploymentState, Deployment, HealthStatus, queries, and FileLock types"
```

---

### Task 10: Implement `internal/profile/` — profile and snapshot types

**Files:**

- Create: `internal/profile/profile.go`
- Create: `internal/profile/snapshot.go`
- Create: `internal/profile/switch_diff.go`
- Test: `internal/profile/profile_test.go`
- Test: `internal/profile/snapshot_test.go`
- Test: `internal/profile/switch_diff_test.go`

**Step 1: Write failing tests for Profile**

```go
// internal/profile/profile_test.go
package profile_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
)

func TestProfileYAMLRoundTrip(t *testing.T) {
	p := profile.Profile{
		Version:   1,
		Name:      "go-backend",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "my-src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal},
		},
	}
	data, err := yaml.Marshal(&p)
	if err != nil {
		t.Fatal(err)
	}
	var got profile.Profile
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "go-backend" {
		t.Errorf("name: got %q", got.Name)
	}
	if len(got.Assets) != 1 {
		t.Fatalf("assets: got %d", len(got.Assets))
	}
}

func TestProfileValidateRejectsPlugins(t *testing.T) {
	p := profile.Profile{
		Version: 1,
		Name:    "bad",
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetPlugin, AssetName: "p", Scope: nd.ScopeGlobal},
		},
	}
	errs := p.Validate()
	if len(errs) == 0 {
		t.Error("should reject plugin assets in profiles")
	}
}

func TestProfileAssetIdentity(t *testing.T) {
	pa := profile.ProfileAsset{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x"}
	id := pa.Identity()
	if id.SourceID != "s" || id.Name != "x" {
		t.Errorf("unexpected identity: %+v", id)
	}
}
```

**Step 2: Write failing tests for Snapshot**

```go
// internal/profile/snapshot_test.go
package profile_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
)

func TestSnapshotYAMLRoundTrip(t *testing.T) {
	s := profile.Snapshot{
		Version:   1,
		Name:      "before-switch",
		CreatedAt: time.Now().Truncate(time.Second),
		Auto:      true,
		Deployments: []profile.SnapshotEntry{
			{
				SourceID:   "src",
				AssetType:  nd.AssetSkill,
				AssetName:  "review",
				SourcePath: "/a/b",
				LinkPath:   "/c/d",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
				DeployedAt: time.Now().Truncate(time.Second),
			},
		},
	}
	data, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var got profile.Snapshot
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Auto {
		t.Error("auto should be true")
	}
	if len(got.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(got.Deployments))
	}
	if got.Deployments[0].DeployedAt.IsZero() {
		t.Error("deployed_at should be preserved in snapshot entries")
	}
}

func TestSnapshotValidateRejectsPlugins(t *testing.T) {
	s := profile.Snapshot{
		Version: 1,
		Name:    "bad",
		Deployments: []profile.SnapshotEntry{
			{AssetType: nd.AssetPlugin, AssetName: "p"},
		},
	}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Error("should reject plugin assets in snapshots")
	}
}
```

**Step 3: Write failing tests for SwitchDiff**

```go
// internal/profile/switch_diff_test.go
package profile_test

import (
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
)

func TestComputeSwitchDiff(t *testing.T) {
	now := time.Now()
	current := &profile.Profile{
		Version: 1, Name: "current", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "b", Scope: nd.ScopeGlobal},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "c", Scope: nd.ScopeProject},
		},
	}
	target := &profile.Profile{
		Version: 1, Name: "target", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},  // keep
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "d", Scope: nd.ScopeGlobal},  // deploy
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "c", Scope: nd.ScopeGlobal},  // different scope = remove + deploy
		},
	}

	diff := profile.ComputeSwitchDiff(current, target)

	if len(diff.Keep) != 1 {
		t.Errorf("keep: expected 1, got %d", len(diff.Keep))
	}
	// b is removed, c@project is removed (scope changed)
	if len(diff.Remove) != 2 {
		t.Errorf("remove: expected 2, got %d", len(diff.Remove))
	}
	// d is deployed, c@global is deployed (scope changed)
	if len(diff.Deploy) != 2 {
		t.Errorf("deploy: expected 2, got %d", len(diff.Deploy))
	}
}

func TestComputeSwitchDiffEmpty(t *testing.T) {
	now := time.Now()
	empty := &profile.Profile{Version: 1, Name: "empty", CreatedAt: now, UpdatedAt: now}
	full := &profile.Profile{
		Version: 1, Name: "full", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x", Scope: nd.ScopeGlobal},
		},
	}
	diff := profile.ComputeSwitchDiff(empty, full)
	if len(diff.Keep) != 0 {
		t.Errorf("keep: expected 0, got %d", len(diff.Keep))
	}
	if len(diff.Remove) != 0 {
		t.Errorf("remove: expected 0, got %d", len(diff.Remove))
	}
	if len(diff.Deploy) != 1 {
		t.Errorf("deploy: expected 1, got %d", len(diff.Deploy))
	}
}
```

**Step 4: Run to verify failure, implement, run to verify pass**

- `profile.go`: `Profile`, `ProfileAsset`, `Identity()`, `Validate()` (reject plugins)
- `snapshot.go`: `Snapshot`, `SnapshotEntry`, `Validate()` (reject plugins)
- `switch_diff.go`: `SwitchDiff`, `ComputeSwitchDiff` with four-tuple equality (source_id, asset_type, asset_name, scope)

**Step 5: Run tests**

```shell
go test ./internal/profile/... -v
```

Expected: all PASS.

**Step 6: Commit**

```shell
git add internal/profile/
git commit -m "feat(profile): add Profile, Snapshot, SwitchDiff with plugin validation"
```

---

### Task 11: Implement `internal/oplog/` — operation log types

**Files:**

- Create: `internal/oplog/oplog.go`
- Test: `internal/oplog/oplog_test.go`

**Step 1: Write failing tests**

```go
// internal/oplog/oplog_test.go
package oplog_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/oplog"
)

func TestLogEntryJSONRoundTrip(t *testing.T) {
	entry := oplog.LogEntry{
		Timestamp: time.Now().Truncate(time.Second),
		Operation: oplog.OpDeploy,
		Assets:    []asset.Identity{{SourceID: "s", Type: nd.AssetSkill, Name: "x"}},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
		Failed:    0,
	}
	data, err := json.Marshal(&entry)
	if err != nil {
		t.Fatal(err)
	}
	var got oplog.LogEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Operation != oplog.OpDeploy {
		t.Errorf("operation: got %q", got.Operation)
	}
}
```

**Step 2: Implement, test, commit**

```shell
go test ./internal/oplog/... -v
git add internal/oplog/
git commit -m "feat(oplog): add LogEntry and OperationType types"
```

---

### Task 12: Implement `internal/deploy/` — deploy engine types

**Files:**

- Create: `internal/deploy/request.go`
- Create: `internal/deploy/result.go`
- Create: `internal/deploy/action.go`
- Create: `internal/deploy/bulk.go`
- Create: `internal/deploy/sync.go`
- Create: `internal/deploy/uninstall.go`
- Test: `internal/deploy/action_test.go`
- Test: `internal/deploy/bulk_test.go`
- Test: `internal/deploy/result_test.go`

**Step 1: Write failing tests**

```go
// internal/deploy/action_test.go
package deploy_test

import (
	"encoding/json"
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
)

func TestActionString(t *testing.T) {
	tests := []struct {
		a    deploy.Action
		want string
	}{
		{deploy.ActionCreated, "created"},
		{deploy.ActionRemoved, "removed"},
		{deploy.ActionReplaced, "replaced"},
		{deploy.ActionSkipped, "skipped"},
		{deploy.ActionBackedUp, "backed-up"},
		{deploy.ActionFailed, "failed"},
		{deploy.ActionDryRun, "dry-run"},
		{deploy.Action(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("Action(%d).String() = %q, want %q", tt.a, got, tt.want)
		}
	}
}

func TestActionMarshalJSON(t *testing.T) {
	data, err := json.Marshal(deploy.ActionCreated)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"created"` {
		t.Errorf("got %s", data)
	}
}
```

```go
// internal/deploy/bulk_test.go
package deploy_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
)

func TestBulkResultHasFailures(t *testing.T) {
	br := deploy.BulkResult{Succeeded: 3, Failed: 0}
	if br.HasFailures() {
		t.Error("should not have failures")
	}
	br.Failed = 1
	if !br.HasFailures() {
		t.Error("should have failures")
	}
}

func TestBulkResultFailedResults(t *testing.T) {
	br := deploy.BulkResult{
		Results: []deploy.Result{
			{Success: true, Action: deploy.ActionCreated},
			{Success: false, Action: deploy.ActionFailed, ErrorMsg: "permission denied"},
			{Success: true, Action: deploy.ActionCreated},
		},
		Succeeded: 2,
		Failed:    1,
	}
	failed := br.FailedResults()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed, got %d", len(failed))
	}
	if failed[0].ErrorMsg != "permission denied" {
		t.Errorf("error: got %q", failed[0].ErrorMsg)
	}
}
```

**Step 2: Run to verify failure, implement all deploy files, run to verify pass**

- `request.go`: `Request` struct
- `result.go`: `Result` struct
- `action.go`: `Action` type, constants, `String()`, `MarshalJSON()`
- `bulk.go`: `BulkResult`, `HasFailures()`, `FailedResults()`
- `sync.go`: `SyncPlan`, `SyncAction`
- `uninstall.go`: `UninstallPlan`

**Step 3: Run tests**

```shell
go test ./internal/deploy/... -v
```

Expected: all PASS.

**Step 4: Commit**

```shell
git add internal/deploy/
git commit -m "feat(deploy): add Request, Result, Action, BulkResult, SyncPlan, UninstallPlan types"
```

---

### Task 13: Implement `internal/doctor/` — doctor report types

**Files:**

- Create: `internal/doctor/report.go`
- Test: `internal/doctor/report_test.go`

**Step 1: Write failing tests**

```go
// internal/doctor/report_test.go
package doctor_test

import (
	"encoding/json"
	"testing"

	"github.com/armstrongl/nd/internal/doctor"
)

func TestReportJSONRoundTrip(t *testing.T) {
	r := doctor.Report{
		Config: doctor.ConfigCheck{GlobalValid: true, ProjectValid: true},
		Git:    doctor.GitCheck{Available: true, Version: "2.44.0"},
		Summary: doctor.Summary{Pass: 5, Warn: 1, Fail: 0},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var got doctor.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.Pass != 5 {
		t.Errorf("pass: got %d", got.Summary.Pass)
	}
	if !got.Git.Available {
		t.Error("git should be available")
	}
}
```

**Step 2: Implement, test, commit**

```shell
go test ./internal/doctor/... -v
git add internal/doctor/
git commit -m "feat(doctor): add DoctorReport, ConfigCheck, SourceCheck, AgentCheck, GitCheck types"
```

---

### Task 14: Implement `internal/output/` — JSON envelope types

**Files:**

- Create: `internal/output/json.go`
- Test: `internal/output/json_test.go`

**Step 1: Write failing tests**

```go
// internal/output/json_test.go
package output_test

import (
	"encoding/json"
	"testing"

	"github.com/armstrongl/nd/internal/output"
)

func TestJSONResponseOK(t *testing.T) {
	r := output.JSONResponse{
		Status: "ok",
		Data:   map[string]int{"count": 3},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var got output.JSONResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "ok" {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestJSONResponseError(t *testing.T) {
	r := output.JSONResponse{
		Status: "error",
		Errors: []output.JSONError{
			{Code: "INVALID_CONFIG", Message: "bad config", Field: "sources[0].path"},
		},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty JSON")
	}
}
```

**Step 2: Implement, test, commit**

```shell
go test ./internal/output/... -v
git add internal/output/
git commit -m "feat(output): add JSONResponse and JSONError envelope types"
```

---

### Task 15: Full test suite and coverage check

**Step 1: Run all tests with race detector**

```shell
go test -race ./... -v
```

Expected: all PASS, no race conditions.

**Step 2: Check test coverage**

```shell
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Expected: >80% coverage on all `internal/` packages.

**Step 3: Verify build**

```shell
go build ./...
go vet ./...
```

Expected: clean build, no vet warnings.

**Step 4: Commit coverage baseline (do not commit the file)**

```shell
git add -A
git commit -m "test: complete test suite for all 48 data types across 12 packages"
```

---

### Task 16: Clean up placeholder files and final verification

**Step 1: Remove any remaining doc.Go placeholder files that are now redundant**

If a package has real `.go` files, the `doc.go` created in Task 1 can be removed (unless it contains a useful package doc comment, in which case keep it).

**Step 2: Run gofumpt**

```shell
gofumpt -w .
```

**Step 3: Final full test run**

```shell
go test -race ./... -count=1
```

Expected: all PASS.

**Step 4: Commit and verify clean state**

```shell
git add -A
git status
git commit -m "chore: clean up placeholders and format with gofumpt"
```
