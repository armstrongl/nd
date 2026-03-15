package deploy_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/asset"
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
