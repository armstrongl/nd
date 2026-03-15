package profile_test

import (
	"testing"
	"time"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/profile"
	"github.com/larah/nd/internal/state"
)

// mockStateStore implements profile.StateStore for testing.
type mockStateStore struct {
	st      *state.DeploymentState
	saveErr error
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{
		st: &state.DeploymentState{Version: nd.SchemaVersion},
	}
}

func (m *mockStateStore) Load() (*state.DeploymentState, []string, error) {
	cp := *m.st
	cp.Deployments = make([]state.Deployment, len(m.st.Deployments))
	copy(cp.Deployments, m.st.Deployments)
	return &cp, nil, nil
}

func (m *mockStateStore) Save(st *state.DeploymentState) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.st = st
	return nil
}

func (m *mockStateStore) WithLock(fn func() error) error {
	return fn()
}

// mockDeployEngine implements profile.DeployEngine for testing.
type mockDeployEngine struct {
	deployBulkCalled bool
	removeBulkCalled bool
	deployReqs       []deploy.DeployRequest
	removeReqs       []deploy.RemoveRequest
	deployResult     *deploy.BulkDeployResult
	removeResult     *deploy.BulkRemoveResult
	deployErr        error
	removeErr        error
}

func (m *mockDeployEngine) DeployBulk(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
	m.deployBulkCalled = true
	m.deployReqs = reqs
	if m.deployErr != nil {
		return nil, m.deployErr
	}
	if m.deployResult != nil {
		return m.deployResult, nil
	}
	return &deploy.BulkDeployResult{}, nil
}

func (m *mockDeployEngine) RemoveBulk(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
	m.removeBulkCalled = true
	m.removeReqs = reqs
	if m.removeErr != nil {
		return nil, m.removeErr
	}
	if m.removeResult != nil {
		return m.removeResult, nil
	}
	return &deploy.BulkRemoveResult{}, nil
}

// Silence unused import warnings for tests that come later.
var (
	_ = asset.NewIndex
	_ = time.Now
	_ = deploy.DeployRequest{}
)

func TestManagerActiveProfileEmpty(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	name, err := mgr.ActiveProfile()
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty, got %q", name)
	}
}

func TestManagerSetActiveProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	if err := mgr.SetActiveProfile("go-backend"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}

	name, _ := mgr.ActiveProfile()
	if name != "go-backend" {
		t.Errorf("active profile: got %q", name)
	}
}

func TestManagerClearActiveProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	mgr.SetActiveProfile("something")
	if err := mgr.SetActiveProfile(""); err != nil {
		t.Fatalf("SetActiveProfile empty: %v", err)
	}

	name, _ := mgr.ActiveProfile()
	if name != "" {
		t.Errorf("should be empty, got %q", name)
	}
}

func TestManagerDeleteProfileRefusesActive(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "active-one", CreatedAt: now, UpdatedAt: now})
	mgr.SetActiveProfile("active-one")

	if err := mgr.DeleteProfile("active-one"); err == nil {
		t.Error("should refuse to delete the active profile")
	}

	// Should still exist
	if _, err := store.GetProfile("active-one"); err != nil {
		t.Errorf("profile should not have been deleted: %v", err)
	}
}

func TestManagerDeleteProfileAllowsInactive(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "deletable", CreatedAt: now, UpdatedAt: now})

	if err := mgr.DeleteProfile("deletable"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	if _, err := store.GetProfile("deletable"); err == nil {
		t.Error("profile should be deleted")
	}
}

// --- Switch tests ---

func TestManagerSwitch(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "current", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "old-skill", Scope: nd.ScopeGlobal},
		},
	})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "target", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "new-skill", Scope: nd.ScopeGlobal},
		},
	})

	// Seed deployment state so old-skill has the correct profile origin
	ss.st.Deployments = []state.Deployment{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "old-skill",
			SourcePath: "/src/skills/old-skill", LinkPath: "/link/old-skill",
			Scope: nd.ScopeGlobal, Origin: nd.OriginProfile("current")},
	}

	// Build a source index with the target asset
	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "new-skill"},
			SourcePath: "/src/skills/new-skill", IsDir: true},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.Switch("current", "target", eng, idx, "")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if result.FromProfile != "current" || result.ToProfile != "target" {
		t.Errorf("profiles: %q -> %q", result.FromProfile, result.ToProfile)
	}
	if !eng.removeBulkCalled {
		t.Error("RemoveBulk was not called")
	}
	if !eng.deployBulkCalled {
		t.Error("DeployBulk was not called")
	}
	if len(eng.removeReqs) != 1 {
		t.Fatalf("expected 1 remove request, got %d", len(eng.removeReqs))
	}
	if eng.removeReqs[0].Identity.Name != "old-skill" {
		t.Errorf("remove request name: got %q, want %q", eng.removeReqs[0].Identity.Name, "old-skill")
	}
	if len(eng.deployReqs) != 1 {
		t.Fatalf("expected 1 deploy request, got %d", len(eng.deployReqs))
	}
	if eng.deployReqs[0].Origin != nd.OriginProfile("target") {
		t.Errorf("deploy request origin: got %q, want %q", eng.deployReqs[0].Origin, nd.OriginProfile("target"))
	}
	if len(result.SkippedPinned) != 0 {
		t.Errorf("expected 0 skipped pinned, got %d", len(result.SkippedPinned))
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(result.Conflicts))
	}

	// Active profile should be updated
	active, _ := mgr.ActiveProfile()
	if active != "target" {
		t.Errorf("active profile should be 'target', got %q", active)
	}
}

func TestManagerSwitchProfileNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	_, err := mgr.Switch("nonexistent", "also-missing", eng, idx, "")
	if err == nil {
		t.Error("should error on nonexistent profiles")
	}
}

func TestManagerSwitchMissingAssetInIndex(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "from", CreatedAt: now, UpdatedAt: now,
	})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "to", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "missing", Scope: nd.ScopeGlobal},
		},
	})

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil) // Empty index — asset won't be found

	result, err := mgr.Switch("from", "to", eng, idx, "")
	if err != nil {
		t.Fatalf("Switch should succeed with warnings: %v", err)
	}
	if len(result.MissingAssets) != 1 {
		t.Errorf("expected 1 missing asset, got %d", len(result.MissingAssets))
	}
}

func TestManagerSwitchIdenticalProfiles(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	assets := []profile.ProfileAsset{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "shared", Scope: nd.ScopeGlobal},
	}
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "alpha", CreatedAt: now, UpdatedAt: now, Assets: assets,
	})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "beta", CreatedAt: now, UpdatedAt: now, Assets: assets,
	})

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	result, err := mgr.Switch("alpha", "beta", eng, idx, "")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if eng.removeBulkCalled {
		t.Error("RemoveBulk should not be called for identical profiles")
	}
	if eng.deployBulkCalled {
		t.Error("DeployBulk should not be called for identical profiles")
	}

	active, _ := mgr.ActiveProfile()
	if active != "beta" {
		t.Errorf("active profile should be 'beta', got %q", active)
	}
	_ = result
}

func TestManagerSwitchSkipsPinnedAssets(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "current", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "pinned-skill", Scope: nd.ScopeGlobal},
		},
	})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "target", CreatedAt: now, UpdatedAt: now,
	})

	// Deployment state has asset X with origin "pinned" (not profile:current)
	ss.st.Deployments = []state.Deployment{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "pinned-skill",
			SourcePath: "/src/pinned", LinkPath: "/link/pinned",
			Scope: nd.ScopeGlobal, Origin: nd.OriginPinned},
	}

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	result, err := mgr.Switch("current", "target", eng, idx, "")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if eng.removeBulkCalled {
		t.Error("RemoveBulk should not be called when all removals are skipped")
	}
	if len(result.SkippedPinned) != 1 {
		t.Fatalf("expected 1 skipped pinned asset, got %d", len(result.SkippedPinned))
	}
	if result.SkippedPinned[0].AssetName != "pinned-skill" {
		t.Errorf("skipped asset name: got %q, want %q", result.SkippedPinned[0].AssetName, "pinned-skill")
	}
}

func TestManagerSwitchDetectsConflicts(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "current", CreatedAt: now, UpdatedAt: now,
	})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "target", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "manual-skill", Scope: nd.ScopeGlobal},
		},
	})

	// Deployment state already has asset Y with origin "manual"
	ss.st.Deployments = []state.Deployment{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "manual-skill",
			SourcePath: "/src/manual", LinkPath: "/link/manual",
			Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
	}

	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "manual-skill"},
			SourcePath: "/src/manual", IsDir: true},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.Switch("current", "target", eng, idx, "")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if len(result.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(result.Conflicts))
	}
	if result.Conflicts[0].AssetName != "manual-skill" {
		t.Errorf("conflict asset name: got %q, want %q", result.Conflicts[0].AssetName, "manual-skill")
	}

	if !eng.deployBulkCalled {
		t.Error("DeployBulk should still be called even with conflicts")
	}
	if len(eng.deployReqs) != 1 {
		t.Fatalf("expected 1 deploy request, got %d", len(eng.deployReqs))
	}
	if eng.deployReqs[0].Origin != nd.OriginProfile("target") {
		t.Errorf("deploy request origin: got %q, want %q", eng.deployReqs[0].Origin, nd.OriginProfile("target"))
	}
}

// --- DeployProfile tests ---

func TestManagerDeployProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "first-profile", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "my-skill", Scope: nd.ScopeGlobal},
		},
	})

	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "my-skill"},
			SourcePath: "/src/skills/my-skill", IsDir: true},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.DeployProfile("first-profile", eng, idx, "")
	if err != nil {
		t.Fatalf("DeployProfile: %v", err)
	}
	if !eng.deployBulkCalled {
		t.Error("DeployBulk was not called")
	}
	if eng.removeBulkCalled {
		t.Error("RemoveBulk should not be called for first deploy")
	}
	if len(eng.deployReqs) != 1 {
		t.Errorf("expected 1 deploy request, got %d", len(eng.deployReqs))
	}
	if eng.deployReqs[0].Origin != nd.OriginProfile("first-profile") {
		t.Errorf("expected origin profile:first-profile, got %v", eng.deployReqs[0].Origin)
	}

	active, _ := mgr.ActiveProfile()
	if active != "first-profile" {
		t.Errorf("active profile should be 'first-profile', got %q", active)
	}
	_ = result
}

func TestManagerDeployProfileNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	_, err := mgr.DeployProfile("nonexistent", eng, idx, "")
	if err == nil {
		t.Error("should error on nonexistent profile")
	}
}

// --- Restore tests ---

func TestManagerRestore(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	// Seed existing deployment state so restore knows what to remove
	ss.st.Deployments = []state.Deployment{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "current-skill",
			SourcePath: "/a", LinkPath: "/b", Scope: nd.ScopeGlobal,
			Origin: nd.OriginManual},
	}

	// Create a snapshot to restore
	now := time.Now().Truncate(time.Second)
	store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "restore-me", CreatedAt: now,
		Deployments: []profile.SnapshotEntry{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "old-skill",
				SourcePath: "/src/old", LinkPath: "/link/old", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
		},
	})

	// Index has the asset from the snapshot
	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "old-skill"},
			SourcePath: "/src/old", IsDir: true},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.Restore("restore-me", eng, idx)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if result.SnapshotName != "restore-me" {
		t.Errorf("snapshot name: got %q", result.SnapshotName)
	}
	if !eng.removeBulkCalled {
		t.Error("RemoveBulk was not called")
	}
	if !eng.deployBulkCalled {
		t.Error("DeployBulk was not called")
	}

	// Active profile should be cleared after restore
	active, _ := mgr.ActiveProfile()
	if active != "" {
		t.Errorf("active profile should be cleared, got %q", active)
	}
}

func TestManagerRestoreSnapshotNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	_, err := mgr.Restore("nonexistent", eng, idx)
	if err == nil {
		t.Error("should error on nonexistent snapshot")
	}
}

func TestManagerRestoreMissingAssets(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "has-missing", CreatedAt: now,
		Deployments: []profile.SnapshotEntry{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "gone",
				SourcePath: "/src/gone", LinkPath: "/link/gone", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "found",
				SourcePath: "/src/found", LinkPath: "/link/found", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
		},
	})

	// Only "found" is in the index
	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetAgent, Name: "found"},
			SourcePath: "/src/found"},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.Restore("has-missing", eng, idx)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(result.MissingAssets) != 1 {
		t.Errorf("expected 1 missing asset, got %d", len(result.MissingAssets))
	}
	if result.MissingAssets[0].AssetName != "gone" {
		t.Errorf("missing asset name: got %q", result.MissingAssets[0].AssetName)
	}
}

func TestManagerRestoreAutoSnapshot(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := newMockStateStore()
	mgr := profile.NewManager(store, ss)

	now := time.Now().Truncate(time.Second)
	store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "auto-20260315T140000", CreatedAt: now, Auto: true,
		Deployments: []profile.SnapshotEntry{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "restored",
				SourcePath: "/src/restored", LinkPath: "/link/restored", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
		},
	})

	idx := asset.NewIndex([]asset.Asset{
		{Identity: asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "restored"},
			SourcePath: "/src/restored"},
	})

	eng := &mockDeployEngine{}

	result, err := mgr.Restore("auto-20260315T140000", eng, idx)
	if err != nil {
		t.Fatalf("Restore auto-snapshot: %v", err)
	}
	if result.SnapshotName != "auto-20260315T140000" {
		t.Errorf("snapshot name: got %q", result.SnapshotName)
	}
}
