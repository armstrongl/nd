package deploy

import "github.com/larah/nd/internal/state"

// SyncPlan represents what the sync command will do.
type SyncPlan struct {
	Repairs []SyncAction `json:"repairs"`
	Removes []SyncAction `json:"removes"`
	Healthy int          `json:"healthy"`
}

// SyncAction describes a single repair or removal during sync.
type SyncAction struct {
	Deployment state.Deployment  `json:"deployment"`
	Health     state.HealthCheck `json:"health"`
	Action     Action            `json:"action"`
}
