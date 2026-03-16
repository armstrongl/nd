package asset

import (
	"github.com/armstrongl/nd/internal/nd"
)

// Identity uniquely identifies an asset across all sources.
// Used as the primary key in profiles, snapshots, and deployment state.
// The tuple (SourceID, Type, Name) is globally unique.
type Identity struct {
	SourceID string       `yaml:"source_id"  json:"source_id"`
	Type     nd.AssetType `yaml:"asset_type" json:"asset_type"`
	Name     string       `yaml:"asset_name" json:"asset_name"`
}

// String returns "source:type/name" for display and logging.
func (id Identity) String() string {
	subdir := id.Type.DeploySubdir()
	if subdir == "" {
		return id.SourceID + ":" + id.Name
	}
	return id.SourceID + ":" + subdir + "/" + id.Name
}
