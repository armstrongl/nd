package agent

import (
	"fmt"
	"path/filepath"

	"github.com/larah/nd/internal/nd"
)

// Agent represents a detected coding agent installation.
type Agent struct {
	Name       string `json:"name"`
	GlobalDir  string `json:"global_dir"`
	ProjectDir string `json:"project_dir"`
	Detected   bool   `json:"detected"`
	InPath     bool   `json:"in_path"`
}

// DeployPath computes the full path where an asset's symlink should be created.
// Handles the special cases:
//   - Context files deploy to project root (not inside .claude/) at project scope
//   - Context files deploy to the agent's global dir (not a subdirectory) at global scope
//   - .local.md context files deploy only at project scope; returns an error
//     if scope=global and IsLocalOnlyContext(contextFile) is true
//   - All other types deploy to <configDir>/<assetType>/<assetName>
//
// contextFile is the filename inside the context folder (e.g., "CLAUDE.md").
// Pass "" for non-context asset types.
// Returns ("", error) for invalid combinations (e.g., global + .local.md).
func (a *Agent) DeployPath(
	assetType nd.AssetType,
	assetName string,
	scope nd.Scope,
	projectRoot string,
	contextFile string,
) (string, error) {
	if assetType == nd.AssetContext {
		return a.contextDeployPath(scope, projectRoot, contextFile)
	}

	configDir := a.configDir(scope, projectRoot)
	subdir := assetType.DeploySubdir()
	return filepath.Join(configDir, subdir, assetName), nil
}

// contextDeployPath handles the special deployment rules for context files.
func (a *Agent) contextDeployPath(scope nd.Scope, projectRoot, contextFile string) (string, error) {
	if nd.IsLocalOnlyContext(contextFile) && scope == nd.ScopeGlobal {
		return "", fmt.Errorf("context file %q is local-only and cannot be deployed at global scope", contextFile)
	}

	if scope == nd.ScopeProject {
		// Context files deploy to the project root, not inside .claude/
		return filepath.Join(projectRoot, contextFile), nil
	}

	// Global scope: deploy to the agent's global dir
	return filepath.Join(a.GlobalDir, contextFile), nil
}

// configDir returns the agent configuration directory for the given scope.
func (a *Agent) configDir(scope nd.Scope, projectRoot string) string {
	if scope == nd.ScopeProject {
		return filepath.Join(projectRoot, a.ProjectDir)
	}
	return a.GlobalDir
}

// DetectionResult holds the output of scanning for installed agents.
type DetectionResult struct {
	Agents   []Agent  `json:"agents"`
	Warnings []string `json:"warnings,omitempty"`
}
