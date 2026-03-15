package deploy

import (
	"fmt"
	"os"
	"time"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// StateStore abstracts state persistence for testing.
type StateStore interface {
	Load() (*state.DeploymentState, []string, error)
	Save(st *state.DeploymentState) error
	WithLock(fn func() error) error
}

// Engine orchestrates symlink deployment, removal, health checks, and repair.
type Engine struct {
	store     StateStore
	agent     *agent.Agent
	backupDir string

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

// DeployRequest describes a single asset deployment.
type DeployRequest struct {
	Asset       asset.Asset
	Scope       nd.Scope
	ProjectRoot string
	Origin      nd.DeployOrigin
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
