package deploy

import "github.com/larah/nd/internal/state"

// UninstallPlan represents what nd uninstall --dry-run would do.
type UninstallPlan struct {
	Symlinks     []state.Deployment `json:"symlinks"`
	Directories  []string           `json:"directories"`
	SymlinkCount int                `json:"symlink_count"`
}
