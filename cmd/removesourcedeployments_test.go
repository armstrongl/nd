package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// fakeRemoveStore is a deploy.StateStore for exercising removeSourceDeployments
// (cmd/source.go:341-374) directly without a real filesystem-backed store.
type fakeRemoveStore struct {
	st      *state.DeploymentState
	saved   *state.DeploymentState
	loadErr error
	saveErr error
}

func (f *fakeRemoveStore) Load() (*state.DeploymentState, []string, error) {
	if f.loadErr != nil {
		return nil, nil, f.loadErr
	}
	cp := *f.st
	cp.Deployments = make([]state.Deployment, len(f.st.Deployments))
	copy(cp.Deployments, f.st.Deployments)
	return &cp, nil, nil
}

func (f *fakeRemoveStore) Save(st *state.DeploymentState) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = st
	f.st = st
	return nil
}

func (f *fakeRemoveStore) WithLock(fn func() error) error { return fn() }

func newRemoveEngine(t *testing.T, store deploy.StateStore) *deploy.Engine {
	t.Helper()
	return deploy.New(store, &agent.Agent{Name: "claude-code"}, t.TempDir())
}

func TestRemoveSourceDeployments_StatusError(t *testing.T) {
	store := &fakeRemoveStore{st: &state.DeploymentState{}, loadErr: errors.New("corrupt state")}
	eng := newRemoveEngine(t, store)

	err := removeSourceDeployments(eng, "target")
	if err == nil {
		t.Fatal("expected error when Status() fails")
	}
}

func TestRemoveSourceDeployments_NoMatches(t *testing.T) {
	store := &fakeRemoveStore{st: &state.DeploymentState{
		Deployments: []state.Deployment{
			{SourceID: "other", AssetType: nd.AssetSkill, AssetName: "a",
				LinkPath: "/x", Scope: nd.ScopeGlobal, Agent: "claude-code"},
		},
	}}
	eng := newRemoveEngine(t, store)

	if err := removeSourceDeployments(eng, "target"); err != nil {
		t.Fatalf("expected nil when no deployments match the source, got: %v", err)
	}
}

func TestRemoveSourceDeployments_RemoveBulkError(t *testing.T) {
	store := &fakeRemoveStore{
		st: &state.DeploymentState{
			Deployments: []state.Deployment{
				{SourceID: "target", AssetType: nd.AssetSkill, AssetName: "a",
					LinkPath: "/x", Scope: nd.ScopeGlobal, Agent: "claude-code"},
			},
		},
		saveErr: errors.New("disk full"),
	}
	eng := newRemoveEngine(t, store)
	eng.SetRemove(func(string) error { return nil })

	err := removeSourceDeployments(eng, "target")
	if err == nil {
		t.Fatal("expected error when RemoveBulk fails to persist")
	}
}

func TestRemoveSourceDeployments_PartialFailure(t *testing.T) {
	store := &fakeRemoveStore{st: &state.DeploymentState{
		Deployments: []state.Deployment{
			{SourceID: "target", AssetType: nd.AssetSkill, AssetName: "a",
				LinkPath: "/x", Scope: nd.ScopeGlobal, Agent: "claude-code"},
		},
	}}
	eng := newRemoveEngine(t, store)
	// A non-NotExist remove error makes removeOne fail, populating result.Failed
	// while RemoveBulk itself still returns nil.
	eng.SetRemove(func(string) error { return errors.New("permission denied") })

	err := removeSourceDeployments(eng, "target")
	if err == nil || !strings.Contains(err.Error(), "failed to remove") {
		t.Fatalf("expected 'failed to remove' partial-failure error, got: %v", err)
	}
}
