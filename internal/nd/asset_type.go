package nd

// AssetType represents the category of a deployable asset.
type AssetType string

const (
	AssetSkill       AssetType = "skills"
	AssetAgent       AssetType = "agents"
	AssetCommand     AssetType = "commands"
	AssetOutputStyle AssetType = "output-styles"
	AssetRule        AssetType = "rules"
	AssetContext     AssetType = "context"
	AssetPlugin      AssetType = "plugins"
	AssetHook        AssetType = "hooks"
)

// AllAssetTypes returns all recognized asset types in discovery order.
func AllAssetTypes() []AssetType {
	return []AssetType{
		AssetSkill, AssetAgent, AssetCommand, AssetOutputStyle,
		AssetRule, AssetContext, AssetPlugin, AssetHook,
	}
}

// DeployableAssetTypes returns asset types that can be deployed via symlink.
// Plugins are excluded (they use the export workflow, not symlink deployment).
func DeployableAssetTypes() []AssetType {
	return []AssetType{
		AssetSkill, AssetAgent, AssetCommand, AssetOutputStyle,
		AssetRule, AssetContext, AssetHook,
	}
}

// IsDeployable returns true if this asset type can be deployed via symlink.
// Plugins are not deployable (they use nd export + /plugin install).
func (t AssetType) IsDeployable() bool {
	return t != AssetPlugin
}

// IsDirectory returns true if this asset type deploys as a directory symlink.
func (t AssetType) IsDirectory() bool {
	switch t {
	case AssetSkill, AssetPlugin, AssetHook:
		return true
	default:
		return false
	}
}

// DeploySubdir returns the subdirectory name within an agent's config dir.
// Returns "" for context (which deploys to fixed paths determined by filename).
func (t AssetType) DeploySubdir() string {
	if t == AssetContext {
		return ""
	}
	return string(t)
}

// RequiresSettingsRegistration returns true if deploying this asset type
// requires the user to manually edit settings.json afterward.
func (t AssetType) RequiresSettingsRegistration() bool {
	switch t {
	case AssetHook, AssetOutputStyle:
		return true
	default:
		return false
	}
}
