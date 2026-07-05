package state

import (
	"fmt"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
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
	SourceID    string             `yaml:"source_id"                json:"source_id"`
	AssetType   nd.AssetType       `yaml:"asset_type"               json:"asset_type"`
	AssetName   string             `yaml:"asset_name"               json:"asset_name"`
	SourcePath  string             `yaml:"source_path"              json:"source_path"`
	LinkPath    string             `yaml:"link_path"                json:"link_path"`
	Scope       nd.Scope           `yaml:"scope"                    json:"scope"`
	ProjectPath string             `yaml:"project_path,omitempty"   json:"project_path,omitempty"`
	Origin      nd.DeployOrigin    `yaml:"origin"                   json:"origin"`
	Agent       string             `yaml:"agent"                    json:"agent"`
	Strategy    nd.SymlinkStrategy `yaml:"strategy,omitempty"       json:"strategy,omitempty"`
	DeployedAt  time.Time          `yaml:"deployed_at"              json:"deployed_at"`
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
// Reports duplicate deployment identities (keyed on {SourceID, AssetType,
// AssetName}) and any deployment whose scope is not global or project.
// A nil/empty result means the state is structurally valid.
func (s *DeploymentState) Validate() []error {
	var errs []error
	seen := make(map[asset.Identity]bool)
	for i := range s.Deployments {
		d := &s.Deployments[i]
		id := d.Identity()
		if seen[id] {
			errs = append(errs, fmt.Errorf("deployments[%d]: duplicate identity %s/%s from %s", i, id.Type, id.Name, id.SourceID))
		}
		seen[id] = true

		switch d.Scope {
		case nd.ScopeGlobal, nd.ScopeProject:
		default:
			errs = append(errs, fmt.Errorf("deployments[%d]: invalid scope %q", i, d.Scope))
		}
	}
	return errs
}
