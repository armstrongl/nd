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
