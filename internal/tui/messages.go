package tui

import (
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/armstrongl/nd/internal/state"
)

// Service result messages — returned by async tea.Cmd functions.

// StatusResultMsg carries the result of loading deployment status.
type StatusResultMsg struct {
	Entries []deploy.StatusEntry
	Err     error
}

// HealthCheckMsg carries the result of running health checks.
type HealthCheckMsg struct {
	Checks []state.HealthCheck
	Err    error
}

// ScanCompleteMsg carries the result of scanning sources.
type ScanCompleteMsg struct {
	Summary *sourcemanager.ScanSummary
	Err     error
}

// SyncResultMsg carries the result of syncing sources.
type SyncResultMsg struct {
	Result *deploy.SyncResult
	Err    error
}

// DeployResultMsg carries the result of deploying an asset.
type DeployResultMsg struct {
	Result          *deploy.DeployResult
	RequiresManual  bool   // true for hooks/output-styles needing settings.json registration
	ManualInstructs string // settings.json snippet to display
	Err             error
}

// RemoveResultMsg carries the result of removing an asset.
type RemoveResultMsg struct {
	Err error
}

// ProfileSwitchMsg carries the result of switching profiles.
type ProfileSwitchMsg struct {
	Result *profile.SwitchResult
	Err    error
}

// SnapshotSaveMsg carries the result of saving a snapshot.
type SnapshotSaveMsg struct {
	Name string
	Err  error
}

// SnapshotRestoreMsg carries the result of restoring a snapshot.
type SnapshotRestoreMsg struct {
	Result *profile.RestoreResult
	Err    error
}

// UI state messages.

// ToastMsg triggers a toast notification.
type ToastMsg struct {
	Message string
	Level   ToastLevel
}

// ToastDismissMsg signals the active toast should be dismissed.
type ToastDismissMsg struct{}

// ErrorMsg carries an error from any operation.
type ErrorMsg struct {
	Err error
}
