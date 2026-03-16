package tui

import (
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/profile"
)

// profileAdapter wraps *profile.Manager with pre-bound dependencies,
// simplifying the interface for TUI consumers. The TUI should not
// manage engine, index, or projectRoot directly.
type profileAdapter struct {
	mgr         *profile.Manager
	engine      profile.DeployEngine
	indexFn     func() *asset.Index
	projectRoot string
}

// NewProfileAdapter creates a ProfileSwitcher backed by a profile.Manager.
// indexFn is called on each Switch/DeployProfile to get the latest asset index.
func NewProfileAdapter(mgr *profile.Manager, engine profile.DeployEngine, indexFn func() *asset.Index, projectRoot string) ProfileSwitcher {
	return &profileAdapter{
		mgr:         mgr,
		engine:      engine,
		indexFn:     indexFn,
		projectRoot: projectRoot,
	}
}

func (a *profileAdapter) ActiveProfile() (string, error) {
	return a.mgr.ActiveProfile()
}

func (a *profileAdapter) Switch(current, target string) (*profile.SwitchResult, error) {
	return a.mgr.Switch(current, target, a.engine, a.indexFn(), a.projectRoot)
}

func (a *profileAdapter) Restore(name string) (*profile.RestoreResult, error) {
	return a.mgr.Restore(name, a.engine, a.indexFn())
}

func (a *profileAdapter) SaveSnapshot(name string) error {
	return a.mgr.SaveSnapshot(name)
}

func (a *profileAdapter) ListProfiles() ([]profile.ProfileSummary, error) {
	return a.mgr.ListProfiles()
}

func (a *profileAdapter) ListSnapshots() ([]profile.SnapshotSummary, error) {
	return a.mgr.ListSnapshots()
}
