package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/armstrongl/nd/internal/nd"
)

// PluginExporter exports nd-managed assets into the Claude Code plugin format.
type PluginExporter struct{}

// Export creates a plugin directory from the given configuration.
// It copies assets to the correct locations, generates plugin.json and README.md,
// and returns the result including any warnings for missing source paths.
func (e *PluginExporter) Export(cfg ExportConfig) (_ *ExportResult, retErr error) {
	// 1. Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 2. Check for plugin-type assets
	for _, a := range cfg.Assets {
		if a.Type == nd.AssetPlugin {
			return nil, fmt.Errorf("plugins cannot be exported")
		}
	}

	// 3. Check if output dir exists
	if info, err := os.Stat(cfg.OutputDir); err == nil && info.IsDir() {
		if !cfg.Overwrite {
			return nil, fmt.Errorf("output directory %q already exists (use --overwrite to replace)", cfg.OutputDir)
		}
		// Remove existing directory for clean overwrite
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return nil, fmt.Errorf("remove existing output dir: %w", err)
		}
	}

	// 4. Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	// Clean up output directory on failure for atomicity.
	defer func() {
		if retErr != nil {
			os.RemoveAll(cfg.OutputDir)
		}
	}()

	result := &ExportResult{
		PluginDir:  cfg.OutputDir,
		PluginName: cfg.Name,
	}

	// 5. Group assets by type
	assetsByType := make(map[nd.AssetType][]AssetRef)
	for _, a := range cfg.Assets {
		assetsByType[a.Type] = append(assetsByType[a.Type], a)
	}

	// Track whether we have output styles for plugin.json
	hasOutputStyles := false

	// Collect hook dirs for merged processing
	var hookDirs []HookDir

	// 6. Process each asset
	for _, asset := range cfg.Assets {
		// Check if source path exists
		if _, err := os.Stat(asset.Path); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping %s %q: source path %q does not exist", asset.Type, asset.Name, asset.Path))
			continue
		}

		switch asset.Type {
		case nd.AssetSkill:
			dst := filepath.Join(cfg.OutputDir, "skills", asset.Name)
			if err := CopyDir(asset.Path, dst); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to copy skill %q: %v", asset.Name, err))
				continue
			}
			result.CopiedAssets = append(result.CopiedAssets, asset)

		case nd.AssetAgent:
			dst := filepath.Join(cfg.OutputDir, "agents", asset.Name)
			if err := CopyFile(asset.Path, dst); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to copy agent %q: %v", asset.Name, err))
				continue
			}
			result.CopiedAssets = append(result.CopiedAssets, asset)

		case nd.AssetCommand:
			dst := filepath.Join(cfg.OutputDir, "commands", asset.Name)
			if err := CopyFile(asset.Path, dst); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to copy command %q: %v", asset.Name, err))
				continue
			}
			result.CopiedAssets = append(result.CopiedAssets, asset)

		case nd.AssetOutputStyle:
			dst := filepath.Join(cfg.OutputDir, "output-styles", asset.Name)
			if err := CopyFile(asset.Path, dst); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to copy output-style %q: %v", asset.Name, err))
				continue
			}
			hasOutputStyles = true
			result.CopiedAssets = append(result.CopiedAssets, asset)

		case nd.AssetHook:
			hookDirs = append(hookDirs, HookDir{Name: asset.Name, Path: asset.Path})

		case nd.AssetRule:
			if asset.IsDir {
				dst := filepath.Join(cfg.OutputDir, "extras", "rules", asset.Name)
				if err := CopyDir(asset.Path, dst); err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("failed to copy rule dir %q: %v", asset.Name, err))
					continue
				}
			} else {
				dst := filepath.Join(cfg.OutputDir, "extras", "rules", asset.Name)
				if err := CopyFile(asset.Path, dst); err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("failed to copy rule %q: %v", asset.Name, err))
					continue
				}
			}
			result.BundledExtras = append(result.BundledExtras, asset)

		case nd.AssetContext:
			// Path points to the context file (e.g., CLAUDE.md); use Dir to get the folder
			contextFolder := filepath.Dir(asset.Path)
			dst := filepath.Join(cfg.OutputDir, "extras", "context", asset.Name)
			if err := CopyDir(contextFolder, dst); err != nil {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("failed to copy context %q: %v", asset.Name, err))
				continue
			}
			result.BundledExtras = append(result.BundledExtras, asset)
		}
	}

	// 7. Process hooks: merge and write (failure aborts export — hooks are not optional if selected)
	if len(hookDirs) > 0 {
		merged, err := MergeHooks(hookDirs)
		if err != nil {
			return nil, fmt.Errorf("hook merging failed: %w", err)
		}

		// Write merged hooks.json
		hooksData, err := json.MarshalIndent(merged.Config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal merged hooks.json: %w", err)
		}
		hooksDir := filepath.Join(cfg.OutputDir, "hooks")
		if err := os.MkdirAll(hooksDir, 0o755); err != nil {
			return nil, fmt.Errorf("create hooks dir: %w", err)
		}
		if err := os.WriteFile(filepath.Join(hooksDir, "hooks.json"), hooksData, 0o644); err != nil {
			return nil, fmt.Errorf("write hooks.json: %w", err)
		}

		// Copy scripts (failure aborts — hooks.json references these scripts)
		scriptsDir := filepath.Join(cfg.OutputDir, "scripts")
		for srcPath, relDest := range merged.Scripts {
			dst := filepath.Join(scriptsDir, relDest)
			if err := CopyFile(srcPath, dst); err != nil {
				return nil, fmt.Errorf("copy hook script %q: %w", relDest, err)
			}
		}

		// Track hook assets as copied
		for _, hd := range hookDirs {
			for _, a := range cfg.Assets {
				if a.Type == nd.AssetHook && a.Name == hd.Name {
					result.CopiedAssets = append(result.CopiedAssets, a)
					break
				}
			}
		}
	}

	// 8. Check if all assets failed
	if len(result.CopiedAssets) == 0 && len(result.BundledExtras) == 0 {
		return nil, fmt.Errorf("all assets failed to export: no assets were copied or bundled")
	}

	// 9. Generate plugin.json
	pluginJSON, err := generatePluginJSON(cfg, hasOutputStyles)
	if err != nil {
		return nil, fmt.Errorf("generate plugin.json: %w", err)
	}
	pluginDir := filepath.Join(cfg.OutputDir, ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .claude-plugin dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), pluginJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write plugin.json: %w", err)
	}

	// 10. Generate README.md
	readme := generateREADME(cfg, result)
	if err := os.WriteFile(filepath.Join(cfg.OutputDir, "README.md"), []byte(readme), 0o644); err != nil {
		return nil, fmt.Errorf("write README.md: %w", err)
	}

	return result, nil
}

// generatePluginJSON creates the plugin.json content with conditional fields.
func generatePluginJSON(cfg ExportConfig, hasOutputStyles bool) ([]byte, error) {
	// Use an ordered map approach via json.Marshal on a struct-like map.
	// We build a map and only include non-empty optional fields.
	m := make(map[string]any)

	m["name"] = cfg.Name
	m["version"] = cfg.Version
	if cfg.Description != "" {
		m["description"] = cfg.Description
	}
	if cfg.Author.Name != "" {
		m["author"] = cfg.Author
	}
	if cfg.Homepage != "" {
		m["homepage"] = cfg.Homepage
	}
	if cfg.Repository != "" {
		m["repository"] = cfg.Repository
	}
	if cfg.License != "" {
		m["license"] = cfg.License
	}
	if len(cfg.Keywords) > 0 {
		m["keywords"] = cfg.Keywords
	}
	if hasOutputStyles {
		m["outputStyles"] = "./output-styles/"
	}

	return json.MarshalIndent(m, "", "  ")
}

// generateREADME creates the README.md content for the plugin.
func generateREADME(cfg ExportConfig, result *ExportResult) string {
	var sb strings.Builder

	// Title
	sb.WriteString("# " + cfg.Name + "\n")

	// Description
	if cfg.Description != "" {
		sb.WriteString("\n" + cfg.Description + "\n")
	}

	// Installation
	sb.WriteString("\n## Installation\n\n")
	sb.WriteString("Install with: `/plugin install <source>`\n")

	// Assets listing grouped by type
	assetsByType := make(map[nd.AssetType][]AssetRef)
	for _, a := range result.CopiedAssets {
		assetsByType[a.Type] = append(assetsByType[a.Type], a)
	}

	// Use a stable ordering of types
	typeOrder := []nd.AssetType{
		nd.AssetSkill, nd.AssetAgent, nd.AssetCommand,
		nd.AssetOutputStyle, nd.AssetHook,
	}
	typeLabels := map[nd.AssetType]string{
		nd.AssetSkill:       "Skills",
		nd.AssetAgent:       "Agents",
		nd.AssetCommand:     "Commands",
		nd.AssetOutputStyle: "Output Styles",
		nd.AssetHook:        "Hooks",
	}

	for _, at := range typeOrder {
		assets, ok := assetsByType[at]
		if !ok || len(assets) == 0 {
			continue
		}
		sb.WriteString("\n## " + typeLabels[at] + "\n\n")
		for _, a := range assets {
			sb.WriteString("- `" + a.Name + "`\n")
		}
	}

	// Extras section
	if len(result.BundledExtras) > 0 {
		sb.WriteString("\n## Extras (manual deployment required)\n\n")
		sb.WriteString("The following assets are included but are NOT automatically loaded by Claude Code's plugin system.\n")

		// Group extras by type
		var rules []AssetRef
		var contexts []AssetRef
		for _, a := range result.BundledExtras {
			switch a.Type {
			case nd.AssetRule:
				rules = append(rules, a)
			case nd.AssetContext:
				contexts = append(contexts, a)
			}
		}

		if len(rules) > 0 {
			sb.WriteString("\n### Rules\n\n")
			sb.WriteString("Copy to your `.claude/rules/` directory:\n")
			for _, r := range rules {
				sb.WriteString("- `extras/rules/" + r.Name + "`\n")
			}
		}

		if len(contexts) > 0 {
			sb.WriteString("\n### Context files\n\n")
			sb.WriteString("Copy to your project root or `~/.claude/`:\n")
			for _, c := range contexts {
				sb.WriteString("- `extras/context/" + c.Name + "/`\n")
			}
		}
	}

	return sb.String()
}
