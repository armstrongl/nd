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
// Missing file returns empty state. Corrupt YAML renames the file and returns empty state with a warning.
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

// handleCorrupt renames a corrupt state file and returns empty state with warning.
func (s *Store) handleCorrupt(_ error) (*DeploymentState, []string, error) {
	ts := time.Now().Format("2006-01-02T15-04-05")
	corruptPath := fmt.Sprintf("%s.corrupt.%s", s.path, ts)
	os.Rename(s.path, corruptPath)

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
