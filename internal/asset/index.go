package asset

import (
	"strings"

	"github.com/larah/nd/internal/nd"
)

// Index is an in-memory collection of all discovered assets across all sources.
// Built once after source scanning, queried by the deploy engine, TUI, and CLI.
type Index struct {
	assets    []Asset
	byID      map[Identity]*Asset
	byType    map[nd.AssetType][]*Asset
	bySource  map[string][]*Asset
	conflicts []Conflict
}

// Conflict records when two sources have the same (type, name) pair.
type Conflict struct {
	Type   nd.AssetType `json:"asset_type"`
	Name   string       `json:"asset_name"`
	Winner string       `json:"winner"`
	Loser  string       `json:"loser"`
}

// conflictKey is used to detect cross-source conflicts by (type, lowercase name).
type conflictKey struct {
	Type nd.AssetType
	Name string
}

// NewIndex builds an asset index from a slice of assets,
// detecting conflicts (first source wins by registration order).
func NewIndex(assets []Asset) *Index {
	idx := &Index{
		byID:     make(map[Identity]*Asset),
		byType:   make(map[nd.AssetType][]*Asset),
		bySource: make(map[string][]*Asset),
	}

	seen := make(map[conflictKey]string) // conflictKey -> first source ID

	for i := range assets {
		a := &assets[i]
		key := conflictKey{Type: a.Type, Name: strings.ToLower(a.Name)}

		if winner, exists := seen[key]; exists {
			// Conflict: this asset's (type, name) was already claimed by another source.
			if winner != a.SourceID {
				idx.conflicts = append(idx.conflicts, Conflict{
					Type:   a.Type,
					Name:   a.Name,
					Winner: winner,
					Loser:  a.SourceID,
				})
				continue // skip the loser
			}
		}
		seen[key] = a.SourceID

		idx.assets = append(idx.assets, *a)
		stored := &idx.assets[len(idx.assets)-1]
		idx.byID[a.Identity] = stored
		idx.byType[a.Type] = append(idx.byType[a.Type], stored)
		idx.bySource[a.SourceID] = append(idx.bySource[a.SourceID], stored)
	}

	return idx
}

// Lookup finds an asset by identity. Returns nil if not found.
func (idx *Index) Lookup(id Identity) *Asset {
	return idx.byID[id]
}

// ByType returns all assets of a given type.
func (idx *Index) ByType(t nd.AssetType) []*Asset {
	return idx.byType[t]
}

// BySource returns all assets from a given source.
func (idx *Index) BySource(sourceID string) []*Asset {
	return idx.bySource[sourceID]
}

// All returns all assets in discovery order.
func (idx *Index) All() []*Asset {
	result := make([]*Asset, len(idx.assets))
	for i := range idx.assets {
		result[i] = &idx.assets[i]
	}
	return result
}

// Conflicts returns all detected duplicate-name conflicts.
func (idx *Index) Conflicts() []Conflict {
	return idx.conflicts
}
