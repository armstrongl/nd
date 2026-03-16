package deploy

import "github.com/armstrongl/nd/internal/asset"

// Result represents the outcome of a single deploy/remove operation.
type Result struct {
	Request  Request        `json:"-"`
	AssetID  asset.Identity `json:"asset"`
	Success  bool           `json:"success"`
	Action   Action         `json:"action"`
	Error    error          `json:"-"`
	ErrorMsg string         `json:"error,omitempty"`
	LinkPath string         `json:"link_path"`
}
