package tui

import "fmt"

// Empty state messages with actionable hints.
// Called by screens when they have no data to display.

// NoSources returns the empty state message for when no asset sources are configured.
func NoSources() string {
	return "No asset sources configured.\n\n  Run nd source add <path> to add one, or press enter to set up now."
}

// NoAssets returns the empty state message for when no assets are found.
func NoAssets() string {
	return "No assets found.\n\n  Add a source with nd source add <path> first."
}

// NothingDeployed returns the empty state message for when nothing has been deployed.
func NothingDeployed() string {
	return "Nothing deployed yet.\n\n  Select Deploy from the menu to get started."
}

// NoProfiles returns the empty state message for when no profiles exist.
func NoProfiles() string {
	return "No profiles yet.\n\n  Create one with nd profile create <name>."
}

// NoSnapshots returns the empty state message for when no snapshots exist.
func NoSnapshots() string {
	return "No snapshots yet.\n\n  Save one with nd snapshot save <name>."
}

// AllDeployed returns the empty state message for when all assets of a given type
// are already deployed.
func AllDeployed(typeName string) string {
	return fmt.Sprintf("All %s assets are already deployed.", typeName)
}
