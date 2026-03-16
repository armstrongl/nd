package source

import "github.com/armstrongl/nd/internal/asset"

// ScanResult holds the output of scanning a single source.
type ScanResult struct {
	SourceID string
	Assets   []asset.Asset
	Warnings []string // Non-fatal issues (e.g., unreadable directories)
	Errors   []error  // Fatal issues for specific paths
}
