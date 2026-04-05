package state_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

func TestStoreLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	st, warnings, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if st.Version != nd.SchemaVersion {
		t.Errorf("version: got %d, want %d", st.Version, nd.SchemaVersion)
	}
	if len(st.Deployments) != 0 {
		t.Errorf("deployments: got %d, want 0", len(st.Deployments))
	}
}

func TestStoreLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	data := `version: 1
deployments:
  - source_id: src
    asset_type: skills
    asset_name: review
    source_path: /src/skills/review
    link_path: /home/.claude/skills/review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
`
	os.WriteFile(path, []byte(data), 0o644)

	store := state.NewStore(path)
	st, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(st.Deployments) != 1 {
		t.Fatalf("deployments: got %d, want 1", len(st.Deployments))
	}
	if st.Deployments[0].AssetName != "review" {
		t.Errorf("asset_name: got %q", st.Deployments[0].AssetName)
	}
}

func TestStoreLoadCorruptYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	os.WriteFile(path, []byte("{{{{not yaml at all"), 0o644)

	store := state.NewStore(path)
	st, warnings, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Version != nd.SchemaVersion {
		t.Errorf("version: got %d", st.Version)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "corrupted") {
		t.Errorf("warning should mention corruption: %s", warnings[0])
	}

	// Original file should be renamed to .corrupt.<timestamp>
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".corrupt.") {
			found = true
		}
	}
	if !found {
		t.Error("corrupt file should be renamed with .corrupt. suffix")
	}
}

func TestStoreLoadNewerVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	os.WriteFile(path, []byte("version: 999\ndeployments: []\n"), 0o644)

	store := state.NewStore(path)
	_, _, err := store.Load()
	if err == nil {
		t.Fatal("expected error for newer version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error should mention version: %v", err)
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	original := &state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{
				SourceID:   "src",
				AssetType:  nd.AssetSkill,
				AssetName:  "review",
				SourcePath: "/src/skills/review",
				LinkPath:   "/home/.claude/skills/review",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
			},
		},
	}

	if err := store.Save(original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if len(loaded.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(loaded.Deployments))
	}
	if loaded.Deployments[0].AssetName != "review" {
		t.Errorf("asset_name: got %q", loaded.Deployments[0].AssetName)
	}
}

func TestStoreWithLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	called := false
	err := store.WithLock(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock: %v", err)
	}
	if !called {
		t.Error("fn should have been called")
	}
}

func TestStoreWithLockPropagatesError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	sentinel := errors.New("boom")
	err := store.WithLock(func() error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestStoreMigrationV1toV2InMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	// Write v1 state WITHOUT agent field
	v1data := `version: 1
deployments:
  - source_id: src
    asset_type: skills
    asset_name: review
    source_path: /src/skills/review
    link_path: /home/.claude/skills/review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
`
	os.WriteFile(path, []byte(v1data), 0o644)
	store := state.NewStore(path)

	// Load: migration backfills Agent in memory, bumps version to 2
	st, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Version != 2 {
		t.Errorf("expected version 2 in memory, got %d", st.Version)
	}
	if st.Deployments[0].Agent != "claude-code" {
		t.Errorf("expected Agent='claude-code', got %q", st.Deployments[0].Agent)
	}
}

func TestStoreMigrationDoesNotPersistToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	v1data := `version: 1
deployments:
  - source_id: src
    asset_type: skills
    asset_name: review
    source_path: /src/skills/review
    link_path: /home/.claude/skills/review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
`
	os.WriteFile(path, []byte(v1data), 0o644)
	store := state.NewStore(path)

	// First Load: in-memory migration only
	store.Load()

	// Read file directly: should still be v1 (no disk write)
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "version: 2") {
		t.Error("migration should NOT persist to disk in Load()")
	}
}

func TestStoreMigrationPersistsOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")

	v1data := `version: 1
deployments:
  - source_id: src
    asset_type: skills
    asset_name: review
    source_path: /src/skills/review
    link_path: /home/.claude/skills/review
    scope: global
    origin: manual
    deployed_at: "2026-03-10T14:30:00Z"
`
	os.WriteFile(path, []byte(v1data), 0o644)
	store := state.NewStore(path)

	// Load, then Save
	st, _, _ := store.Load()
	store.Save(st)

	// Second Load reads persisted v2
	st2, _, err := store.Load()
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	if st2.Version != 2 {
		t.Errorf("expected version 2 after Save, got %d", st2.Version)
	}
	if st2.Deployments[0].Agent != "claude-code" {
		t.Errorf("expected Agent='claude-code' persisted, got %q", st2.Deployments[0].Agent)
	}
}

func TestStoreV2RoundTripsAgent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployments.yaml")
	store := state.NewStore(path)

	original := &state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{
				SourceID:  "src",
				AssetType: nd.AssetSkill,
				AssetName: "review",
				Agent:     "copilot",
				Scope:     nd.ScopeGlobal,
				Origin:    nd.OriginManual,
			},
		},
	}
	store.Save(original)

	loaded, _, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Deployments[0].Agent != "copilot" {
		t.Errorf("expected Agent='copilot', got %q", loaded.Deployments[0].Agent)
	}
}
