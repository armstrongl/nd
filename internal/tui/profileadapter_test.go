package tui_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
)

// Compile-time assertion: profileAdapter satisfies ProfileSwitcher.
// (NewProfileAdapter returns ProfileSwitcher, so this is implicitly verified.)
var _ tui.ProfileSwitcher = tui.NewProfileAdapter(nil, nil, nil, "")

// mockStateStore implements profile.StateStore for testing.
type mockStateStore struct {
	st *state.DeploymentState
}

func (m *mockStateStore) Load() (*state.DeploymentState, []string, error) {
	cp := *m.st
	cp.Deployments = make([]state.Deployment, len(m.st.Deployments))
	copy(cp.Deployments, m.st.Deployments)
	return &cp, nil, nil
}

func (m *mockStateStore) Save(st *state.DeploymentState) error {
	m.st = st
	return nil
}

func (m *mockStateStore) WithLock(fn func() error) error {
	return fn()
}

// mockDeployEngine implements profile.DeployEngine.
type mockDeployEngine struct {
	deployResult *deploy.BulkDeployResult
	removeResult *deploy.BulkRemoveResult
}

func (m *mockDeployEngine) DeployBulk(_ []deploy.DeployRequest) (*deploy.BulkDeployResult, error) {
	if m.deployResult != nil {
		return m.deployResult, nil
	}
	return &deploy.BulkDeployResult{}, nil
}

func (m *mockDeployEngine) RemoveBulk(_ []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error) {
	if m.removeResult != nil {
		return m.removeResult, nil
	}
	return &deploy.BulkRemoveResult{}, nil
}

func setupAdapter(t *testing.T) (tui.ProfileSwitcher, *profile.Store) {
	t.Helper()
	dir := t.TempDir()
	profilesDir := dir + "/profiles"
	snapshotsDir := dir + "/snapshots"

	store := profile.NewStore(profilesDir, snapshotsDir)
	ss := &mockStateStore{st: &state.DeploymentState{Version: nd.SchemaVersion}}
	mgr := profile.NewManager(store, ss)
	eng := &mockDeployEngine{}
	idx := asset.NewIndex(nil)

	adapter := tui.NewProfileAdapter(mgr, eng, func() *asset.Index { return idx }, "")
	return adapter, store
}

func TestProfileAdapterActiveProfile(t *testing.T) {
	adapter, _ := setupAdapter(t)

	name, err := adapter.ActiveProfile()
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if name != "" {
		t.Errorf("expected empty active profile, got %q", name)
	}
}

func TestProfileAdapterListProfiles(t *testing.T) {
	adapter, store := setupAdapter(t)

	_ = store.CreateProfile(profile.Profile{Name: "test-profile", Assets: []profile.ProfileAsset{}})

	summaries, err := adapter.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(summaries))
	}
	if summaries[0].Name != "test-profile" {
		t.Errorf("expected test-profile, got %q", summaries[0].Name)
	}
}

func TestProfileAdapterListSnapshots(t *testing.T) {
	adapter, store := setupAdapter(t)

	_ = store.SaveSnapshot(profile.Snapshot{
		Name:        "snap-a",
		Deployments: []profile.SnapshotEntry{},
	})

	summaries, err := adapter.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(summaries))
	}
	if summaries[0].Name != "snap-a" {
		t.Errorf("expected snap-a, got %q", summaries[0].Name)
	}
}

func TestProfileAdapterSaveSnapshot(t *testing.T) {
	adapter, store := setupAdapter(t)

	if err := adapter.SaveSnapshot("my-snap"); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	snap, err := store.GetSnapshot("my-snap", false)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snap.Name != "my-snap" {
		t.Errorf("expected my-snap, got %q", snap.Name)
	}
}
