package deploy

import (
	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
)

// Request represents a single deploy or remove operation.
type Request struct {
	Asset       asset.Asset
	Agent       agent.Agent
	Scope       nd.Scope
	ProjectRoot string
	Strategy    nd.SymlinkStrategy
	Origin      nd.DeployOrigin
	DryRun      bool
}
