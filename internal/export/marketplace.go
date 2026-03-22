package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MarketplaceGenerator creates a marketplace directory structure from a
// MarketplaceConfig. Each referenced plugin is validated and copied into the
// output, and a marketplace.json manifest is written.
type MarketplaceGenerator struct{}

// Generate validates the config, copies each plugin into the output directory,
// and writes the marketplace.json manifest. The Source field in each PluginEntry
// is treated as the absolute path to an existing exported plugin on disk; in the
// output marketplace.json it is rewritten to a relative "./plugins/<name>" path.
func (g *MarketplaceGenerator) Generate(cfg MarketplaceConfig) (*MarketplaceResult, error) {
	// 1. Validate config.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 2. Check if output dir already exists.
	if _, err := os.Stat(cfg.OutputDir); err == nil {
		if !cfg.Overwrite {
			return nil, fmt.Errorf("output directory %q already exists (use --overwrite to replace)", cfg.OutputDir)
		}
		// Remove existing directory when overwriting.
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return nil, fmt.Errorf("remove existing output directory: %w", err)
		}
	}

	// 3. Create output directory structure.
	claudePluginDir := filepath.Join(cfg.OutputDir, ".claude-plugin")
	pluginsDir := filepath.Join(cfg.OutputDir, "plugins")
	if err := os.MkdirAll(claudePluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .claude-plugin dir: %w", err)
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create plugins dir: %w", err)
	}

	// 4. Validate and copy each plugin.
	for _, plugin := range cfg.Plugins {
		if err := validatePluginSource(plugin.Name, plugin.Source); err != nil {
			return nil, err
		}
		dst := filepath.Join(pluginsDir, plugin.Name)
		if err := CopyDir(plugin.Source, dst); err != nil {
			return nil, fmt.Errorf("copy plugin %q: %w", plugin.Name, err)
		}
	}

	// 5. Build marketplace.json with relative source paths.
	manifest := buildMarketplaceManifest(cfg)

	// 6. Write marketplace.json.
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal marketplace.json: %w", err)
	}
	manifestPath := filepath.Join(claudePluginDir, "marketplace.json")
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write marketplace.json: %w", err)
	}

	// 7. Return result.
	return &MarketplaceResult{
		MarketplaceDir:  cfg.OutputDir,
		MarketplaceName: cfg.Name,
		PluginCount:     len(cfg.Plugins),
	}, nil
}

// validatePluginSource checks that the source directory exists and contains a
// valid .claude-plugin/plugin.json with a "name" field.
func validatePluginSource(pluginName, sourceDir string) error {
	pluginJSONPath := filepath.Join(sourceDir, ".claude-plugin", "plugin.json")

	data, err := os.ReadFile(pluginJSONPath)
	if err != nil {
		return fmt.Errorf("plugin %q: cannot read plugin.json: %w", pluginName, err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("plugin %q: malformed plugin.json: %w", pluginName, err)
	}

	nameVal, ok := parsed["name"]
	if !ok {
		return fmt.Errorf("plugin %q: plugin.json missing required \"name\" field", pluginName)
	}
	if _, ok := nameVal.(string); !ok {
		return fmt.Errorf("plugin %q: plugin.json \"name\" must be a string", pluginName)
	}

	return nil
}

// marketplaceManifest is the JSON structure written to marketplace.json.
type marketplaceManifest struct {
	Name     string              `json:"name"`
	Owner    Author              `json:"owner"`
	Metadata *marketplaceMetdata `json:"metadata,omitempty"`
	Plugins  []PluginEntry       `json:"plugins"`
}

type marketplaceMetdata struct {
	Description string `json:"description"`
}

// buildMarketplaceManifest constructs the marketplace.json content from config,
// rewriting plugin Source fields to relative paths.
func buildMarketplaceManifest(cfg MarketplaceConfig) marketplaceManifest {
	m := marketplaceManifest{
		Name:  cfg.Name,
		Owner: cfg.Owner,
	}

	if cfg.Description != "" {
		m.Metadata = &marketplaceMetdata{Description: cfg.Description}
	}

	m.Plugins = make([]PluginEntry, len(cfg.Plugins))
	for i, p := range cfg.Plugins {
		m.Plugins[i] = PluginEntry{
			Name:        p.Name,
			Source:      "./plugins/" + p.Name,
			Description: p.Description,
			Version:     p.Version,
			Author:      p.Author,
		}
	}

	return m
}
