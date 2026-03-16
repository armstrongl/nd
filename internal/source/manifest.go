package source

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/armstrongl/nd/internal/nd"
)

// Manifest represents an nd-source.yaml file (FR-008).
// Overrides convention-based discovery with custom paths and exclusions.
type Manifest struct {
	Version  int                       `yaml:"version"`
	Paths    map[nd.AssetType][]string `yaml:"paths"`
	Exclude  []string                  `yaml:"exclude,omitempty"`
	Metadata *ManifestMetadata         `yaml:"metadata,omitempty"`
}

// ManifestMetadata is optional metadata about the source itself.
type ManifestMetadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Author      string   `yaml:"author,omitempty"`
	URL         string   `yaml:"url,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// maxPathEntries is the limit on total path entries across all asset types (NFR-013).
const maxPathEntries = 1000

// Validate checks the manifest for correctness.
// Enforces:
//   - NFR-012: all paths resolve within sourceRoot (no path traversal).
//   - NFR-013: path lists limited to 1,000 entries.
func (m *Manifest) Validate(sourceRoot string) []error {
	var errs []error

	totalPaths := 0
	for assetType, paths := range m.Paths {
		totalPaths += len(paths)
		for _, p := range paths {
			resolved := filepath.Join(sourceRoot, p)
			rel, err := filepath.Rel(sourceRoot, resolved)
			if err != nil || strings.HasPrefix(rel, "..") {
				errs = append(errs, &nd.PathTraversalError{
					Path:     p,
					Root:     sourceRoot,
					SourceID: fmt.Sprintf("manifest[%s]", assetType),
				})
			}
		}
	}

	if totalPaths > maxPathEntries {
		errs = append(errs, fmt.Errorf("manifest has %d path entries, maximum is %d", totalPaths, maxPathEntries))
	}

	return errs
}
