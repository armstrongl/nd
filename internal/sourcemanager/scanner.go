package sourcemanager

import (
	"os"
	"path/filepath"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/source"
)

// excludedDirs are directories that source scanning always skips (NFR-017).
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
}

// dirToAssetType maps conventional directory names to asset types.
var dirToAssetType = map[string]nd.AssetType{
	"skills":        nd.AssetSkill,
	"agents":        nd.AssetAgent,
	"commands":      nd.AssetCommand,
	"output-styles": nd.AssetOutputStyle,
	"rules":         nd.AssetRule,
	"context":       nd.AssetContext,
	"plugins":       nd.AssetPlugin,
	"hooks":         nd.AssetHook,
}

// ScanSource scans a single source directory for assets using convention-based
// discovery. Returns a ScanResult with discovered assets, warnings, and errors.
func ScanSource(sourceID string, rootPath string) source.ScanResult {
	result := source.ScanResult{SourceID: sourceID}

	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		result.Warnings = append(result.Warnings,
			"source "+sourceID+" at "+rootPath+" is unavailable")
		return result
	}

	for dirName, assetType := range dirToAssetType {
		dirPath := filepath.Join(rootPath, dirName)
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			continue
		}

		if assetType == nd.AssetContext {
			scanContextDir(&result, sourceID, dirPath)
			continue
		}

		scanAssetDir(&result, sourceID, assetType, dirPath)
	}

	return result
}

// scanAssetDir scans a single asset type directory for entries.
func scanAssetDir(result *source.ScanResult, sourceID string, assetType nd.AssetType, dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if excludedDirs[name] || name[0] == '.' {
			continue
		}

		result.Assets = append(result.Assets, asset.Asset{
			Identity: asset.Identity{
				SourceID: sourceID,
				Type:     assetType,
				Name:     name,
			},
			SourcePath: filepath.Join(dirPath, name),
			IsDir:      entry.IsDir(),
		})
	}
}

// scanContextDir scans the context/ directory for context assets.
// Context assets use a folder-per-asset layout with optional _meta.yaml.
func scanContextDir(result *source.ScanResult, sourceID string, dirPath string) {
	// Implemented in Task 10
}
