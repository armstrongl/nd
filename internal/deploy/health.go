package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/armstrongl/nd/internal/state"
)

// StatusEntry pairs a deployment with its health status.
type StatusEntry struct {
	Deployment state.Deployment
	Health     state.HealthStatus
	Detail     string
}

// SyncResult holds the outcomes of a sync/repair operation.
type SyncResult struct {
	Repaired []state.Deployment
	Removed  []state.Deployment
	Warnings []string
}

// Check detects deployment health issues (FR-013).
// Returns only unhealthy entries. Empty slice means all healthy.
func (e *Engine) Check() ([]state.HealthCheck, error) {
	var issues []state.HealthCheck

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range st.Deployments {
			if hc := e.checkOne(dep); hc.Status != state.HealthOK {
				issues = append(issues, hc)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return issues, nil
}

// checkOne evaluates the health of a single deployment.
func (e *Engine) checkOne(dep state.Deployment) state.HealthCheck {
	hc := state.HealthCheck{Deployment: dep, Status: state.HealthOK}

	// Step 1: Does the symlink node exist?
	_, err := e.lstat(dep.LinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			hc.Status = state.HealthMissing
			hc.Detail = fmt.Sprintf("symlink %s does not exist", dep.LinkPath)
			return hc
		}
		hc.Status = state.HealthBroken
		hc.Detail = fmt.Sprintf("cannot access %s: %v", dep.LinkPath, err)
		return hc
	}

	// Step 2: Does it point to the expected target?
	target, err := e.readlink(dep.LinkPath)
	if err != nil {
		hc.Status = state.HealthBroken
		hc.Detail = fmt.Sprintf("cannot read symlink %s: %v", dep.LinkPath, err)
		return hc
	}
	if target != dep.SourcePath {
		hc.Status = state.HealthDrifted
		hc.Detail = fmt.Sprintf("symlink points to %s, expected %s", target, dep.SourcePath)
		return hc
	}

	// Step 3: Does the target actually exist? (follows symlinks)
	if _, err := e.stat(dep.LinkPath); err != nil {
		hc.Status = state.HealthBroken
		hc.Detail = fmt.Sprintf("target %s does not exist", dep.SourcePath)
		return hc
	}

	return hc
}

// Sync repairs detected deployment issues (FR-014).
func (e *Engine) Sync() (*SyncResult, error) {
	var result SyncResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		var keep []state.Deployment
		for _, dep := range st.Deployments {
			hc := e.checkOne(dep)
			switch hc.Status {
			case state.HealthOK:
				keep = append(keep, dep)

			case state.HealthBroken, state.HealthOrphaned:
				// Source gone: remove symlink and state entry
				e.remove(dep.LinkPath)
				result.Removed = append(result.Removed, dep)

			case state.HealthMissing:
				// Symlink deleted externally: re-create if source exists
				if _, err := e.stat(dep.SourcePath); err == nil {
					e.mkdirAll(filepath.Dir(dep.LinkPath), 0o755)
					if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
						result.Repaired = append(result.Repaired, dep)
						keep = append(keep, dep)
					} else {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Failed to re-create %s: %v", dep.LinkPath, err))
						result.Removed = append(result.Removed, dep)
					}
				} else {
					// Source also gone
					result.Removed = append(result.Removed, dep)
				}

			case state.HealthDrifted:
				// Re-create symlink to correct target
				e.remove(dep.LinkPath)
				if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
					result.Repaired = append(result.Repaired, dep)
					keep = append(keep, dep)
				} else {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Failed to repair %s: %v", dep.LinkPath, err))
					keep = append(keep, dep) // keep entry, it might be fixable later
				}
			}
		}

		st.Deployments = keep
		return e.store.Save(st)
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Status returns all deployments with their health status (FR-015).
// Returns a flat list; grouping by type is the caller's responsibility.
func (e *Engine) Status() ([]StatusEntry, error) {
	var entries []StatusEntry

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range st.Deployments {
			hc := e.checkOne(dep)
			entries = append(entries, StatusEntry{
				Deployment: dep,
				Health:     hc.Status,
				Detail:     hc.Detail,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return entries, nil
}
