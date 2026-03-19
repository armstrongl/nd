package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// StateStore abstracts state persistence for testing.
type StateStore interface {
	Load() (*state.DeploymentState, []string, error)
	Save(st *state.DeploymentState) error
	WithLock(fn func() error) error
}

// SnapshotSaver creates auto-snapshots before destructive bulk operations (FR-029a).
// Implementations should be best-effort: a snapshot failure must not block the operation.
type SnapshotSaver interface {
	AutoSave(deployments []state.Deployment) error
}

// Engine orchestrates symlink deployment, removal, health checks, and repair.
type Engine struct {
	store         StateStore
	agent         *agent.Agent
	backupDir     string
	snapshotSaver SnapshotSaver

	// Injected for testing (default to os.*)
	symlink  func(oldname, newname string) error
	readlink func(name string) (string, error)
	lstat    func(name string) (os.FileInfo, error)
	stat     func(name string) (os.FileInfo, error)
	remove   func(name string) error
	mkdirAll func(path string, perm os.FileMode) error
	rename   func(oldpath, newpath string) error
	now      func() time.Time
}

// New creates an Engine with default OS functions.
func New(store StateStore, ag *agent.Agent, backupDir string) *Engine {
	return &Engine{
		store:     store,
		agent:     ag,
		backupDir: backupDir,
		symlink:   os.Symlink,
		readlink:  os.Readlink,
		lstat:     os.Lstat,
		stat:      os.Stat,
		remove:    os.Remove,
		mkdirAll:  os.MkdirAll,
		rename:    os.Rename,
		now:       time.Now,
	}
}

// SetSymlink replaces the symlink function (for testing).
func (e *Engine) SetSymlink(fn func(oldname, newname string) error) { e.symlink = fn }

// SetReadlink replaces the readlink function (for testing).
func (e *Engine) SetReadlink(fn func(name string) (string, error)) { e.readlink = fn }

// SetLstat replaces the lstat function (for testing).
func (e *Engine) SetLstat(fn func(name string) (os.FileInfo, error)) { e.lstat = fn }

// SetStat replaces the stat function (for testing).
func (e *Engine) SetStat(fn func(name string) (os.FileInfo, error)) { e.stat = fn }

// SetRemove replaces the remove function (for testing).
func (e *Engine) SetRemove(fn func(name string) error) { e.remove = fn }

// SetMkdirAll replaces the mkdirAll function (for testing).
func (e *Engine) SetMkdirAll(fn func(path string, perm os.FileMode) error) { e.mkdirAll = fn }

// SetRename replaces the rename function (for testing).
func (e *Engine) SetRename(fn func(oldpath, newpath string) error) { e.rename = fn }

// SetNow replaces the time function (for testing).
func (e *Engine) SetNow(fn func() time.Time) { e.now = fn }

// SetSnapshotSaver sets the auto-snapshot saver (optional, nil disables).
func (e *Engine) SetSnapshotSaver(s SnapshotSaver) { e.snapshotSaver = s }

// autoSnapshot calls the snapshot saver if configured. Errors are logged but do not block.
func (e *Engine) autoSnapshot(deployments []state.Deployment) {
	if e.snapshotSaver == nil {
		return
	}
	// Best-effort: ignore errors (FR-029a says auto-snapshots should not block operations)
	_ = e.snapshotSaver.AutoSave(deployments)
}

// DeployRequest describes a single asset deployment.
type DeployRequest struct {
	Asset       asset.Asset
	Scope       nd.Scope
	ProjectRoot string
	Origin      nd.DeployOrigin
	Strategy    nd.SymlinkStrategy
}

// DeployResult describes the outcome of a single deployment.
type DeployResult struct {
	Deployment state.Deployment
	Warnings   []string
	BackedUp   string
}

// DeployError describes a failed deployment within a bulk operation.
type DeployError struct {
	AssetName  string
	AssetType  nd.AssetType
	SourcePath string
	Err        error
}

func (e *DeployError) Error() string {
	return fmt.Sprintf("deploy %s %q from %s: %v", e.AssetType, e.AssetName, e.SourcePath, e.Err)
}

// BulkDeployResult holds outcomes of a bulk deploy operation.
type BulkDeployResult struct {
	Succeeded []DeployResult
	Failed    []DeployError
}

// RemoveRequest describes a single asset removal.
type RemoveRequest struct {
	Identity    asset.Identity
	Scope       nd.Scope
	ProjectRoot string
}

// RemoveError describes a failed removal within a bulk operation.
type RemoveError struct {
	Identity asset.Identity
	Err      error
}

func (e *RemoveError) Error() string {
	return fmt.Sprintf("remove %s %q from %s: %v", e.Identity.Type, e.Identity.Name, e.Identity.SourceID, e.Err)
}

// BulkRemoveResult holds outcomes of a bulk remove operation.
type BulkRemoveResult struct {
	Succeeded []RemoveRequest
	Failed    []RemoveError
}

// Deploy deploys a single asset by creating a symlink (FR-009, FR-011).
func (e *Engine) Deploy(req DeployRequest) (*DeployResult, error) {
	var result *DeployResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		r, err := e.deployOne(req, st)
		if err != nil {
			return err
		}
		result = r

		return e.store.Save(st)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// deployOne performs a single deploy within an existing lock+state context.
func (e *Engine) deployOne(req DeployRequest, st *state.DeploymentState) (*DeployResult, error) {
	if !req.Asset.Type.IsDeployable() {
		return nil, fmt.Errorf("asset type %q is not deployable via symlink; use nd export", req.Asset.Type)
	}

	contextFile := ""
	if req.Asset.Type == nd.AssetContext {
		if req.Asset.ContextFile == nil {
			return nil, fmt.Errorf("context asset %q missing ContextFile info", req.Asset.Name)
		}
		contextFile = req.Asset.ContextFile.FileName
	}

	linkPath, err := e.agent.DeployPath(req.Asset.Type, req.Asset.Name, req.Scope, req.ProjectRoot, contextFile)
	if err != nil {
		return nil, fmt.Errorf("compute deploy path: %w", err)
	}

	var result DeployResult

	backed, warnings, skip, err := e.handleConflict(linkPath, req, st)
	if err != nil {
		return nil, err
	}
	result.Warnings = append(result.Warnings, warnings...)
	result.BackedUp = backed

	if skip {
		// Same asset re-deployed: timestamp already updated in handleConflict.
		// Find the existing deployment to return it.
		for _, d := range st.Deployments {
			if d.LinkPath == linkPath {
				result.Deployment = d
				break
			}
		}
		return &result, nil
	}

	parentDir := filepath.Dir(linkPath)
	if err := e.mkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("permission denied: cannot write to %s: %w", parentDir, err)
	}

	target := req.Asset.SourcePath
	if req.Strategy == nd.SymlinkRelative {
		rel, err := filepath.Rel(filepath.Dir(linkPath), req.Asset.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("compute relative path from %s to %s: %w", linkPath, req.Asset.SourcePath, err)
		}
		target = rel
	}
	if err := e.symlink(target, linkPath); err != nil {
		return nil, fmt.Errorf("create symlink at %s: %w", linkPath, err)
	}

	dep := state.Deployment{
		SourceID:    req.Asset.SourceID,
		AssetType:   req.Asset.Type,
		AssetName:   req.Asset.Name,
		SourcePath:  req.Asset.SourcePath,
		LinkPath:    linkPath,
		Scope:       req.Scope,
		ProjectPath: req.ProjectRoot,
		Origin:      req.Origin,
		Strategy:    req.Strategy,
		DeployedAt:  e.now(),
	}
	st.Deployments = append(st.Deployments, dep)
	result.Deployment = dep

	if req.Asset.Type.RequiresSettingsRegistration() {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Asset %q requires manual registration in settings.json or settings.local.json", req.Asset.Name))
	}

	return &result, nil
}

// handleConflict checks for existing files/symlinks at linkPath and handles them.
// Returns (backedUpPath, warnings, skip, error). If skip is true, the caller
// should not create a new symlink or append a new deployment entry — the
// existing entry was updated in-place (same-asset re-deploy).
func (e *Engine) handleConflict(linkPath string, req DeployRequest, st *state.DeploymentState) (string, []string, bool, error) {
	info, err := e.lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, false, nil // No conflict
		}
		return "", nil, false, fmt.Errorf("check target path %s: %w", linkPath, err)
	}

	// Something exists. Classify it.
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := e.readlink(linkPath)
		if err != nil {
			return "", nil, false, fmt.Errorf("readlink %s: %w", linkPath, err)
		}

		// Check if it's an nd-managed symlink
		for i, d := range st.Deployments {
			if d.LinkPath == linkPath {
				if d.SourcePath == req.Asset.SourcePath {
					// Same asset re-deployed: update timestamp, signal skip
					st.Deployments[i].DeployedAt = e.now()
					st.Deployments[i].Origin = req.Origin
					return "", nil, true, nil
				}
				// Different nd-managed asset: remove old symlink and state entry
				_ = e.remove(linkPath)
				st.Deployments = append(st.Deployments[:i], st.Deployments[i+1:]...)
				return "", nil, false, nil
			}
		}

		// Foreign symlink (not in state)
		if req.Asset.Type == nd.AssetContext {
			backed, w := e.backupAndWarn(linkPath, nd.FileKindForeignSymlink, target)
			return backed, w, false, nil
		}
		return "", nil, false, &nd.ConflictError{
			TargetPath:   linkPath,
			ExistingKind: nd.FileKindForeignSymlink,
			AssetName:    req.Asset.Name,
		}
	}

	// Plain file
	if req.Asset.Type == nd.AssetContext {
		backed, w := e.backupAndWarn(linkPath, nd.FileKindPlainFile, "")
		return backed, w, false, nil
	}
	return "", nil, false, &nd.ConflictError{
		TargetPath:   linkPath,
		ExistingKind: nd.FileKindPlainFile,
		AssetName:    req.Asset.Name,
	}
}

// backupAndWarn backs up an existing file and returns the backup path + warnings.
func (e *Engine) backupAndWarn(linkPath string, kind nd.OriginalFileKind, target string) (string, []string) {
	backed, err := e.backupExistingFile(linkPath)
	var warnings []string
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to back up %s: %v", linkPath, err))
		return "", warnings
	}

	msg := fmt.Sprintf("Backed up existing %s at %s to %s", kind, linkPath, backed)
	if kind == nd.FileKindPlainFile {
		msg = fmt.Sprintf("Backed up existing manually created file at %s to %s", linkPath, backed)
	}
	_ = target // suppress unused warning; target used for foreign symlink display if needed
	warnings = append(warnings, msg)
	return backed, warnings
}

// backupExistingFile moves the file at path to backupDir with a timestamp suffix.
// Retains only the last 5 backups per base filename.
func (e *Engine) backupExistingFile(path string) (string, error) {
	if err := e.mkdirAll(e.backupDir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Base(path)
	ts := e.now().Format("2006-01-02T15-04-05")
	backupName := fmt.Sprintf("%s.%s.bak", base, ts)
	backupPath := filepath.Join(e.backupDir, backupName)

	if err := e.rename(path, backupPath); err != nil {
		return "", err
	}

	// Prune: keep only last 5 backups for this base filename
	e.pruneBackups(base, 5)
	return backupPath, nil
}

// pruneBackups removes old backups exceeding maxKeep for files matching the given base name.
func (e *Engine) pruneBackups(baseName string, maxKeep int) {
	entries, err := os.ReadDir(e.backupDir)
	if err != nil {
		return
	}

	prefix := baseName + "."
	var matching []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			matching = append(matching, entry.Name())
		}
	}

	// ReadDir returns entries sorted by name. Timestamps in filenames sort chronologically.
	if len(matching) > maxKeep {
		for _, name := range matching[:len(matching)-maxKeep] {
			_ = e.remove(filepath.Join(e.backupDir, name))
		}
	}
}

// DeployBulk deploys multiple assets with fail-open behavior (FR-010).
// Acquires lock once, loads state once, saves once at the end.
func (e *Engine) DeployBulk(reqs []DeployRequest) (*BulkDeployResult, error) {
	var result BulkDeployResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		e.autoSnapshot(st.Deployments)

		for _, req := range reqs {
			dr, err := e.deployOne(req, st)
			if err != nil {
				result.Failed = append(result.Failed, DeployError{
					AssetName:  req.Asset.Name,
					AssetType:  req.Asset.Type,
					SourcePath: req.Asset.SourcePath,
					Err:        err,
				})
				continue
			}
			result.Succeeded = append(result.Succeeded, *dr)
		}

		return e.store.Save(st)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Remove removes a single deployed asset (FR-012).
func (e *Engine) Remove(req RemoveRequest) error {
	return e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		if err := e.removeOne(req, st); err != nil {
			return err
		}

		return e.store.Save(st)
	})
}

// RemoveBulk removes multiple deployed assets with fail-open behavior (FR-012).
func (e *Engine) RemoveBulk(reqs []RemoveRequest) (*BulkRemoveResult, error) {
	var result BulkRemoveResult

	err := e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		e.autoSnapshot(st.Deployments)

		for _, req := range reqs {
			if err := e.removeOne(req, st); err != nil {
				result.Failed = append(result.Failed, RemoveError{
					Identity: req.Identity,
					Err:      err,
				})
				continue
			}
			result.Succeeded = append(result.Succeeded, req)
		}

		return e.store.Save(st)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SetOrigin updates the deploy origin for an existing deployment.
// Used by nd pin / nd unpin to change origin between manual and pinned.
func (e *Engine) SetOrigin(identity asset.Identity, scope nd.Scope, projectRoot string, origin nd.DeployOrigin) error {
	return e.store.WithLock(func() error {
		st, _, err := e.store.Load()
		if err != nil {
			return fmt.Errorf("load state: %w", err)
		}

		for i, d := range st.Deployments {
			if d.SourceID == identity.SourceID &&
				d.AssetType == identity.Type &&
				d.AssetName == identity.Name &&
				d.Scope == scope {
				if scope == nd.ScopeProject && d.ProjectPath != projectRoot {
					continue
				}
				st.Deployments[i].Origin = origin
				return e.store.Save(st)
			}
		}

		return fmt.Errorf("deployment not found: %s/%s from %s", identity.Type, identity.Name, identity.SourceID)
	})
}

// removeOne removes a single deployment within an existing lock+state context.
func (e *Engine) removeOne(req RemoveRequest, st *state.DeploymentState) error {
	idx := -1
	for i, d := range st.Deployments {
		if d.SourceID == req.Identity.SourceID &&
			d.AssetType == req.Identity.Type &&
			d.AssetName == req.Identity.Name &&
			d.Scope == req.Scope {
			if req.Scope == nd.ScopeProject && d.ProjectPath != req.ProjectRoot {
				continue
			}
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("deployment not found: %s/%s from %s", req.Identity.Type, req.Identity.Name, req.Identity.SourceID)
	}

	err := e.remove(st.Deployments[idx].LinkPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove symlink %s: %w", st.Deployments[idx].LinkPath, err)
	}

	st.Deployments = append(st.Deployments[:idx], st.Deployments[idx+1:]...)
	return nil
}
