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

// deploymentsForAgent returns only the deployments belonging to the given agent.
// Treats empty Agent field as "claude-code" for backward compatibility.
func deploymentsForAgent(deps []state.Deployment, agentName string) []state.Deployment {
	var result []state.Deployment
	for _, d := range deps {
		if d.Agent == agentName || (d.Agent == "" && agentName == "claude-code") {
			result = append(result, d)
		}
	}
	return result
}

// Check detects deployment health issues (FR-013).
// Returns only unhealthy entries for the active agent. Empty slice means all healthy.
func (e *Engine) Check() ([]state.HealthCheck, error) {
	var issues []state.HealthCheck

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range deploymentsForAgent(st.Deployments, e.agent.Name) {
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

// Prune removes ghost deployments for the active agent whose symlinks no
// longer exist on disk. Returns the number of records pruned. Only ENOENT
// triggers removal; other errors (e.g., EACCES) keep the record.
func (e *Engine) Prune() (int, error) {
	return e.pruneFiltered(true)
}

// PruneAll removes ghost deployments for ALL agents regardless of the
// engine's bound agent. Use this for pre-operation cleanup where stale
// records from any agent should be removed.
func (e *Engine) PruneAll() (int, error) {
	return e.pruneFiltered(false)
}

// pruneFiltered implements prune with optional agent filtering.
func (e *Engine) pruneFiltered(agentOnly bool) (int, error) {
	pruned := 0

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		if len(st.Deployments) == 0 {
			return nil // short-circuit: nothing to prune
		}

		var keep []state.Deployment
		for _, dep := range st.Deployments {
			// Skip deployments that don't belong to this agent when filtering
			if agentOnly && dep.Agent != e.agent.Name && !(dep.Agent == "" && e.agent.Name == "claude-code") {
				keep = append(keep, dep)
				continue
			}
			_, err := e.lstat(dep.LinkPath)
			if err != nil {
				if os.IsNotExist(err) {
					pruned++
					continue // ghost: skip this record
				}
				// Permission error or other: keep the record
			}
			keep = append(keep, dep)
		}

		if pruned == 0 {
			return nil // no changes, skip save
		}

		st.Deployments = keep
		return e.store.Save(st)
	})

	if err != nil {
		return 0, err
	}
	return pruned, nil
}

// Sync repairs detected deployment issues for the active agent (FR-014).
func (e *Engine) Sync() (*SyncResult, error) {
	var result SyncResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		agentDeps := deploymentsForAgent(st.Deployments, e.agent.Name)
		agentSet := make(map[string]bool, len(agentDeps))
		for _, d := range agentDeps {
			agentSet[d.LinkPath] = true
		}

		// Keep non-agent deployments untouched, process only agent's
		var keep []state.Deployment
		for _, dep := range st.Deployments {
			if !agentSet[dep.LinkPath] {
				keep = append(keep, dep)
				continue
			}

			hc := e.checkOne(dep)
			switch hc.Status {
			case state.HealthOK:
				keep = append(keep, dep)

			case state.HealthBroken, state.HealthOrphaned:
				_ = e.remove(dep.LinkPath)
				result.Removed = append(result.Removed, dep)

			case state.HealthMissing:
				if _, err := e.stat(dep.SourcePath); err == nil {
					_ = e.mkdirAll(filepath.Dir(dep.LinkPath), 0o755)
					if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
						result.Repaired = append(result.Repaired, dep)
						keep = append(keep, dep)
					} else {
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Failed to re-create %s: %v", dep.LinkPath, err))
						result.Removed = append(result.Removed, dep)
					}
				} else {
					result.Removed = append(result.Removed, dep)
				}

			case state.HealthDrifted:
				_ = e.remove(dep.LinkPath)
				if err := e.symlink(dep.SourcePath, dep.LinkPath); err == nil {
					result.Repaired = append(result.Repaired, dep)
					keep = append(keep, dep)
				} else {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Failed to repair %s: %v", dep.LinkPath, err))
					keep = append(keep, dep)
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

// Status returns deployments for the active agent with their health status (FR-015).
// Returns a flat list; grouping by type is the caller's responsibility.
func (e *Engine) Status() ([]StatusEntry, error) {
	var entries []StatusEntry

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for _, dep := range deploymentsForAgent(st.Deployments, e.agent.Name) {
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
