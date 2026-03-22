package export

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/armstrongl/nd/internal/nd"
)

// ValidateAssetName checks that a name is safe for use in file paths.
// It rejects names containing path separators, traversal patterns, or that are empty.
func ValidateAssetName(name string) error {
	if name == "" {
		return fmt.Errorf("asset name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("asset name %q contains path separators", name)
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return fmt.Errorf("asset name %q contains path traversal", name)
	}
	if len(name) > 255 {
		return fmt.Errorf("asset name %q exceeds maximum length of 255 characters", name)
	}
	return nil
}

// ValidateOutputDir checks that an output directory path is safe for
// destructive operations (overwrite removes the directory).
func ValidateOutputDir(dir string) error {
	clean := filepath.Clean(dir)
	if clean == "." || clean == ".." || clean == "/" {
		return fmt.Errorf("output directory %q is not a safe target for export", dir)
	}
	return nil
}

// kebabCaseRe matches valid kebab-case strings: lowercase alphanumeric
// segments separated by single hyphens, no leading or trailing hyphens.
var kebabCaseRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether s is a valid kebab-case identifier.
func IsKebabCase(s string) bool {
	return kebabCaseRe.MatchString(s)
}

// Author represents a plugin or marketplace author.
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// AssetRef identifies a single asset to include in an export.
type AssetRef struct {
	Type     nd.AssetType `json:"type"`
	Name     string       `json:"name"`
	SourceID string       `json:"source_id,omitempty"`
	Path     string       `json:"path"`
	IsDir    bool         `json:"is_dir,omitempty"`
}

// ExportConfig holds the configuration for a plugin export operation.
type ExportConfig struct {
	Name        string     `json:"name"`
	Version     string     `json:"version,omitempty"`
	Description string     `json:"description,omitempty"`
	Author      Author     `json:"author,omitempty"`
	Homepage    string     `json:"homepage,omitempty"`
	Repository  string     `json:"repository,omitempty"`
	License     string     `json:"license,omitempty"`
	Keywords    []string   `json:"keywords,omitempty"`
	OutputDir   string     `json:"outputDir"`
	Assets      []AssetRef `json:"assets"`
	Overwrite   bool       `json:"overwrite,omitempty"`
}

// Validate checks that the ExportConfig has all required fields.
// It defaults Version to "1.0.0" if empty.
func (c *ExportConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !IsKebabCase(c.Name) {
		return fmt.Errorf("name must be kebab-case (got %q)", c.Name)
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if err := ValidateOutputDir(c.OutputDir); err != nil {
		return err
	}
	if len(c.Assets) == 0 {
		return fmt.Errorf("at least one asset is required")
	}
	for i, a := range c.Assets {
		if a.Path == "" {
			return fmt.Errorf("asset %d (%s): path is required", i, a.Name)
		}
		if err := ValidateAssetName(a.Name); err != nil {
			return fmt.Errorf("asset %d: %w", i, err)
		}
	}
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	return nil
}

// PluginEntry represents a plugin reference in a marketplace configuration.
type PluginEntry struct {
	Name        string  `json:"name"`
	Source      string  `json:"source"`
	Description string  `json:"description,omitempty"`
	Version     string  `json:"version,omitempty"`
	Author      *Author `json:"author,omitempty"`
}

// MarketplaceConfig holds the configuration for a marketplace generation.
type MarketplaceConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Owner       Author        `json:"owner"`
	Plugins     []PluginEntry `json:"plugins"`
	OutputDir   string        `json:"outputDir"`
	Overwrite   bool          `json:"overwrite,omitempty"`
}

// Validate checks that the MarketplaceConfig has all required fields.
func (c *MarketplaceConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !IsKebabCase(c.Name) {
		return fmt.Errorf("name must be kebab-case (got %q)", c.Name)
	}
	if c.Owner.Name == "" {
		return fmt.Errorf("owner name is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	if err := ValidateOutputDir(c.OutputDir); err != nil {
		return err
	}
	if len(c.Plugins) == 0 {
		return fmt.Errorf("at least one plugin is required")
	}
	for i, p := range c.Plugins {
		if err := ValidateAssetName(p.Name); err != nil {
			return fmt.Errorf("plugin %d: %w", i, err)
		}
	}
	return nil
}

// ExportResult holds the outcome of a plugin export operation.
type ExportResult struct {
	PluginDir     string     `json:"pluginDir"`
	PluginName    string     `json:"pluginName"`
	CopiedAssets  []AssetRef `json:"copiedAssets"`
	BundledExtras []AssetRef `json:"bundledExtras,omitempty"`
	Warnings      []string   `json:"warnings,omitempty"`
}

// MarketplaceResult holds the outcome of a marketplace generation.
type MarketplaceResult struct {
	MarketplaceDir  string   `json:"marketplaceDir"`
	MarketplaceName string   `json:"marketplaceName"`
	PluginCount     int      `json:"pluginCount"`
	Warnings        []string `json:"warnings,omitempty"`
}
