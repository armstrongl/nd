package deploy_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
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

// symCall records a symlink creation for test assertions.
type symCall struct {
	oldname, newname string
}

// fakeFileInfo implements os.FileInfo for testing conflict detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string        { return "fake" }
func (f fakeFileInfo) Size() int64         { return 0 }
func (f fakeFileInfo) Mode() os.FileMode   { return f.mode }
func (f fakeFileInfo) ModTime() time.Time  { return time.Time{} }
func (f fakeFileInfo) IsDir() bool         { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any            { return nil }

func TestNewEngine(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	engine := deploy.New(store, ag, "/tmp/backups")
	if engine == nil {
		t.Fatal("New returned nil")
	}
}

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

func TestRemoveBulk(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a",
			LinkPath: "/home/user/.claude/skills/a", Scope: nd.ScopeGlobal},
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "b",
			LinkPath: "/home/user/.claude/skills/b", Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetRemove(func(string) error { return nil })

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "b"}, Scope: nd.ScopeGlobal},
	}

	result, err := engine.RemoveBulk(reqs)
	if err != nil {
		t.Fatalf("RemoveBulk: %v", err)
	}
	if len(result.Succeeded) != 2 {
		t.Errorf("succeeded: got %d, want 2", len(result.Succeeded))
	}
	if len(result.Failed) != 0 {
		t.Errorf("failed: got %d, want 0", len(result.Failed))
	}
	if len(store.saved.Deployments) != 0 {
		t.Error("state should have 0 deployments after bulk remove")
	}
}

func TestDeployContextWithExistingPlainFile(t *testing.T) {
	store := newMockStore()
	backupDir := t.TempDir()
	engine := deploy.New(store, testAgent(), backupDir)

	renamedFrom := ""
	renamedTo := ""
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: 0o644}, nil // plain file exists
	})
	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	engine.SetRename(func(old, new string) error {
		renamedFrom = old
		renamedTo = new
		return nil
	})
	engine.SetNow(func() time.Time {
		return time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:    asset.Identity{SourceID: "src", Type: nd.AssetContext, Name: "rules"},
			SourcePath:  "/sources/context/rules/CLAUDE.md",
			ContextFile: &asset.ContextInfo{FolderName: "rules", FileName: "CLAUDE.md"},
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if result.BackedUp == "" {
		t.Error("expected backup path to be set")
	}
	if renamedFrom == "" {
		t.Error("expected rename to be called for backup")
	}
	_ = renamedTo
	// Check warning mentions "manually created file"
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "manually created file") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about manually created file, got %v", result.Warnings)
	}
}

func TestDeployForeignSymlinkContextBacksUp(t *testing.T) {
	store := newMockStore()
	backupDir := t.TempDir()
	engine := deploy.New(store, testAgent(), backupDir)

	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/some/other/target", nil
	})
	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	engine.SetRename(func(old, new string) error { return nil })
	engine.SetNow(func() time.Time {
		return time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:    asset.Identity{SourceID: "src", Type: nd.AssetContext, Name: "rules"},
			SourcePath:  "/sources/context/rules/CLAUDE.md",
			ContextFile: &asset.ContextInfo{FolderName: "rules", FileName: "CLAUDE.md"},
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if result.BackedUp == "" {
		t.Error("expected backup path to be set for foreign symlink on context")
	}
	// Should not return a ConflictError for context assets
}

func TestDeployForeignSymlinkNonContext(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/some/other/target", nil
	})

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
		t.Fatalf("expected ConflictError for foreign symlink, got %T: %v", err, err)
	}
	if conflictErr.ExistingKind != nd.FileKindForeignSymlink {
		t.Errorf("expected FileKindForeignSymlink, got %v", conflictErr.ExistingKind)
	}
}

func TestDeployManagedSymlinkSameAsset(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/sources/skills/review",
			LinkPath:   "/home/user/.claude/skills/review",
			Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/sources/skills/review", nil
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
			SourcePath: "/sources/skills/review",
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	// Should reuse existing deployment, not create new one
	if store.saved == nil || len(store.saved.Deployments) != 1 {
		t.Errorf("expected exactly 1 deployment (re-deploy), got %d", len(store.saved.Deployments))
	}
	_ = result
}

func TestDeployManagedSymlinkDifferentAsset(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "old", AssetType: nd.AssetSkill, AssetName: "old-review",
			SourcePath: "/old/skills/old-review",
			LinkPath:   "/home/user/.claude/skills/review",
			Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	removedPath := ""
	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/old/skills/old-review", nil
	})
	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	engine.SetRemove(func(name string) error { removedPath = name; return nil })

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
			SourcePath: "/sources/skills/review",
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	_, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if removedPath != "/home/user/.claude/skills/review" {
		t.Errorf("expected old symlink removed, got %q", removedPath)
	}
	if store.saved == nil || len(store.saved.Deployments) != 1 {
		t.Errorf("expected 1 deployment (replaced), got %d", len(store.saved.Deployments))
	}
	if store.saved.Deployments[0].SourcePath != "/sources/skills/review" {
		t.Errorf("expected new source path, got %q", store.saved.Deployments[0].SourcePath)
	}
}

func TestDeployErrorString(t *testing.T) {
	e := deploy.DeployError{
		AssetName: "review", AssetType: nd.AssetSkill,
		SourcePath: "/s/review", Err: fmt.Errorf("disk full"),
	}
	s := e.Error()
	if !strings.Contains(s, "review") || !strings.Contains(s, "disk full") {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestRemoveErrorString(t *testing.T) {
	e := deploy.RemoveError{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "review"},
		Err:      fmt.Errorf("permission denied"),
	}
	s := e.Error()
	if !strings.Contains(s, "review") || !strings.Contains(s, "permission denied") {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestBackupFailureHandling(t *testing.T) {
	store := newMockStore()
	backupDir := t.TempDir()
	engine := deploy.New(store, testAgent(), backupDir)

	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: 0o644}, nil // plain file exists
	})
	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetMkdirAll(func(path string, perm os.FileMode) error {
		// Let backup dir creation fail for backup, succeed for deploy parent dir
		if path == backupDir {
			return fmt.Errorf("permission denied")
		}
		return nil
	})
	engine.SetNow(func() time.Time {
		return time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:    asset.Identity{SourceID: "src", Type: nd.AssetContext, Name: "rules"},
			SourcePath:  "/sources/context/rules/CLAUDE.md",
			ContextFile: &asset.ContextInfo{FolderName: "rules", FileName: "CLAUDE.md"},
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	result, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy should succeed even if backup fails: %v", err)
	}
	if result.BackedUp != "" {
		t.Error("expected empty backup path when backup fails")
	}
	// Should have a warning about failed backup
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Failed to back up") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about backup failure, got %v", result.Warnings)
	}
}

func TestRemoveNotFound(t *testing.T) {
	store := newMockStore()
	engine := deploy.New(store, testAgent(), t.TempDir())

	err := engine.Remove(deploy.RemoveRequest{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "nonexistent"},
		Scope:    nd.ScopeGlobal,
	})
	if err == nil {
		t.Fatal("expected error for removing nonexistent deployment")
	}
	if !strings.Contains(err.Error(), "deployment not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveLoadError(t *testing.T) {
	store := newMockStore()
	store.loadErr = fmt.Errorf("corrupt state")
	engine := deploy.New(store, testAgent(), t.TempDir())

	err := engine.Remove(deploy.RemoveRequest{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "review"},
		Scope:    nd.ScopeGlobal,
	})
	if err == nil {
		t.Fatal("expected error when load fails")
	}
}

func TestRemoveBulkPartialFailure(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "a",
			LinkPath: "/home/user/.claude/skills/a", Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetRemove(func(string) error { return nil })

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "a"}, Scope: nd.ScopeGlobal},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "nonexistent"}, Scope: nd.ScopeGlobal},
	}

	result, err := engine.RemoveBulk(reqs)
	if err != nil {
		t.Fatalf("RemoveBulk: %v", err)
	}
	if len(result.Succeeded) != 1 {
		t.Errorf("succeeded: got %d, want 1", len(result.Succeeded))
	}
	if len(result.Failed) != 1 {
		t.Errorf("failed: got %d, want 1", len(result.Failed))
	}
}

func TestDeployLoadError(t *testing.T) {
	store := newMockStore()
	store.loadErr = fmt.Errorf("corrupt state")
	engine := deploy.New(store, testAgent(), t.TempDir())

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:   asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
			SourcePath: "/sources/skills/review",
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	_, err := engine.Deploy(req)
	if err == nil {
		t.Fatal("expected error when load fails")
	}
}

func TestRemoveSymlinkError(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/s/skills/review", LinkPath: "/home/user/.claude/skills/review",
			Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetRemove(func(name string) error { return fmt.Errorf("permission denied") })

	err := engine.Remove(deploy.RemoveRequest{
		Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "review"},
		Scope:    nd.ScopeGlobal,
	})
	if err == nil {
		t.Fatal("expected error when remove fails")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPruneBackups(t *testing.T) {
	// Create a real backup directory with 7 backup files, then deploy a context
	// asset that triggers backup+prune. Only 5 should remain after pruning.
	backupDir := t.TempDir()

	// Pre-create 6 existing backup files (alphabetically sorted by timestamp)
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("CLAUDE.md.2026-03-15T10-%02d-00.bak", i)
		path := backupDir + "/" + name
		if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := newMockStore()
	engine := deploy.New(store, testAgent(), backupDir)

	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: 0o644}, nil // plain file exists
	})
	engine.SetSymlink(func(o, n string) error { return nil })
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	// Use real rename: since the source file doesn't exist, rename will fail.
	// Instead, mock rename to create the backup file.
	engine.SetRename(func(old, new string) error {
		return os.WriteFile(new, []byte("backed up"), 0o644)
	})
	engine.SetNow(func() time.Time {
		return time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	})

	req := deploy.DeployRequest{
		Asset: asset.Asset{
			Identity:    asset.Identity{SourceID: "src", Type: nd.AssetContext, Name: "rules"},
			SourcePath:  "/sources/context/rules/CLAUDE.md",
			ContextFile: &asset.ContextInfo{FolderName: "rules", FileName: "CLAUDE.md"},
		},
		Scope:  nd.ScopeGlobal,
		Origin: nd.OriginManual,
	}

	_, err := engine.Deploy(req)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	// Count remaining backup files
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if strings.Contains(e.Name(), "CLAUDE.md.") && strings.HasSuffix(e.Name(), ".bak") {
			count++
		}
	}
	if count != 5 {
		t.Errorf("expected 5 backups after pruning, got %d", count)
	}
}

// --- SnapshotSaver tests ---

type mockSnapshotSaver struct {
	called      bool
	deployments []state.Deployment
	err         error
}

func (m *mockSnapshotSaver) AutoSave(deployments []state.Deployment) error {
	m.called = true
	m.deployments = deployments
	return m.err
}

func TestDeployBulkTriggersAutoSnapshot(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	eng := deploy.New(store, ag, t.TempDir())

	saver := &mockSnapshotSaver{}
	eng.SetSnapshotSaver(saver)

	// Seed an existing deployment so the snapshot captures it
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "existing",
			SourcePath: "/a", LinkPath: "/b", Scope: nd.ScopeGlobal,
			Origin: nd.OriginManual},
	}

	eng.SetSymlink(func(_, _ string) error { return nil })
	eng.SetLstat(func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	eng.SetMkdirAll(func(_ string, _ os.FileMode) error { return nil })

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "new"},
			SourcePath: "/src/skills/new"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}
	_, err := eng.DeployBulk(reqs)
	if err != nil {
		t.Fatalf("DeployBulk: %v", err)
	}

	if !saver.called {
		t.Error("SnapshotSaver.AutoSave was not called")
	}
	if len(saver.deployments) != 1 {
		t.Errorf("expected 1 existing deployment captured, got %d", len(saver.deployments))
	}
}

func TestRemoveBulkTriggersAutoSnapshot(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	eng := deploy.New(store, ag, t.TempDir())

	saver := &mockSnapshotSaver{}
	eng.SetSnapshotSaver(saver)

	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "target",
			SourcePath: "/a", LinkPath: "/b", Scope: nd.ScopeGlobal,
			Origin: nd.OriginManual},
	}

	eng.SetRemove(func(_ string) error { return nil })

	reqs := []deploy.RemoveRequest{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "target"},
			Scope: nd.ScopeGlobal},
	}
	_, err := eng.RemoveBulk(reqs)
	if err != nil {
		t.Fatalf("RemoveBulk: %v", err)
	}

	if !saver.called {
		t.Error("SnapshotSaver.AutoSave was not called")
	}
}

func TestBulkWorksWithoutSnapshotSaver(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	eng := deploy.New(store, ag, t.TempDir())
	// No saver set — should still work

	eng.SetSymlink(func(_, _ string) error { return nil })
	eng.SetLstat(func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	eng.SetMkdirAll(func(_ string, _ os.FileMode) error { return nil })

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "x"},
			SourcePath: "/src/skills/x"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}
	_, err := eng.DeployBulk(reqs)
	if err != nil {
		t.Fatalf("DeployBulk without saver: %v", err)
	}
}

func TestAutoSnapshotFailureDoesNotBlockBulk(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	eng := deploy.New(store, ag, t.TempDir())

	saver := &mockSnapshotSaver{err: fmt.Errorf("disk full")}
	eng.SetSnapshotSaver(saver)

	eng.SetSymlink(func(_, _ string) error { return nil })
	eng.SetLstat(func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	eng.SetMkdirAll(func(_ string, _ os.FileMode) error { return nil })

	reqs := []deploy.DeployRequest{
		{Asset: asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "x"},
			SourcePath: "/src/skills/x"}, Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}
	result, err := eng.DeployBulk(reqs)
	if err != nil {
		t.Fatalf("DeployBulk should proceed despite snapshot failure: %v", err)
	}
	if len(result.Succeeded) != 1 {
		t.Errorf("expected 1 succeeded, got %d", len(result.Succeeded))
	}
}
