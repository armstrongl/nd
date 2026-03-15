package profile

import (
	"fmt"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// StateStore abstracts state persistence (same contract as deploy.StateStore).
type StateStore interface {
	Load() (*state.DeploymentState, []string, error)
	Save(st *state.DeploymentState) error
	WithLock(fn func() error) error
}

// DeployEngine abstracts the deploy engine for profile switch and restore operations.
type DeployEngine interface {
	DeployBulk(reqs []deploy.DeployRequest) (*deploy.BulkDeployResult, error)
	RemoveBulk(reqs []deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
}

// SwitchResult describes the outcome of a profile switch.
type SwitchResult struct {
	FromProfile   string
	ToProfile     string
	Diff          SwitchDiff
	Removed       *deploy.BulkRemoveResult
	Deployed      *deploy.BulkDeployResult
	MissingAssets []ProfileAsset
	SkippedPinned []ProfileAsset // assets not removed because they have non-profile origin
	Conflicts     []ProfileAsset // target assets that conflict with pinned/manual deployments
}

// RestoreResult describes the outcome of a snapshot restoration.
type RestoreResult struct {
	SnapshotName  string
	Removed       *deploy.BulkRemoveResult
	Deployed      *deploy.BulkDeployResult
	MissingAssets []SnapshotEntry
}

// Manager orchestrates profile switching and snapshot restoration.
type Manager struct {
	store      *Store
	stateStore StateStore
}

// NewManager creates a Manager with the given profile store and state store.
func NewManager(store *Store, stateStore StateStore) *Manager {
	return &Manager{
		store:      store,
		stateStore: stateStore,
	}
}

// ActiveProfile returns the currently active profile name from deployment state.
// Returns "" if no profile is active.
func (m *Manager) ActiveProfile() (string, error) {
	var name string
	err := m.stateStore.WithLock(func() error {
		st, _, err := m.stateStore.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}
		name = st.ActiveProfile
		return nil
	})
	return name, err
}

// SetActiveProfile updates the active profile in deployment state.
// Pass "" to clear the active profile.
func (m *Manager) SetActiveProfile(name string) error {
	if name != "" {
		if err := ValidateName(name); err != nil {
			return fmt.Errorf("set active profile: %w", err)
		}
	}

	return m.stateStore.WithLock(func() error {
		st, _, err := m.stateStore.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}
		st.ActiveProfile = name
		return m.stateStore.Save(st)
	})
}

// DeleteProfile deletes a profile, refusing if it is the active profile.
func (m *Manager) DeleteProfile(name string) error {
	active, err := m.ActiveProfile()
	if err != nil {
		return fmt.Errorf("check active profile: %w", err)
	}
	if active == name {
		return fmt.Errorf("cannot delete active profile %q: switch to another profile first", name)
	}
	return m.store.DeleteProfile(name)
}

// Switch changes from the current profile to the target profile.
// Loads both profiles, computes diff, removes old assets, deploys new assets,
// and updates the active profile in state. projectRoot is required for
// project-scoped assets (pass "" for global-only switches).
func (m *Manager) Switch(currentName, targetName string, engine DeployEngine, index *asset.Index, projectRoot string) (*SwitchResult, error) {
	current, err := m.store.GetProfile(currentName)
	if err != nil {
		return nil, fmt.Errorf("load current profile: %w", err)
	}
	target, err := m.store.GetProfile(targetName)
	if err != nil {
		return nil, fmt.Errorf("load target profile: %w", err)
	}

	diff := ComputeSwitchDiff(current, target)

	result := &SwitchResult{
		FromProfile: currentName,
		ToProfile:   targetName,
		Diff:        diff,
	}

	// Load deployment state to check actual origins (FR-023).
	deployState, _, err := m.stateStore.Load()
	if err != nil {
		return nil, fmt.Errorf("load deployment state: %w", err)
	}

	// Build an origin lookup: (source_id, asset_type, asset_name) -> DeployOrigin
	type deployKey struct {
		SourceID  string
		AssetType nd.AssetType
		AssetName string
	}
	originMap := make(map[deployKey]nd.DeployOrigin, len(deployState.Deployments))
	for _, d := range deployState.Deployments {
		originMap[deployKey{d.SourceID, d.AssetType, d.AssetName}] = d.Origin
	}

	// Build remove requests from diff.Remove, filtering by origin (FR-023).
	if len(diff.Remove) > 0 {
		expectedOrigin := nd.OriginProfile(currentName)
		var removeReqs []deploy.RemoveRequest
		for _, pa := range diff.Remove {
			key := deployKey{pa.SourceID, pa.AssetType, pa.AssetName}
			actualOrigin, exists := originMap[key]
			if exists && actualOrigin != expectedOrigin {
				result.SkippedPinned = append(result.SkippedPinned, pa)
				continue
			}
			removeReqs = append(removeReqs, deploy.RemoveRequest{
				Identity:    pa.Identity(),
				Scope:       pa.Scope,
				ProjectRoot: projectRoot,
			})
		}
		if len(removeReqs) > 0 {
			removed, err := engine.RemoveBulk(removeReqs)
			if err != nil {
				return nil, fmt.Errorf("remove old profile assets: %w", err)
			}
			result.Removed = removed
		}
	}

	// Build deploy requests from diff.Deploy, looking up assets in index.
	if len(diff.Deploy) > 0 {
		var deployReqs []deploy.DeployRequest
		for _, pa := range diff.Deploy {
			// Check for conflicts with existing pinned/manual deployments
			key := deployKey{pa.SourceID, pa.AssetType, pa.AssetName}
			if actualOrigin, exists := originMap[key]; exists {
				if actualOrigin == nd.OriginPinned || actualOrigin == nd.OriginManual {
					result.Conflicts = append(result.Conflicts, pa)
				}
			}

			a := index.Lookup(pa.Identity())
			if a == nil {
				result.MissingAssets = append(result.MissingAssets, pa)
				continue
			}
			deployReqs = append(deployReqs, deploy.DeployRequest{
				Asset:       *a,
				Scope:       pa.Scope,
				ProjectRoot: projectRoot,
				Origin:      nd.OriginProfile(targetName),
			})
		}
		if len(deployReqs) > 0 {
			deployed, err := engine.DeployBulk(deployReqs)
			if err != nil {
				return nil, fmt.Errorf("deploy target profile assets: %w", err)
			}
			result.Deployed = deployed
		}
	}

	// Update active profile
	if err := m.SetActiveProfile(targetName); err != nil {
		return nil, fmt.Errorf("set active profile: %w", err)
	}

	return result, nil
}

// DeployProfile deploys all assets in a profile without requiring a current active profile.
// This is the "first deploy" path (FR-024). For switching between profiles, use Switch.
func (m *Manager) DeployProfile(name string, engine DeployEngine, index *asset.Index, projectRoot string) (*SwitchResult, error) {
	target, err := m.store.GetProfile(name)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}

	result := &SwitchResult{
		ToProfile: name,
	}

	var deployReqs []deploy.DeployRequest
	for _, pa := range target.Assets {
		a := index.Lookup(pa.Identity())
		if a == nil {
			result.MissingAssets = append(result.MissingAssets, pa)
			continue
		}
		deployReqs = append(deployReqs, deploy.DeployRequest{
			Asset:       *a,
			Scope:       pa.Scope,
			ProjectRoot: projectRoot,
			Origin:      nd.OriginProfile(name),
		})
	}

	if len(deployReqs) > 0 {
		deployed, err := engine.DeployBulk(deployReqs)
		if err != nil {
			return nil, fmt.Errorf("deploy profile assets: %w", err)
		}
		result.Deployed = deployed
	}

	if err := m.SetActiveProfile(name); err != nil {
		return nil, fmt.Errorf("set active profile: %w", err)
	}

	return result, nil
}

// Restore reverts deployments to match a saved snapshot.
// Removes all current deployments, then re-deploys snapshot entries
// by looking up assets in the source index (re-deploy from sources strategy).
func (m *Manager) Restore(snapshotName string, engine DeployEngine, index *asset.Index) (*RestoreResult, error) {
	// Try user snapshot first, then auto
	snap, err := m.store.GetSnapshot(snapshotName, false)
	if err != nil {
		snap, err = m.store.GetSnapshot(snapshotName, true)
		if err != nil {
			return nil, fmt.Errorf("snapshot %q not found", snapshotName)
		}
	}

	result := &RestoreResult{SnapshotName: snapshotName}

	// Read current deployments to know what to remove.
	st, _, err := m.stateStore.Load()
	if err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	// Remove all current deployments
	if len(st.Deployments) > 0 {
		removeReqs := make([]deploy.RemoveRequest, len(st.Deployments))
		for i, d := range st.Deployments {
			removeReqs[i] = deploy.RemoveRequest{
				Identity:    d.Identity(),
				Scope:       d.Scope,
				ProjectRoot: d.ProjectPath,
			}
		}
		removed, err := engine.RemoveBulk(removeReqs)
		if err != nil {
			return nil, fmt.Errorf("remove current deployments: %w", err)
		}
		result.Removed = removed
	}

	// Re-deploy snapshot entries from sources
	var deployReqs []deploy.DeployRequest
	for _, entry := range snap.Deployments {
		id := asset.Identity{
			SourceID: entry.SourceID,
			Type:     entry.AssetType,
			Name:     entry.AssetName,
		}
		a := index.Lookup(id)
		if a == nil {
			result.MissingAssets = append(result.MissingAssets, entry)
			continue
		}
		deployReqs = append(deployReqs, deploy.DeployRequest{
			Asset:       *a,
			Scope:       entry.Scope,
			ProjectRoot: entry.ProjectPath,
			Origin:      entry.Origin,
		})
	}

	if len(deployReqs) > 0 {
		deployed, err := engine.DeployBulk(deployReqs)
		if err != nil {
			return nil, fmt.Errorf("deploy snapshot assets: %w", err)
		}
		result.Deployed = deployed
	}

	// Clear active profile on restore (snapshot may not correspond to any profile)
	if err := m.SetActiveProfile(""); err != nil {
		return nil, fmt.Errorf("clear active profile: %w", err)
	}

	return result, nil
}
