package state

import (
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

// FindByIdentity returns deployments matching an asset identity.
func (s *DeploymentState) FindByIdentity(id asset.Identity) []Deployment {
	var result []Deployment
	for _, d := range s.Deployments {
		if d.SourceID == id.SourceID && d.AssetType == id.Type && d.AssetName == id.Name {
			result = append(result, d)
		}
	}
	return result
}

// FindByScope returns all deployments for a given scope.
func (s *DeploymentState) FindByScope(scope nd.Scope) []Deployment {
	var result []Deployment
	for _, d := range s.Deployments {
		if d.Scope == scope {
			result = append(result, d)
		}
	}
	return result
}

// FindByOrigin returns all deployments with a specific origin.
func (s *DeploymentState) FindByOrigin(origin nd.DeployOrigin) []Deployment {
	var result []Deployment
	for _, d := range s.Deployments {
		if d.Origin == origin {
			result = append(result, d)
		}
	}
	return result
}

// FindByProject returns all project-scoped deployments for a given project path.
func (s *DeploymentState) FindByProject(projectPath string) []Deployment {
	var result []Deployment
	for _, d := range s.Deployments {
		if d.Scope == nd.ScopeProject && d.ProjectPath == projectPath {
			result = append(result, d)
		}
	}
	return result
}

// FindByAgent returns all deployments for a given agent.
// Treats empty Agent as "claude-code" for backward compatibility with
// partially migrated state (v1 records that haven't been persisted yet).
func (s *DeploymentState) FindByAgent(agentName string) []Deployment {
	var result []Deployment
	for _, d := range s.Deployments {
		if d.Agent == agentName || (d.Agent == "" && agentName == "claude-code") {
			result = append(result, d)
		}
	}
	return result
}
