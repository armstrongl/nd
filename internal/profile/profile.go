package profile

import (
	"fmt"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

// Profile represents a named, curated collection of assets (FR-022).
// Stored as ~/.config/nd/profiles/<name>.yaml
type Profile struct {
	Version     int            `yaml:"version"               json:"version"`
	Name        string         `yaml:"name"                  json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time      `yaml:"created_at"            json:"created_at"`
	UpdatedAt   time.Time      `yaml:"updated_at"            json:"updated_at"`
	Assets      []ProfileAsset `yaml:"assets"                json:"assets"`
}

// ProfileAsset is a reference to an asset within a profile.
// Each entry is scope-aware: one profile can mix global and project-scoped assets.
type ProfileAsset struct {
	SourceID  string       `yaml:"source_id"            json:"source_id"`
	AssetType nd.AssetType `yaml:"asset_type"           json:"asset_type"`
	AssetName string       `yaml:"asset_name"           json:"asset_name"`
	Scope     nd.Scope     `yaml:"scope"                json:"scope"`
	PathHint  string       `yaml:"path_hint,omitempty"  json:"path_hint,omitempty"`
}

// Identity returns the asset identity for this profile entry.
func (pa *ProfileAsset) Identity() asset.Identity {
	return asset.Identity{
		SourceID: pa.SourceID,
		Type:     pa.AssetType,
		Name:     pa.AssetName,
	}
}

// Validate checks the profile for internal consistency.
// Enforces: profiles must not reference plugin assets (spec line 106).
func (p *Profile) Validate() []error {
	var errs []error
	for i, a := range p.Assets {
		if a.AssetType == nd.AssetPlugin {
			errs = append(errs, fmt.Errorf("assets[%d]: plugin assets are not allowed in profiles", i))
		}
	}
	return errs
}
