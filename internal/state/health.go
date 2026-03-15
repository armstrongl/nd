package state

// HealthStatus represents the result of checking a single deployment.
type HealthStatus int

const (
	HealthOK       HealthStatus = iota // Symlink exists and points to correct target
	HealthBroken                       // Symlink exists but target is missing
	HealthDrifted                      // Symlink points to wrong target
	HealthOrphaned                     // Source no longer exists in any registered source
	HealthMissing                      // Symlink was deleted externally
)

// String returns the human-readable name of the health status.
func (h HealthStatus) String() string {
	switch h {
	case HealthOK:
		return "ok"
	case HealthBroken:
		return "broken"
	case HealthDrifted:
		return "drifted"
	case HealthOrphaned:
		return "orphaned"
	case HealthMissing:
		return "missing"
	default:
		return "unknown"
	}
}

// HealthCheck is the result of checking one deployment's health.
type HealthCheck struct {
	Deployment Deployment   `json:"deployment"`
	Status     HealthStatus `json:"status"`
	Detail     string       `json:"detail"`
}
