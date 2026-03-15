package profile

import (
	"fmt"
	"time"

	"github.com/larah/nd/internal/nd"
)

// Snapshot represents a point-in-time record of all deployments (FR-020).
// User-created: ~/.config/nd/snapshots/user/<name>.yaml
// Auto-created: ~/.config/nd/snapshots/auto/auto-<timestamp>.yaml (last 5 retained)
type Snapshot struct {
	Version     int             `yaml:"version"     json:"version"`
	Name        string          `yaml:"name"        json:"name"`
	CreatedAt   time.Time       `yaml:"created_at"  json:"created_at"`
	Auto        bool            `yaml:"auto"        json:"auto"`
	Deployments []SnapshotEntry `yaml:"deployments" json:"deployments"`
}

// SnapshotEntry captures the exact state of one deployment at snapshot time.
// Intentionally a full copy (not a reference) -- snapshots are immutable
// records that remain valid even if sources or profiles change later.
type SnapshotEntry struct {
	SourceID    string          `yaml:"source_id"              json:"source_id"`
	AssetType   nd.AssetType    `yaml:"asset_type"             json:"asset_type"`
	AssetName   string          `yaml:"asset_name"             json:"asset_name"`
	SourcePath  string          `yaml:"source_path"            json:"source_path"`
	LinkPath    string          `yaml:"link_path"              json:"link_path"`
	Scope       nd.Scope        `yaml:"scope"                  json:"scope"`
	ProjectPath string          `yaml:"project_path,omitempty" json:"project_path,omitempty"`
	Origin      nd.DeployOrigin `yaml:"origin"                 json:"origin"`
	DeployedAt  time.Time       `yaml:"deployed_at"            json:"deployed_at"`
}

// Validate checks the snapshot for internal consistency.
// Enforces: snapshots must not reference plugin assets (spec line 106).
func (s *Snapshot) Validate() []error {
	var errs []error
	for i, d := range s.Deployments {
		if d.AssetType == nd.AssetPlugin {
			errs = append(errs, fmt.Errorf("deployments[%d]: plugin assets are not allowed in snapshots", i))
		}
	}
	return errs
}
