package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
)

// Store manages the deployment state file on disk.
type Store struct {
	path     string
	lockPath string
}

// NewStore creates a Store targeting the given deployments.yaml path.
func NewStore(path string) *Store {
	return &Store{
		path:     path,
		lockPath: path + ".lock",
	}
}

// Load reads and parses deployments.yaml.
// Missing file returns empty state. Corrupt YAML is quarantined by renaming the
// file, returning empty state with a warning; if that rename fails, Load returns
// a non-nil error so the caller aborts before overwriting the still-present
// original. Structurally-invalid (but parseable) state also returns an error.
// Newer schema version refuses to load (NFR-014).
func (s *Store) Load() (*DeploymentState, []string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DeploymentState{Version: nd.SchemaVersion}, nil, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", s.path, err)
	}

	var st DeploymentState
	if err := yaml.Unmarshal(data, &st); err != nil {
		return s.handleCorrupt(err)
	}

	// Schema version check (NFR-014)
	if st.Version > nd.SchemaVersion {
		return nil, nil, fmt.Errorf(
			"deployments.yaml has schema version %d, but this version of nd only supports version %d; upgrade nd to read this file",
			st.Version, nd.SchemaVersion,
		)
	}
	if st.Version < nd.SchemaVersion {
		s.migrate(&st)
	}

	// Reject structurally-invalid (but parseable) state so callers don't Save()
	// over it and silently accept/propagate corruption.
	if errs := st.Validate(); len(errs) > 0 {
		return nil, nil, fmt.Errorf("deployments.yaml is invalid: %w", errs[0])
	}

	return &st, nil, nil
}

// migrate applies in-memory schema migrations. Does NOT persist to disk —
// the caller's next Save() will write the migrated state. This keeps Load() read-only.
func (s *Store) migrate(st *DeploymentState) {
	// v1 → v2: backfill Agent="claude-code" on all deployments missing an agent.
	if st.Version < 2 {
		for i := range st.Deployments {
			if st.Deployments[i].Agent == "" {
				st.Deployments[i].Agent = "claude-code"
			}
		}
		st.Version = 2
	}
}

// handleCorrupt quarantines a corrupt state file by renaming it to a timestamped
// .corrupt.* path. Only on a successful rename does it return empty state plus a
// warning and a nil error so the caller may continue. If the rename fails the
// corrupt-but-present original must not be clobbered, so it returns a non-nil
// error and no empty state, causing every "st, _, err := Load()" caller to abort
// before Save().
func (s *Store) handleCorrupt(_ error) (*DeploymentState, []string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05")
	corruptPath := fmt.Sprintf("%s.corrupt.%s", s.path, ts)
	if err := os.Rename(s.path, corruptPath); err != nil {
		return nil, nil, fmt.Errorf(
			"deployments.yaml is corrupt and could not be quarantined (rename to %s failed: %w); refusing to continue to avoid overwriting it",
			corruptPath, err,
		)
	}

	warning := fmt.Sprintf(
		"Warning: deployments.yaml was corrupted and has been renamed to %s. Run nd sync to rebuild deployment state from the filesystem.",
		filepath.Base(corruptPath),
	)
	return &DeploymentState{Version: nd.SchemaVersion}, []string{warning}, nil
}

// Save atomically writes the deployment state to disk using nd.AtomicWrite (NFR-010).
func (s *Store) Save(st *DeploymentState) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := yaml.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return nd.AtomicWrite(s.path, data)
}

// WithLock acquires the file lock, runs fn, then releases. Times out after 5s.
func (s *Store) WithLock(fn func() error) error {
	lock := NewFileLock(s.lockPath)
	if err := lock.Acquire(5 * time.Second); err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()
	return fn()
}
