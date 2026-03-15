package asset

import (
	"strings"

	"github.com/larah/nd/internal/nd"
)

// SearchByName returns all assets matching the given name (case-insensitive)
// across all types and sources. Used by the CLI for name-only asset resolution.
func (idx *Index) SearchByName(name string) []*Asset {
	lower := strings.ToLower(name)
	var matches []*Asset
	for i := range idx.assets {
		if strings.ToLower(idx.assets[i].Name) == lower {
			matches = append(matches, &idx.assets[i])
		}
	}
	return matches
}

// SearchByTypeAndName returns the first asset matching the given type and name
// (case-insensitive name match). Returns nil if not found.
// Used by the CLI for type-qualified asset resolution (e.g., "skills/my-skill").
func (idx *Index) SearchByTypeAndName(assetType nd.AssetType, name string) *Asset {
	lower := strings.ToLower(name)
	for i := range idx.assets {
		if idx.assets[i].Type == assetType && strings.ToLower(idx.assets[i].Name) == lower {
			return &idx.assets[i]
		}
	}
	return nil
}
