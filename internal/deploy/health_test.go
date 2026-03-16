package deploy_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
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

func TestSyncRepairsDrifted(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	removedPath := ""
	createdOld := ""
	createdNew := ""
	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: os.ModeSymlink}, nil
	})
	engine.SetReadlink(func(string) (string, error) {
		return "/wrong/path", nil // drifted
	})
	engine.SetRemove(func(name string) error { removedPath = name; return nil })
	engine.SetSymlink(func(old, new string) error {
		createdOld = old
		createdNew = new
		return nil
	})

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Repaired) != 1 {
		t.Errorf("repaired: got %d, want 1", len(result.Repaired))
	}
	if removedPath != "/home/.claude/skills/review" {
		t.Errorf("should have removed drifted symlink, got %q", removedPath)
	}
	if createdOld != "/src/skills/review" || createdNew != "/home/.claude/skills/review" {
		t.Errorf("should have re-created correct symlink, got %q -> %q", createdOld, createdNew)
	}
	if len(store.saved.Deployments) != 1 {
		t.Error("deployment should be kept after drift repair")
	}
}

func TestSyncMissingSourceGone(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetStat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }) // source also gone
	engine.SetRemove(func(string) error { return nil })

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Removed) != 1 {
		t.Errorf("removed: got %d, want 1", len(result.Removed))
	}
	if len(store.saved.Deployments) != 0 {
		t.Error("state should be empty when both symlink and source are gone")
	}
}

func TestSyncDriftedRepairFails(t *testing.T) {
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
		return "/wrong/path", nil // drifted
	})
	engine.SetRemove(func(string) error { return nil })
	engine.SetSymlink(func(old, new string) error {
		return fmt.Errorf("disk full")
	})

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning about failed repair, got %d", len(result.Warnings))
	}
	// Deployment should be kept (might be fixable later)
	if len(store.saved.Deployments) != 1 {
		t.Error("deployment should be kept even when repair fails")
	}
}

func TestCheckLoadError(t *testing.T) {
	store := newMockStore()
	store.loadErr = fmt.Errorf("corrupt state")
	engine := deploy.New(store, testAgent(), t.TempDir())

	_, err := engine.Check()
	if err == nil {
		t.Fatal("expected error when load fails")
	}
}

func TestSyncLoadError(t *testing.T) {
	store := newMockStore()
	store.loadErr = fmt.Errorf("corrupt state")
	engine := deploy.New(store, testAgent(), t.TempDir())

	_, err := engine.Sync()
	if err == nil {
		t.Fatal("expected error when load fails")
	}
}

func TestStatusLoadError(t *testing.T) {
	store := newMockStore()
	store.loadErr = fmt.Errorf("corrupt state")
	engine := deploy.New(store, testAgent(), t.TempDir())

	_, err := engine.Status()
	if err == nil {
		t.Fatal("expected error when load fails")
	}
}

func TestSyncHealthyNoOp(t *testing.T) {
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
		return fakeFileInfo{}, nil
	})

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Repaired) != 0 {
		t.Errorf("repaired: got %d, want 0", len(result.Repaired))
	}
	if len(result.Removed) != 0 {
		t.Errorf("removed: got %d, want 0", len(result.Removed))
	}
	if len(store.saved.Deployments) != 1 {
		t.Error("deployment should remain after healthy sync")
	}
}

func TestSyncMissingRepairFails(t *testing.T) {
	store := newMockStore()
	store.state.Deployments = []state.Deployment{
		{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/src/skills/review", LinkPath: "/home/.claude/skills/review",
			Scope: nd.ScopeGlobal},
	}

	engine := deploy.New(store, testAgent(), t.TempDir())
	engine.SetLstat(func(string) (os.FileInfo, error) { return nil, os.ErrNotExist })
	engine.SetStat(func(string) (os.FileInfo, error) { return fakeFileInfo{}, nil }) // source exists
	engine.SetMkdirAll(func(string, os.FileMode) error { return nil })
	engine.SetSymlink(func(o, n string) error { return fmt.Errorf("disk full") })

	result, err := engine.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Removed) != 1 {
		t.Errorf("removed: got %d, want 1 (failed repair should remove)", len(result.Removed))
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning about failed re-creation, got %d", len(result.Warnings))
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
