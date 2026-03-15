package deploy_test

import (
	"os"
	"testing"
	"time"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// mockStore implements deploy.StateStore for testing.
type mockStore struct {
	state    *state.DeploymentState
	saved    *state.DeploymentState
	warnings []string
	loadErr  error
	saveErr  error
	lockErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		state: &state.DeploymentState{Version: nd.SchemaVersion},
	}
}

func (m *mockStore) Load() (*state.DeploymentState, []string, error) {
	if m.loadErr != nil {
		return nil, nil, m.loadErr
	}
	// Return a copy to detect mutations
	cp := *m.state
	cp.Deployments = make([]state.Deployment, len(m.state.Deployments))
	copy(cp.Deployments, m.state.Deployments)
	return &cp, m.warnings, nil
}

func (m *mockStore) Save(st *state.DeploymentState) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = st
	m.state = st
	return nil
}

func (m *mockStore) WithLock(fn func() error) error {
	if m.lockErr != nil {
		return m.lockErr
	}
	return fn()
}

func testAgent() *agent.Agent {
	return &agent.Agent{
		Name:       "claude-code",
		GlobalDir:  "/home/user/.claude",
		ProjectDir: ".claude",
		Detected:   true,
	}
}

// symCall records a symlink creation for test assertions.
type symCall struct {
	oldname, newname string
}

// fakeFileInfo implements os.FileInfo for testing conflict detection.
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string        { return "fake" }
func (f fakeFileInfo) Size() int64         { return 0 }
func (f fakeFileInfo) Mode() os.FileMode   { return f.mode }
func (f fakeFileInfo) ModTime() time.Time  { return time.Time{} }
func (f fakeFileInfo) IsDir() bool         { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any            { return nil }

func TestNewEngine(t *testing.T) {
	store := newMockStore()
	ag := testAgent()
	engine := deploy.New(store, ag, "/tmp/backups")
	if engine == nil {
		t.Fatal("New returned nil")
	}
}
