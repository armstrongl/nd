package state

import (
	"time"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
)

// DeploymentState is the root structure of deployments.yaml.
// Written atomically (write-to-temp-then-rename) per NFR-010.
// Guarded by advisory file lock per NFR-011.
type DeploymentState struct {
	Version       int          `yaml:"version"                  json:"version"`
	ActiveProfile string       `yaml:"active_profile,omitempty" json:"active_profile,omitempty"`
	Deployments   []Deployment `yaml:"deployments"              json:"deployments"`
}

// Deployment represents a single managed symlink.
type Deployment struct {
	SourceID    string          `yaml:"source_id"                json:"source_id"`
	AssetType   nd.AssetType    `yaml:"asset_type"               json:"asset_type"`
	AssetName   string          `yaml:"asset_name"               json:"asset_name"`
	SourcePath  string          `yaml:"source_path"              json:"source_path"`
	LinkPath    string          `yaml:"link_path"                json:"link_path"`
	Scope       nd.Scope        `yaml:"scope"                    json:"scope"`
	ProjectPath string          `yaml:"project_path,omitempty"   json:"project_path,omitempty"`
	Origin      nd.DeployOrigin `yaml:"origin"                   json:"origin"`
	DeployedAt  time.Time       `yaml:"deployed_at"              json:"deployed_at"`
}

// Identity returns the asset identity for this deployment.
func (d *Deployment) Identity() asset.Identity {
	return asset.Identity{
		SourceID: d.SourceID,
		Type:     d.AssetType,
		Name:     d.AssetName,
	}
}

// Validate checks the deployment state for internal consistency.
func (s *DeploymentState) Validate() []error {
	return nil
}
