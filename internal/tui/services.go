package tui

import (
	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/source"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// Deployer abstracts the deploy engine for TUI operations.
type Deployer interface {
	Deploy(deploy.DeployRequest) (*deploy.DeployResult, error)
	DeployBulk([]deploy.DeployRequest) (*deploy.BulkDeployResult, error)
	Remove(deploy.RemoveRequest) error
	RemoveBulk([]deploy.RemoveRequest) (*deploy.BulkRemoveResult, error)
	Status() ([]deploy.StatusEntry, error)
	Check() ([]state.HealthCheck, error)
	Sync() (*deploy.SyncResult, error)
	SetOrigin(asset.Identity, nd.Scope, string, nd.DeployOrigin) error
}

// ProfileSwitcher abstracts profile operations for the TUI.
// Uses a simplified interface that captures engine/index at construction time.
// An adapter wraps *profile.Manager with pre-bound deps.
type ProfileSwitcher interface {
	ActiveProfile() (string, error)
	Switch(current, target string) (*profile.SwitchResult, error)
	Restore(name string) (*profile.RestoreResult, error)
	SaveSnapshot(name string) error
	ListProfiles() ([]profile.ProfileSummary, error)
	ListSnapshots() ([]profile.SnapshotSummary, error)
}

// SourceScanner abstracts source management for the TUI.
type SourceScanner interface {
	Sources() []source.Source
	Scan() (*sourcemanager.ScanSummary, error)
	SyncSource(sourceID string) error
}

// AgentDetector abstracts agent detection for the TUI.
type AgentDetector interface {
	Detect() agent.DetectionResult
	Default() (*agent.Agent, error)
	All() []agent.Agent
}
