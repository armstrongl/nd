package sourcemanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/source"
	"gopkg.in/yaml.v3"
)

// maxManifestSize is the maximum size of an nd-source.yaml file (NFR-013).
const maxManifestSize = 1024 * 1024 // 1MB

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

// ScanSource scans a single source directory for assets.
// If nd-source.yaml exists, uses manifest paths. Otherwise uses convention-based discovery.
func ScanSource(sourceID string, rootPath string) source.ScanResult {
	result := source.ScanResult{SourceID: sourceID}

	info, err := os.Stat(rootPath)
	if err != nil || !info.IsDir() {
		result.Warnings = append(result.Warnings,
			"source "+sourceID+" at "+rootPath+" is unavailable")
		return result
	}

	// Check for manifest
	manifestPath := filepath.Join(rootPath, "nd-source.yaml")
	if manifest, err := loadManifest(manifestPath, rootPath); err != nil {
		result.Errors = append(result.Errors, err)
		return result
	} else if manifest != nil {
		scanWithManifest(&result, sourceID, rootPath, manifest)
		return result
	}

	// Convention-based discovery
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
// Context assets use a folder-per-asset layout (FR-016b):
//
//	context/
//	  go-project-rules/
//	    CLAUDE.md
//	    _meta.yaml
func scanContextDir(result *source.ScanResult, sourceID string, dirPath string) {
	folders, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, folder := range folders {
		if !folder.IsDir() || folder.Name()[0] == '.' {
			continue
		}

		folderPath := filepath.Join(dirPath, folder.Name())
		contextFile := findContextFile(folderPath)
		if contextFile == "" {
			result.Warnings = append(result.Warnings,
				"context folder "+folder.Name()+" has no context file")
			continue
		}

		a := asset.Asset{
			Identity: asset.Identity{
				SourceID: sourceID,
				Type:     nd.AssetContext,
				Name:     folder.Name(),
			},
			SourcePath: filepath.Join(folderPath, contextFile),
			IsDir:      false,
			ContextFile: &asset.ContextInfo{
				FolderName: folder.Name(),
				FileName:   contextFile,
			},
		}

		// Load optional _meta.yaml
		metaPath := filepath.Join(folderPath, "_meta.yaml")
		if meta, err := loadContextMeta(metaPath); err == nil && meta != nil {
			a.Meta = meta
		}

		result.Assets = append(result.Assets, a)
	}
}

// findContextFile looks for a recognized context file in a folder.
func findContextFile(folderPath string) string {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '_' {
			continue
		}
		// Accept any .md file that isn't _meta.yaml
		if filepath.Ext(e.Name()) == ".md" {
			return e.Name()
		}
	}
	return ""
}

// loadManifest reads and validates an nd-source.yaml file.
// Returns nil, nil if the file does not exist.
func loadManifest(path string, sourceRoot string) (*source.Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	if info.Size() > maxManifestSize {
		return nil, fmt.Errorf("manifest %s is %d bytes, maximum is %d (NFR-013)", path, info.Size(), maxManifestSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m source.Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	if errs := m.Validate(sourceRoot); len(errs) > 0 {
		return nil, errs[0]
	}

	return &m, nil
}

// scanWithManifest scans using manifest-defined paths instead of conventions.
// Respects the manifest's Exclude list (FR-008).
func scanWithManifest(result *source.ScanResult, sourceID string, rootPath string, m *source.Manifest) {
	excludeSet := make(map[string]bool)
	for _, e := range m.Exclude {
		excludeSet[strings.TrimSuffix(e, "/")] = true
	}

	for assetType, paths := range m.Paths {
		for _, p := range paths {
			dirPath := filepath.Join(rootPath, p)
			info, err := os.Stat(dirPath)
			if err != nil || !info.IsDir() {
				result.Warnings = append(result.Warnings,
					"manifest path "+p+" for "+string(assetType)+" not found")
				continue
			}

			if assetType == nd.AssetContext {
				scanContextDir(result, sourceID, dirPath)
			} else {
				scanAssetDirExcluding(result, sourceID, assetType, dirPath, excludeSet)
			}
		}
	}
}

// scanAssetDirExcluding is like scanAssetDir but skips entries matching the exclude set.
func scanAssetDirExcluding(result *source.ScanResult, sourceID string, assetType nd.AssetType, dirPath string, excludeSet map[string]bool) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if excludedDirs[name] || name[0] == '.' || excludeSet[name] {
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

// loadContextMeta loads and validates a _meta.yaml file.
func loadContextMeta(path string) (*asset.ContextMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta asset.ContextMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
