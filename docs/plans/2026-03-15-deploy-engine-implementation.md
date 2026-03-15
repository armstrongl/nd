# Deploy Engine Implementation Plan

> **For agentic workers:** REQUIRED: Use supapowers:subagent-driven-development (if subagents available) or supapowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Deploy Engine — the central symlink orchestrator for nd — adding state persistence (`state.Store`) and the deploy package (`deploy.Engine`) with deploy, remove, health check, sync, and status operations.

**Architecture:** Two additions: (1) `state.Store` added to the existing `internal/state/` package for loading, saving, and locking `deployments.yaml`, and (2) a new `internal/deploy/` package containing the `Engine` that manages symlink lifecycle. The engine accepts a `StateStore` interface for testability. All operations wrap Load-mutate-Save in file locks (NFR-011).

**Tech Stack:** Go 1.25, gopkg.in/yaml.v3, syscall (flock), standard library (os, path/filepath, fmt, time, errors)

**Design doc:** `docs/plans/2026-03-15-deploy-engine-design.md`

**Existing types used:**
- `internal/state/state.go` — `DeploymentState`, `Deployment`
- `internal/state/queries.go` — `FindByIdentity`, `FindByScope`, etc.
- `internal/state/health.go` — `HealthStatus` (iota), `HealthCheck`
- `internal/state/lock.go` — `FileLock` (stub, needs implementation)
- `internal/nd/atomic.go` — `AtomicWrite`
- `internal/nd/errors.go` — `LockError`, `ConflictError`, `PathTraversalError`
- `internal/nd/schema.go` — `SchemaVersion` (const = 1)
- `internal/nd/file_kind.go` — `OriginalFileKind`
- `internal/nd/asset_type.go` — `AssetType`, `IsDeployable`, `DeploySubdir`, etc.
- `internal/nd/origin.go` — `DeployOrigin`, `OriginManual`, `OriginPinned`, `OriginProfile`
- `internal/nd/scope.go` — `Scope`, `ScopeGlobal`, `ScopeProject`
- `internal/nd/context.go` — `IsLocalOnlyContext`
- `internal/agent/agent.go` — `Agent`, `DeployPath`
- `internal/asset/asset.go` — `Asset`, `Identity`
- `internal/asset/context.go` — `ContextInfo`, `ContextMeta`
- `internal/asset/index.go` — `Index`

---

## File structure

| File | Responsibility | New/Modify |
| --- | --- | --- |
| `internal/state/lock.go` | Implement FileLock.Acquire/Release using flock(2) | Modify |
| `internal/state/lock_test.go` | Tests for file locking | Create |
| `internal/state/store.go` | Store struct: NewStore, Load, Save, WithLock | Create |
| `internal/state/store_test.go` | Tests for state persistence | Create |
| `internal/deploy/deploy.go` | StateStore interface, Engine, New, Deploy, DeployBulk, Remove, RemoveBulk, types | Create |
| `internal/deploy/health.go` | Check, Sync, Status, backup logic | Create |
| `internal/deploy/deploy_test.go` | Tests for deploy/remove | Create |
| `internal/deploy/health_test.go` | Tests for check/sync/status/backup | Create |

---

## Task 1: Implement FileLock (flock)

The existing `FileLock` in `internal/state/lock.go` is stubbed (Acquire/Release return nil). Implement real advisory file locking using `syscall.Flock`.

**Files:**
- Modify: `internal/state/lock.go`
- Create: `internal/state/lock_test.go`

- [ ] **Step 1: Write failing tests for FileLock**

```go
// internal/state/lock_test.go
package state_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

func TestFileLockAcquireRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock := state.NewFileLock(lockPath)
	if err := lock.Acquire(5 * time.Second); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if lock.AcquiredAt.IsZero() {
		t.Error("AcquiredAt should be set after Acquire")
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

func TestFileLockBlocksConcurrent(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock1 := state.NewFileLock(lockPath)
	if err := lock1.Acquire(5 * time.Second); err != nil {
		t.Fatalf("lock1 Acquire: %v", err)
	}
	defer lock1.Release()

	lock2 := state.NewFileLock(lockPath)
	err := lock2.Acquire(200 * time.Millisecond)
	if err == nil {
		lock2.Release()
		t.Fatal("expected lock2 to fail, got nil")
	}

	var lockErr *nd.LockError
	if !errors.As(err, &lockErr) {
		t.Errorf("expected *nd.LockError, got %T: %v", err, err)
	}
}

func TestFileLockReleaseUnlocks(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock1 := state.NewFileLock(lockPath)
	if err := lock1.Acquire(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	lock1.Release()

	lock2 := state.NewFileLock(lockPath)
	if err := lock2.Acquire(1 * time.Second); err != nil {
		t.Fatalf("lock2 should succeed after release: %v", err)
	}
	lock2.Release()
}

func TestFileLockDoubleRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	lock := state.NewFileLock(lockPath)
	lock.Acquire(5 * time.Second)
	lock.Release()
	// Second release should not panic or error
	if err := lock.Release(); err != nil {
		t.Errorf("double release should be safe: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/ -run TestFileLock -v`
Expected: FAIL — `NewFileLock` undefined, methods return nil (no real locking)

- [ ] **Step 3: Implement FileLock with flock(2)**

Replace the stub implementation in `internal/state/lock.go`:

```go
package state

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/larah/nd/internal/nd"
)

// FileLock provides advisory file locking on deployments.yaml.
// Acquired before read-modify-write cycles, released after the atomic rename.
type FileLock struct {
	Path       string
	AcquiredAt time.Time
	file       *os.File
}

// NewFileLock creates a FileLock for the given path.
func NewFileLock(path string) *FileLock {
	return &FileLock{Path: path}
}

// Acquire attempts to acquire an exclusive flock within the given timeout.
// Returns *nd.LockError if the lock cannot be acquired.
func (l *FileLock) Acquire(timeout time.Duration) error {
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file %s: %w", l.Path, err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			l.file = f
			l.AcquiredAt = time.Now()
			return nil
		}

		if time.Now().After(deadline) {
			f.Close()
			return &nd.LockError{
				Path:    l.Path,
				Timeout: timeout.String(),
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Release releases the file lock. Safe to call multiple times.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	l.AcquiredAt = time.Time{}
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/state/ -run TestFileLock -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Run full state package tests to check for regressions**

Run: `go test ./internal/state/ -v`
Expected: PASS (existing tests + new lock tests)

- [ ] **Step 6: Commit**

```bash
git add internal/state/lock.go internal/state/lock_test.go
git commit -m "feat(state): implement FileLock with flock(2) advisory locking (NFR-011)"
```

---

## Task 2: Implement state.Store (Load, Save, WithLock)

Add state persistence to the existing `internal/state/` package. The Store manages `deployments.yaml` on disk with atomic writes, schema version checking, and file locking.

**Files:**
- Create: `internal/state/store.go`
- Create: `internal/state/store_test.go`

- [ ] **Step 1: Write failing tests for Store.Load**

```go
// internal/state/store_test.go
package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

func TestStoreLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	st, warnings, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if st.Version != nd.SchemaVersion {
		t.Errorf("version: got %d, want %d", st.Version, nd.SchemaVersion)
	}
	if len(st.Deployments) != 0 {
		t.Errorf("deployments: got %d, want 0", len(st.Deployments))
	}
}

func TestStoreLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	data := `version: 1
deployments:
  - source_id: src
    asset_type: skills
    asset_name: review
    source_path: /src/skills/review
    link_path: /home/.claude/skills/review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
`
	os.WriteFile(path, []byte(data), 0o644)

	store := state.NewStore(path)
	st, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(st.Deployments) != 1 {
		t.Fatalf("deployments: got %d, want 1", len(st.Deployments))
	}
	if st.Deployments[0].AssetName != "review" {
		t.Errorf("asset_name: got %q", st.Deployments[0].AssetName)
	}
}

func TestStoreLoadCorruptYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	os.WriteFile(path, []byte("{{{{not yaml at all"), 0o644)

	store := state.NewStore(path)
	st, warnings, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Version != nd.SchemaVersion {
		t.Errorf("version: got %d", st.Version)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "corrupted") {
		t.Errorf("warning should mention corruption: %s", warnings[0])
	}

	// Original file should be renamed to .corrupt.<timestamp>
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".corrupt.") {
			found = true
		}
	}
	if !found {
		t.Error("corrupt file should be renamed with .corrupt. suffix")
	}
}

func TestStoreLoadNewerVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	os.WriteFile(path, []byte("version: 999\ndeployments: []\n"), 0o644)

	store := state.NewStore(path)
	_, _, err := store.Load()
	if err == nil {
		t.Fatal("expected error for newer version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error should mention version: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/state/ -run TestStore -v`
Expected: FAIL — `NewStore` undefined

- [ ] **Step 3: Write failing tests for Store.Save and round-trip**

Add to `internal/state/store_test.go`:

```go
func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	original := &state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{
				SourceID:   "src",
				AssetType:  nd.AssetSkill,
				AssetName:  "review",
				SourcePath: "/src/skills/review",
				LinkPath:   "/home/.claude/skills/review",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
			},
		},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if len(loaded.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(loaded.Deployments))
	}
	if loaded.Deployments[0].AssetName != "review" {
		t.Errorf("asset_name: got %q", loaded.Deployments[0].AssetName)
	}
}
```

- [ ] **Step 4: Write failing tests for WithLock**

Add to `internal/state/store_test.go`:

```go
func TestStoreWithLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	called := false
	err := store.WithLock(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock: %v", err)
	}
	if !called {
		t.Error("fn should have been called")
	}
}

func TestStoreWithLockPropagatesError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	sentinel := errors.New("boom")
	err := store.WithLock(func() error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
```

- [ ] **Step 5: Implement Store**

```go
// internal/state/store.go
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/nd"
)

// Store manages the deployment state file on disk.
type Store struct {
	path     string
	lockPath string
}

// NewStore creates a Store targeting the given deployments.yaml path.
func NewStore(path string) *Store {
	return &Store{
		path:     path,
		lockPath: path + ".lock",
	}
}

// Load reads and parses deployments.yaml.
// Missing file returns empty state. Corrupt YAML renames the file and returns empty state with a warning.
// Newer schema version refuses to load (NFR-014).
func (s *Store) Load() (*DeploymentState, []string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DeploymentState{Version: nd.SchemaVersion}, nil, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", s.path, err)
	}

	var st DeploymentState
	if err := yaml.Unmarshal(data, &st); err != nil {
		return s.handleCorrupt(err)
	}

	// Schema version check (NFR-014)
	if st.Version > nd.SchemaVersion {
		return nil, nil, fmt.Errorf(
			"deployments.yaml has schema version %d, but this version of nd only supports version %d; upgrade nd to read this file",
			st.Version, nd.SchemaVersion,
		)
	}
	if st.Version < nd.SchemaVersion {
		// Future: migrate here. For now, version is 1 so no migration needed.
		st.Version = nd.SchemaVersion
	}

	return &st, nil, nil
}

// handleCorrupt renames a corrupt state file and returns empty state with warning.
func (s *Store) handleCorrupt(parseErr error) (*DeploymentState, []string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05")
	corruptPath := fmt.Sprintf("%s.corrupt.%s", s.path, ts)
	os.Rename(s.path, corruptPath)

	warning := fmt.Sprintf(
		"Warning: deployments.yaml was corrupted and has been renamed to %s. Run nd sync to rebuild deployment state from the filesystem.",
		filepath.Base(corruptPath),
	)
	return &DeploymentState{Version: nd.SchemaVersion}, []string{warning}, nil
}

// Save atomically writes the deployment state to disk using nd.AtomicWrite (NFR-010).
func (s *Store) Save(st *DeploymentState) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := yaml.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return nd.AtomicWrite(s.path, data)
}

// WithLock acquires the file lock, runs fn, then releases. Times out after 5s.
func (s *Store) WithLock(fn func() error) error {
	lock := NewFileLock(s.lockPath)
	if err := lock.Acquire(5 * time.Second); err != nil {
		return err
	}
	defer lock.Release()
	return fn()
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/state/ -run "TestStore|TestFileLock" -v`
Expected: PASS (all store and lock tests)

- [ ] **Step 7: Check coverage**

Run: `go test ./internal/state/ -coverprofile=cover.out && go tool cover -func=cover.out | grep -E "store|lock"`
Expected: >85% for store.go and lock.go

- [ ] **Step 8: Commit**

```bash
git add internal/state/store.go internal/state/store_test.go
git commit -m "feat(state): add Store with Load/Save/WithLock for deployments.yaml (NFR-010, NFR-011, NFR-014)"
```

---

## Task 3: Deploy Engine — types and constructor

Create the deploy package with the StateStore interface, Engine struct, request/result types, and constructor.

**Files:**
- Create: `internal/deploy/deploy.go`
- Create: `internal/deploy/deploy_test.go`

- [ ] **Step 1: Write failing test for Engine construction**

```go
// internal/deploy/deploy_test.go
package deploy_test

import (
	"testing"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// mockStore implements deploy.StateStore for testing.
type mockStore struct {
	state    *state.DeploymentState
	saved    *state.DeploymentState
	warnings []string
	loadErr  error
	saveErr  error
	lockErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		state: &state.DeploymentState{Version: nd.SchemaVersion},
	}
}

func (m *mockStore) Load() (*state.DeploymentState, []string, error) {
	if m.loadErr != nil {
		return nil, nil, m.loadErr
	}
	// Return a copy to detect mutations
	cp := *m.state
	cp.Deployments = make([]state.Deployment, len(m.state.Deployments))
	copy(cp.Deployments, m.state.Deployments)
	return &cp, m.warnings, nil
}

func (m *mockStore) Save(st *state.DeploymentState) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = st
	m.state = st
	return nil
}

func (m *mockStore) WithLock(fn func() error) error {
	if m.lockErr != nil {
		return m.lockErr
	}
	return fn()
}

func testAgent() *agent.Agent {
	return &agent.Agent{
		Name:       "claude-code",
		GlobalDir:  "/home/user/.claude",
		ProjectDir: ".claude",
		Detected:   true,
	}
}

func TestNewEngine(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	engine := deploy.New(store, ag, "/tmp/backups")
	if engine == nil {
		t.Fatal("New returned nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/deploy/ -run TestNewEngine -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement types and constructor**

```go
// internal/deploy/deploy.go
package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// StateStore abstracts state persistence for testing.
type StateStore interface {
	Load() (*state.DeploymentState, []string, error)
	Save(st *state.DeploymentState) error
	WithLock(fn func() error) error
}

// Engine orchestrates symlink deployment, removal, health checks, and repair.
type Engine struct {
	store     StateStore
	agent     *agent.Agent
	backupDir string

	// Injected for testing (default to os.*)
	symlink  func(oldname, newname string) error
	readlink func(name string) (string, error)
	lstat    func(name string) (os.FileInfo, error)
	stat     func(name string) (os.FileInfo, error)
	remove   func(name string) error
	mkdirAll func(path string, perm os.FileMode) error
	rename   func(oldpath, newpath string) error
	now      func() time.Time
}

// New creates an Engine with default OS functions.
func New(store StateStore, ag *agent.Agent, backupDir string) *Engine {
	return &Engine{
		store:     store,
		agent:     ag,
		backupDir: backupDir,
		symlink:   os.Symlink,
		readlink:  os.Readlink,
		lstat:     os.Lstat,
		stat:      os.Stat,
		remove:    os.Remove,
		mkdirAll:  os.MkdirAll,
		rename:    os.Rename,
		now:       time.Now,
	}
}

// DeployRequest describes a single asset deployment.
type DeployRequest struct {
	Asset       asset.Asset
	Scope       nd.Scope
	ProjectRoot string
	Origin      nd.DeployOrigin
}

// DeployResult describes the outcome of a single deployment.
type DeployResult struct {
	Deployment state.Deployment
	Warnings   []string
	BackedUp   string
}

// DeployError describes a failed deployment within a bulk operation.
type DeployError struct {
	AssetName  string
	AssetType  nd.AssetType
	SourcePath string
	Err        error
}

func (e *DeployError) Error() string {
	return fmt.Sprintf("deploy %s %q from %s: %v", e.AssetType, e.AssetName, e.SourcePath, e.Err)
}

// BulkDeployResult holds outcomes of a bulk deploy operation.
type BulkDeployResult struct {
	Succeeded []DeployResult
	Failed    []DeployError
}

// RemoveRequest describes a single asset removal.
type RemoveRequest struct {
	Identity    asset.Identity
	Scope       nd.Scope
	ProjectRoot string
}

// RemoveError describes a failed removal within a bulk operation.
type RemoveError struct {
	Identity asset.Identity
	Err      error
}

func (e *RemoveError) Error() string {
	return fmt.Sprintf("remove %s %q from %s: %v", e.Identity.Type, e.Identity.Name, e.Identity.SourceID, e.Err)
}

// BulkRemoveResult holds outcomes of a bulk remove operation.
type BulkRemoveResult struct {
	Succeeded []RemoveRequest
	Failed    []RemoveError
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/deploy/ -run TestNewEngine -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/deploy/deploy.go internal/deploy/deploy_test.go
git commit -m "feat(deploy): add Engine types and constructor"
```

---

## Task 4: Engine.Deploy — single asset deployment

Implement the core Deploy method that creates a symlink and updates state. This is the most complex method, handling conflict detection, context file backup, and directory creation.

**Files:**
- Modify: `internal/deploy/deploy.go`
- Modify: `internal/deploy/deploy_test.go`

- [ ] **Step 1: Write failing test — deploy a simple skill**

Add to `internal/deploy/deploy_test.go`:

```go
func TestDeploySimpleSkill(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	engine := deploy.New(store, ag, t.TempDir())

	var createdSymlinks []symCall
	engine.SetSymlink(func(oldname, newname string) error {
		createdSymlinks = append(createdSymlinks, symCall{oldname, newname})
		return nil
	})
	engine.SetLstat(func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist // nothing at target
	})
	engine.SetMkdirAll(func(path string, perm os.FileMode) error {
		return nil
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
			SourcePath: "/sources/skills/review",
			IsDir:      true,
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(createdSymlinks) != 1 {
		t.Fatalf("expected 1 symlink, got %d", len(createdSymlinks))
	}
	if createdSymlinks[0].newname != "/home/user/.claude/skills/review" {
		t.Errorf("link path: got %q", createdSymlinks[0].newname)
	}
	if result.Deployment.AssetName != "review" {
		t.Errorf("deployment asset_name: got %q", result.Deployment.AssetName)
	}
	if store.saved == nil || len(store.saved.Deployments) != 1 {
		t.Error("state should have 1 deployment after deploy")
	}
}

type symCall struct {
	oldname, newname string
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/deploy/ -run TestDeploySimpleSkill -v`
Expected: FAIL — `SetSymlink` undefined, `Deploy` undefined

- [ ] **Step 3: Add setter methods for test injection**

Add to `internal/deploy/deploy.go`:

```go
// SetSymlink replaces the symlink function (for testing).
func (e *Engine) SetSymlink(fn func(oldname, newname string) error) { e.symlink = fn }

// SetReadlink replaces the readlink function (for testing).
func (e *Engine) SetReadlink(fn func(name string) (string, error)) { e.readlink = fn }

// SetLstat replaces the lstat function (for testing).
func (e *Engine) SetLstat(fn func(name string) (os.FileInfo, error)) { e.lstat = fn }

// SetStat replaces the stat function (for testing).
func (e *Engine) SetStat(fn func(name string) (os.FileInfo, error)) { e.stat = fn }

// SetRemove replaces the remove function (for testing).
func (e *Engine) SetRemove(fn func(name string) error) { e.remove = fn }

// SetMkdirAll replaces the mkdirAll function (for testing).
func (e *Engine) SetMkdirAll(fn func(path string, perm os.FileMode) error) { e.mkdirAll = fn }

// SetRename replaces the rename function (for testing).
func (e *Engine) SetRename(fn func(oldpath, newpath string) error) { e.rename = fn }

// SetNow replaces the time function (for testing).
func (e *Engine) SetNow(fn func() time.Time) { e.now = fn }
```

- [ ] **Step 4: Implement Deploy method**

Add to `internal/deploy/deploy.go`:

```go
// Deploy deploys a single asset by creating a symlink (FR-009, FR-011).
func (e *Engine) Deploy(req DeployRequest) (*DeployResult, error) {
	if !req.Asset.Type.IsDeployable() {
		return nil, fmt.Errorf("asset type %q is not deployable via symlink; use nd export", req.Asset.Type)
	}

	// Extract contextFile for context assets
	contextFile := ""
	if req.Asset.Type == nd.AssetContext {
		if req.Asset.ContextFile == nil {
			return nil, fmt.Errorf("context asset %q missing ContextFile info", req.Asset.Name)
		}
		contextFile = req.Asset.ContextFile.FileName
	}

	linkPath, err := e.agent.DeployPath(req.Asset.Type, req.Asset.Name, req.Scope, req.ProjectRoot, contextFile)
	if err != nil {
		return nil, fmt.Errorf("compute deploy path: %w", err)
	}

	var result DeployResult
	var deployErr error

	lockErr := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		// Conflict check
		backed, warnings, err := e.handleConflict(linkPath, req, st)
		if err != nil {
			return err
		}
		result.Warnings = append(result.Warnings, warnings...)
		result.BackedUp = backed

		// Create parent directories
		parentDir := filepath.Dir(linkPath)
		if err := e.mkdirAll(parentDir, 0o755); err != nil {
			return fmt.Errorf("permission denied: cannot write to %s: %w", parentDir, err)
		}

		// Create symlink
		if err := e.symlink(req.Asset.SourcePath, linkPath); err != nil {
			return fmt.Errorf("create symlink at %s: %w", linkPath, err)
		}

		// Build deployment entry
		dep := state.Deployment{
			SourceID:    req.Asset.SourceID,
			AssetType:   req.Asset.Type,
			AssetName:   req.Asset.Name,
			SourcePath:  req.Asset.SourcePath,
			LinkPath:    linkPath,
			Scope:       req.Scope,
			ProjectPath: req.ProjectRoot,
			Origin:      req.Origin,
			DeployedAt:  e.now(),
		}
		st.Deployments = append(st.Deployments, dep)
		result.Deployment = dep

		// Settings registration warning
		if req.Asset.Type.RequiresSettingsRegistration() {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Asset %q requires manual registration in settings.json or settings.local.json", req.Asset.Name))
		}

		return e.store.Save(st)
	})

	if lockErr != nil {
		return nil, lockErr
	}
	if deployErr != nil {
		return nil, deployErr
	}
	return &result, nil
}

// handleConflict checks for existing files/symlinks at linkPath and handles them.
// Returns (backupPath, warnings, error).
func (e *Engine) handleConflict(linkPath string, req DeployRequest, st *state.DeploymentState) (string, []string, error) {
	info, err := e.lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil // No conflict
		}
		return "", nil, fmt.Errorf("check target path %s: %w", linkPath, err)
	}

	var warnings []string

	// Something exists. Classify it.
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := e.readlink(linkPath)
		if err != nil {
			return "", nil, fmt.Errorf("readlink %s: %w", linkPath, err)
		}

		// Check if it's an nd-managed symlink
		for i, d := range st.Deployments {
			if d.LinkPath == linkPath {
				if d.SourcePath == req.Asset.SourcePath {
					// Same asset re-deployed: update timestamp
					st.Deployments[i].DeployedAt = e.now()
					st.Deployments[i].Origin = req.Origin
					return "", nil, nil
				}
				// Different nd-managed asset: remove old
				e.remove(linkPath)
				st.Deployments = append(st.Deployments[:i], st.Deployments[i+1:]...)
				return "", nil, nil
			}
		}

		// Foreign symlink (not in state)
		if req.Asset.Type == nd.AssetContext {
			backed, w := e.backupAndWarn(linkPath, nd.FileKindForeignSymlink, target)
			return backed, w, nil
		}
		return "", nil, &nd.ConflictError{
			TargetPath:   linkPath,
			ExistingKind: nd.FileKindForeignSymlink,
			AssetName:    req.Asset.Name,
		}
	}

	// Plain file
	if req.Asset.Type == nd.AssetContext {
		backed, w := e.backupAndWarn(linkPath, nd.FileKindPlainFile, "")
		return backed, w, nil
	}
	return "", nil, &nd.ConflictError{
		TargetPath:   linkPath,
		ExistingKind: nd.FileKindPlainFile,
		AssetName:    req.Asset.Name,
	}
}

// backupAndWarn backs up an existing file and returns the backup path + warnings.
func (e *Engine) backupAndWarn(linkPath string, kind nd.OriginalFileKind, target string) (string, []string) {
	backed, err := e.backupExistingFile(linkPath)
	var warnings []string
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to back up %s: %v", linkPath, err))
		return "", warnings
	}

	msg := fmt.Sprintf("Backed up existing %s at %s to %s", kind, linkPath, backed)
	if kind == nd.FileKindPlainFile {
		msg = fmt.Sprintf("Backed up existing manually created file at %s to %s", linkPath, backed)
	}
	warnings = append(warnings, msg)
	return backed, warnings
}

// backupExistingFile moves the file at path to backupDir with a timestamp suffix.
// Retains only the last 5 backups per base filename.
func (e *Engine) backupExistingFile(path string) (string, error) {
	if err := e.mkdirAll(e.backupDir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Base(path)
	ts := e.now().Format("2006-01-02T15-04-05")
	backupName := fmt.Sprintf("%s.%s.bak", base, ts)
	backupPath := filepath.Join(e.backupDir, backupName)

	if err := e.rename(path, backupPath); err != nil {
		return "", err
	}

	// Prune: keep only last 5 backups for this base filename
	e.pruneBackups(base, 5)
	return backupPath, nil
}

// pruneBackups removes old backups exceeding maxKeep for files matching the given base name.
func (e *Engine) pruneBackups(baseName string, maxKeep int) {
	entries, err := os.ReadDir(e.backupDir)
	if err != nil {
		return
	}

	prefix := baseName + "."
	var matching []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			matching = append(matching, entry.Name())
		}
	}

	// ReadDir returns entries sorted by name. Timestamps in filenames sort chronologically.
	if len(matching) > maxKeep {
		for _, name := range matching[:len(matching)-maxKeep] {
			e.remove(filepath.Join(e.backupDir, name))
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/deploy/ -run TestDeploySimpleSkill -v`
Expected: PASS

- [ ] **Step 6: Write additional deploy tests**

Add to `internal/deploy/deploy_test.go`:

```go
func TestDeployNotDeployable(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity: asset.Identity{SourceID: "src", Type: nd.AssetPlugin, Name: "p"},
		},
	}
	_, err := engine.Deploy(req)
	if err == nil {
		t.Fatal("expected error for plugin deploy")
	}
}

func TestDeployContextFile(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	var created []symCall
	engine.SetSymlink(func(o, n string) error { created = append(created, symCall{o, n}); return nil })
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:    asset.Identity{SourceID: "src", Type: nd.AssetContext, Name: "go-rules"},
			SourcePath:  "/sources/context/go-rules/CLAUDE.md",
			ContextFile: &asset.ContextInfo{FolderName: "go-rules", FileName: "CLAUDE.md"},
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 symlink, got %d", len(created))
	}
	// Context files deploy to agent global dir directly, not a subdirectory
	if created[0].newname != "/home/user/.claude/CLAUDE.md" {
		t.Errorf("link path: got %q, want /home/user/.claude/CLAUDE.md", created[0].newname)
	}
	_ = result
}

func TestDeployConflictNonContext(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	// Existing plain file at target
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: 0o644}, nil // plain file
	})
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
			SourcePath: "/sources/skills/review",
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	_, err := engine.Deploy(req)
	var conflictErr *nd.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected ConflictError, got %T: %v", err, err)
	}
}

func TestDeployHookWarnsSettings(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetHook, Name: "lint"},
			SourcePath: "/sources/hooks/lint",
			IsDir:      true,
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "settings.json") {
			found = true
		}
	}
	if !found {
		t.Error("expected settings registration warning for hook deploy")
	}
}

// fakeFileInfo implements os.FileInfo for testing conflict detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string      { return "fake" }
func (f fakeFileInfo) Size() int64       { return 0 }
func (f fakeFileInfo) Mode() os.FileMode { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool       { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any          { return nil }
```

- [ ] **Step 7: Run all deploy tests**

Run: `go test ./internal/deploy/ -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/deploy/deploy.go internal/deploy/deploy_test.go
git commit -m "feat(deploy): implement Engine.Deploy with conflict detection and context backup (FR-009, FR-011, FR-016b)"
```

---

## Task 5: Engine.DeployBulk and Engine.Remove/RemoveBulk

Implement bulk deploy (fail-open) and remove operations.

**Files:**
- Modify: `internal/deploy/deploy.go`
- Modify: `internal/deploy/deploy_test.go`

- [ ] **Step 1: Write failing tests for DeployBulk and Remove**

Add to `internal/deploy/deploy_test.go`:

```go
func TestDeployBulkPartialFailure(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	callCount := 0
	engine.SetSymlink(func(o, n string) error {
		callCount++
		if callCount == 2 {
			return fmt.Errorf("disk full")
		}
		return nil
	})
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, SourcePath: "/s/a"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, SourcePath: "/s/b"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "c"}, SourcePath: "/s/c"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	result, err := engine.DeployBulk(reqs)
	if err != nil {
		t.Fatalf("DeployBulk: %v", err)
	}
	if len(result.Succeeded) != 2 {
		t.Errorf("succeeded: got %d, want 2", len(result.Succeeded))
	}
	if len(result.Failed) != 1 {
		t.Errorf("failed: got %d, want 1", len(result.Failed))
	}
}

func TestRemoveAsset(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/s/skills/review", LinkPath: "/home/user/.claude/skills/review",
			Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	removed := false
	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetRemove(func(name string) error { removed = true; return nil })

	err := engine.Remove(deploy.RemoveRequest{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "review"},
		Scope:    nd.ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if !removed {
		t.Error("symlink should have been removed")
	}
	if store.saved == nil || len(store.saved.Deployments) != 0 {
		t.Error("state should have 0 deployments after remove")
	}
}

func TestRemoveAlreadyGone(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			LinkPath: "/home/user/.claude/skills/review", Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetRemove(func(string) error { return os.ErrNotExist })

	err := engine.Remove(deploy.RemoveRequest{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "review"},
		Scope:    nd.ScopeGlobal,
	})
	if err != nil {
		t.Fatalf("Remove should tolerate missing symlink: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/deploy/ -run "TestDeployBulk|TestRemove" -v`
Expected: FAIL — methods not implemented

- [ ] **Step 3: Implement DeployBulk, Remove, RemoveBulk**

Add to `internal/deploy/deploy.go`:

```go
// DeployBulk deploys multiple assets with fail-open behavior (FR-010).
// Acquires lock once, loads state once, saves once at the end.
func (e *Engine) DeployBulk(reqs []DeployRequest) (*BulkDeployResult, error) {
	var result BulkDeployResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, req := range reqs {
			dr, err := e.deployOne(req, st)
			if err != nil {
				result.Failed = append(result.Failed, DeployError{
					AssetName:  req.Asset.Name,
					AssetType:  req.Asset.Type,
					SourcePath: req.Asset.SourcePath,
					Err:        err,
				})
				continue
			}
			result.Succeeded = append(result.Succeeded, *dr)
		}

		return e.store.Save(st)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// deployOne performs a single deploy within an existing lock+state context.
func (e *Engine) deployOne(req DeployRequest, st *state.DeploymentState) (*DeployResult, error) {
	if !req.Asset.Type.IsDeployable() {
		return nil, fmt.Errorf("asset type %q is not deployable via symlink; use nd export", req.Asset.Type)
	}

	contextFile := ""
	if req.Asset.Type == nd.AssetContext {
		if req.Asset.ContextFile == nil {
			return nil, fmt.Errorf("context asset %q missing ContextFile info", req.Asset.Name)
		}
		contextFile = req.Asset.ContextFile.FileName
	}

	linkPath, err := e.agent.DeployPath(req.Asset.Type, req.Asset.Name, req.Scope, req.ProjectRoot, contextFile)
	if err != nil {
		return nil, fmt.Errorf("compute deploy path: %w", err)
	}

	var result DeployResult

	backed, warnings, err := e.handleConflict(linkPath, req, st)
	if err != nil {
		return nil, err
	}
	result.Warnings = append(result.Warnings, warnings...)
	result.BackedUp = backed

	parentDir := filepath.Dir(linkPath)
	if err := e.mkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("permission denied: cannot write to %s: %w", parentDir, err)
	}

	if err := e.symlink(req.Asset.SourcePath, linkPath); err != nil {
		return nil, fmt.Errorf("create symlink at %s: %w", linkPath, err)
	}

	dep := state.Deployment{
		SourceID:    req.Asset.SourceID,
		AssetType:   req.Asset.Type,
		AssetName:   req.Asset.Name,
		SourcePath:  req.Asset.SourcePath,
		LinkPath:    linkPath,
		Scope:       req.Scope,
		ProjectPath: req.ProjectRoot,
		Origin:      req.Origin,
		DeployedAt:  e.now(),
	}
	st.Deployments = append(st.Deployments, dep)
	result.Deployment = dep

	if req.Asset.Type.RequiresSettingsRegistration() {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Asset %q requires manual registration in settings.json or settings.local.json", req.Asset.Name))
	}

	return &result, nil
}

// Remove removes a single deployed asset (FR-012).
func (e *Engine) Remove(req RemoveRequest) error {
	return e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		idx := -1
		for i, d := range st.Deployments {
			if d.SourceID == req.Identity.SourceID &&
				d.AssetType == req.Identity.Type &&
				d.AssetName == req.Identity.Name &&
				d.Scope == req.Scope {
				if req.Scope == nd.ScopeProject && d.ProjectPath != req.ProjectRoot {
					continue
				}
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("deployment not found: %s/%s from %s", req.Identity.Type, req.Identity.Name, req.Identity.SourceID)
		}

		err = e.remove(st.Deployments[idx].LinkPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove symlink %s: %w", st.Deployments[idx].LinkPath, err)
		}

		st.Deployments = append(st.Deployments[:idx], st.Deployments[idx+1:]...)
		return e.store.Save(st)
	})
}

// RemoveBulk removes multiple deployed assets with fail-open behavior (FR-012).
func (e *Engine) RemoveBulk(reqs []RemoveRequest) (*BulkRemoveResult, error) {
	var result BulkRemoveResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, req := range reqs {
			if err := e.removeOne(req, st); err != nil {
				result.Failed = append(result.Failed, RemoveError{
					Identity: req.Identity,
					Err:      err,
				})
				continue
			}
			result.Succeeded = append(result.Succeeded, req)
		}

		return e.store.Save(st)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// removeOne removes a single deployment within an existing lock+state context.
func (e *Engine) removeOne(req RemoveRequest, st *state.DeploymentState) error {
	idx := -1
	for i, d := range st.Deployments {
		if d.SourceID == req.Identity.SourceID &&
			d.AssetType == req.Identity.Type &&
			d.AssetName == req.Identity.Name &&
			d.Scope == req.Scope {
			if req.Scope == nd.ScopeProject && d.ProjectPath != req.ProjectRoot {
				continue
			}
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("deployment not found: %s/%s from %s", req.Identity.Type, req.Identity.Name, req.Identity.SourceID)
	}

	err := e.remove(st.Deployments[idx].LinkPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove symlink %s: %w", st.Deployments[idx].LinkPath, err)
	}

	st.Deployments = append(st.Deployments[:idx], st.Deployments[idx+1:]...)
	return nil
}
```

Now refactor `Deploy` to use `deployOne` internally:

```go
// Deploy deploys a single asset by creating a symlink (FR-009, FR-011).
func (e *Engine) Deploy(req DeployRequest) (*DeployResult, error) {
	var result *DeployResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		r, err := e.deployOne(req, st)
		if err != nil {
			return err
		}
		result = r

		return e.store.Save(st)
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}
```

- [ ] **Step 4: Run all deploy/remove tests**

Run: `go test ./internal/deploy/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/deploy/deploy.go internal/deploy/deploy_test.go
git commit -m "feat(deploy): implement DeployBulk, Remove, RemoveBulk with fail-open bulk (FR-010, FR-012)"
```

---

## Task 6: Health checks — Check, Sync, Status

Implement the health checking, repair, and status operations using existing `state.HealthStatus` and `state.HealthCheck` types.

**Files:**
- Create: `internal/deploy/health.go`
- Create: `internal/deploy/health_test.go`

- [ ] **Step 1: Write failing tests for Check**

```go
// internal/deploy/health_test.go
package deploy_test

import (
	"os"
	"testing"

	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

func TestCheckHealthy(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/src/skills/review", nil
	})
	engine.SetStat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{}, nil // target exists
	})

	checks, err := engine.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(checks) != 0 {
		t.Errorf("expected 0 issues for healthy deployment, got %d", len(checks))
	}
}

func TestCheckBrokenLink(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/src/skills/review", nil
	})
	engine.SetStat(func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist // target gone
	})

	checks, err := engine.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(checks))
	}
	if checks[0].Status != state.HealthBroken {
		t.Errorf("status: got %v, want HealthBroken", checks[0].Status)
	}
}

func TestCheckMissingLink(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return nil, os.ErrNotExist // symlink deleted externally
	})

	checks, err := engine.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(checks) != 1 || checks[0].Status != state.HealthMissing {
		t.Errorf("expected HealthMissing, got %v", checks)
	}
}

func TestCheckDriftedLink(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/wrong/path", nil // points somewhere else
	})

	checks, err := engine.Check()
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(checks) != 1 || checks[0].Status != state.HealthDrifted {
		t.Errorf("expected HealthDrifted, got %v", checks)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/deploy/ -run TestCheck -v`
Expected: FAIL — `Check` undefined

- [ ] **Step 3: Implement Check, Sync, Status**

```go
// internal/deploy/health.go
package deploy

import (
	"fmt"
	"os"

	"github.com/larah/nd/internal/state"
)

// StatusEntry pairs a deployment with its health status.
type StatusEntry struct {
	Deployment state.Deployment
	Health     state.HealthStatus
	Detail     string
}

// SyncResult holds the outcomes of a sync/repair operation.
type SyncResult struct {
	Repaired []state.Deployment
	Removed  []state.Deployment
	Warnings []string
}

// Check detects deployment health issues (FR-013).
// Returns only unhealthy entries. Empty slice means all healthy.
func (e *Engine) Check() ([]state.HealthCheck, error) {
	var issues []state.HealthCheck

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range st.Deployments {
			if hc := e.checkOne(dep); hc.Status != state.HealthOK {
				issues = append(issues, hc)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return issues, nil
}

// checkOne evaluates the health of a single deployment.
func (e *Engine) checkOne(dep state.Deployment) state.HealthCheck {
	hc := state.HealthCheck{Deployment: dep, Status: state.HealthOK}

	// Step 1: Does the symlink node exist?
	_, err := e.lstat(dep.LinkPath)
	if err != nil {
		hc.Status = state.HealthMissing
		hc.Detail = fmt.Sprintf("symlink %s does not exist", dep.LinkPath)
		return hc
	}

	// Step 2: Does it point to the expected target?
	target, err := e.readlink(dep.LinkPath)
	if err != nil {
		hc.Status = state.HealthBroken
		hc.Detail = fmt.Sprintf("cannot read symlink %s: %v", dep.LinkPath, err)
		return hc
	}
	if target != dep.SourcePath {
		hc.Status = state.HealthDrifted
		hc.Detail = fmt.Sprintf("symlink points to %s, expected %s", target, dep.SourcePath)
		return hc
	}

	// Step 3: Does the target actually exist? (follows symlinks)
	if _, err := e.stat(dep.LinkPath); err != nil {
		hc.Status = state.HealthBroken
		hc.Detail = fmt.Sprintf("target %s does not exist", dep.SourcePath)
		return hc
	}

	return hc
}

// Sync repairs detected deployment issues (FR-014).
func (e *Engine) Sync() (*SyncResult, error) {
	var result SyncResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		var keep []state.Deployment
		for _, dep := range st.Deployments {
			hc := e.checkOne(dep)
			switch hc.Status {
			case state.HealthOK:
				keep = append(keep, dep)

			case state.HealthBroken, state.HealthOrphaned:
				// Source gone: remove symlink and state entry
				e.remove(dep.LinkPath)
				result.Removed = append(result.Removed, dep)

			case state.HealthMissing:
				// Symlink deleted externally: re-create if source exists
				if _, err := e.stat(dep.SourcePath); err == nil {
					e.mkdirAll(fmt.Sprintf("%s", filepath.Dir(dep.LinkPath)), 0o755)
					if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
						result.Repaired = append(result.Repaired, dep)
						keep = append(keep, dep)
					} else {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Failed to re-create %s: %v", dep.LinkPath, err))
						result.Removed = append(result.Removed, dep)
					}
				} else {
					// Source also gone
					result.Removed = append(result.Removed, dep)
				}

			case state.HealthDrifted:
				// Re-create symlink to correct target
				e.remove(dep.LinkPath)
				if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
					result.Repaired = append(result.Repaired, dep)
					keep = append(keep, dep)
				} else {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Failed to repair %s: %v", dep.LinkPath, err))
					keep = append(keep, dep) // keep entry, it might be fixable later
				}
			}
		}

		st.Deployments = keep
		return e.store.Save(st)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Status returns all deployments with their health status (FR-015).
// Returns a flat list; grouping by type is the caller's responsibility.
func (e *Engine) Status() ([]StatusEntry, error) {
	var entries []StatusEntry

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range st.Deployments {
			hc := e.checkOne(dep)
			entries = append(entries, StatusEntry{
				Deployment: dep,
				Health:     hc.Status,
				Detail:     hc.Detail,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return entries, nil
}
```

Note: Add `"path/filepath"` to the imports in health.go.

- [ ] **Step 4: Run check tests to verify they pass**

Run: `go test ./internal/deploy/ -run TestCheck -v`
Expected: PASS

- [ ] **Step 5: Write tests for Sync and Status**

Add to `internal/deploy/health_test.go`:

```go
func TestSyncRepairsMissing(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	created := false
	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetStat(func(name string) (os.FileInfo, error) {
		if name == "/src/skills/review" {
			return fakeFileInfo{}, nil // source exists
		}
		return nil, os.ErrNotExist
	})
	engine.SetSymlink(func(o, n string) error { created = true; return nil })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	engine.SetRemove(func(string) error { return nil })

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !created {
		t.Error("symlink should have been re-created")
	}
	if len(result.Repaired) != 1 {
		t.Errorf("repaired: got %d, want 1", len(result.Repaired))
	}
}

func TestSyncRemovesBroken(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	removed := false
	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) { return "/src/skills/review", nil })
	engine.SetStat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetRemove(func(string) error { removed = true; return nil })

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !removed {
		t.Error("broken symlink should have been removed")
	}
	if len(result.Removed) != 1 {
		t.Errorf("removed: got %d, want 1", len(result.Removed))
	}
	if len(store.saved.Deployments) != 0 {
		t.Error("state should have 0 deployments after removing broken")
	}
}

func TestStatus(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
		{SourceID: "s", AssetType: nd.AssetAgent, AssetName: "helper",
			SourcePath: "/src/agents/helper.md", LinkPath: "/home/.claude/agents/helper.md",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(name string) (string, error) {
		// Return matching source paths
		for _, d := range store.state.Deployments {
			if d.LinkPath == name {
				return d.SourcePath, nil
			}
		}
		return "", os.ErrNotExist
	})
	engine.SetStat(func(string) (os.FileInfo, error) { return fakeFileInfo{}, nil })

	entries, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(entries))
	}
	for _, e := range entries {
		if e.Health != state.HealthOK {
			t.Errorf("expected HealthOK for %s, got %v", e.Deployment.AssetName, e.Health)
		}
	}
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/deploy/ -v`
Expected: PASS

- [ ] **Step 7: Check coverage**

Run: `go test ./internal/deploy/ -coverprofile=cover.out && go tool cover -func=cover.out`
Expected: >85% for deploy.go and health.go

- [ ] **Step 8: Run full project tests**

Run: `go test ./...`
Expected: PASS (no regressions)

- [ ] **Step 9: Commit**

```bash
git add internal/deploy/health.go internal/deploy/health_test.go
git commit -m "feat(deploy): implement Check, Sync, Status for health monitoring (FR-013, FR-014, FR-015)"
```

---

## Task 7: Final verification and cleanup

Run all tests, check coverage, fix any issues.

**Files:**
- All files from tasks 1-6

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: PASS (all packages)

- [ ] **Step 2: Check coverage for new code**

Run: `go test ./internal/state/ ./internal/deploy/ -coverprofile=cover.out && go tool cover -func=cover.out`
Expected: >85% for store.go, lock.go, deploy.go, health.go

- [ ] **Step 3: Run linter**

Run: `golangci-lint run ./internal/state/ ./internal/deploy/`
Expected: No issues (or only pre-existing issues)

- [ ] **Step 4: Run vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 5: Commit any cleanup**

If any fixes were needed:

```bash
git add -A
git commit -m "chore(deploy): test coverage and lint fixes"
```
