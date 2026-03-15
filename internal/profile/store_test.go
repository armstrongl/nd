package profile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/profile"
	"github.com/larah/nd/internal/state"
)

// Silence unused import warnings for later tests.
var (
	_ = fmt.Sprintf
	_ = state.DeploymentState{}
)

func tempDirs(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "profiles"), filepath.Join(dir, "snapshots")
}

func TestStoreCreateProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{
		Version:   nd.SchemaVersion,
		Name:      "go-backend",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal},
		},
	}
	if err := store.CreateProfile(p); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// File should exist on disk
	path := filepath.Join(profilesDir, "go-backend.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("profile file not created: %v", err)
	}
}

func TestStoreCreateProfileDuplicate(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{Version: nd.SchemaVersion, Name: "dup", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.CreateProfile(p); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProfile(p); err == nil {
		t.Error("should reject duplicate profile name")
	}
}

func TestStoreCreateProfileInvalidName(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{Version: nd.SchemaVersion, Name: "bad name!", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.CreateProfile(p); err == nil {
		t.Error("should reject invalid profile name")
	}
}

func TestStoreCreateProfileRejectsPlugins(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{
		Version: nd.SchemaVersion, Name: "has-plugin",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetPlugin, AssetName: "p", Scope: nd.ScopeGlobal},
		},
	}
	if err := store.CreateProfile(p); err == nil {
		t.Error("should reject profile with plugin assets")
	}
}

func TestStoreGetProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{
		Version: nd.SchemaVersion, Name: "my-profile",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "x", Scope: nd.ScopeGlobal},
		},
	}
	store.CreateProfile(p)

	got, err := store.GetProfile("my-profile")
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.Name != "my-profile" {
		t.Errorf("name: got %q", got.Name)
	}
	if len(got.Assets) != 1 {
		t.Errorf("assets: got %d", len(got.Assets))
	}
}

func TestStoreGetProfileNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	_, err := store.GetProfile("nonexistent")
	if err == nil {
		t.Error("should return error for nonexistent profile")
	}
}

func TestStoreListProfiles(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "alpha", CreatedAt: now, UpdatedAt: now})
	store.CreateProfile(profile.Profile{
		Version: nd.SchemaVersion, Name: "beta", Description: "Beta profile",
		CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x", Scope: nd.ScopeGlobal},
			{SourceID: "s", AssetType: nd.AssetAgent, AssetName: "y", Scope: nd.ScopeGlobal},
		},
	})

	summaries, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(summaries))
	}
	// Find beta
	var beta *profile.ProfileSummary
	for i := range summaries {
		if summaries[i].Name == "beta" {
			beta = &summaries[i]
		}
	}
	if beta == nil {
		t.Fatal("beta not found")
	}
	if beta.AssetCount != 2 {
		t.Errorf("beta asset count: got %d", beta.AssetCount)
	}
	if beta.Description != "Beta profile" {
		t.Errorf("beta description: got %q", beta.Description)
	}
}

func TestStoreListProfilesEmpty(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	summaries, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(summaries))
	}
}

func TestStoreDeleteProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "doomed", CreatedAt: now, UpdatedAt: now})

	if err := store.DeleteProfile("doomed"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	_, err := store.GetProfile("doomed")
	if err == nil {
		t.Error("profile should be deleted")
	}
}

func TestStoreDeleteProfileNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	if err := store.DeleteProfile("ghost"); err == nil {
		t.Error("should error on nonexistent profile")
	}
}

func TestStoreUpdateProfile(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	now := time.Now().Truncate(time.Second)
	store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "evolving", CreatedAt: now, UpdatedAt: now})

	updated := profile.Profile{
		Version: nd.SchemaVersion, Name: "evolving", Description: "now with description",
		CreatedAt: now, UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "new-skill", Scope: nd.ScopeGlobal},
		},
	}
	if err := store.UpdateProfile(updated); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	got, _ := store.GetProfile("evolving")
	if got.Description != "now with description" {
		t.Errorf("description: got %q", got.Description)
	}
	if len(got.Assets) != 1 {
		t.Errorf("assets: got %d", len(got.Assets))
	}
}

func TestStoreUpdateProfileNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{Version: nd.SchemaVersion, Name: "ghost", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.UpdateProfile(p); err == nil {
		t.Error("should error on nonexistent profile")
	}
}

// --- Snapshot tests ---

func TestStoreSaveAndGetSnapshot(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	snap := profile.Snapshot{
		Version:   nd.SchemaVersion,
		Name:      "before-switch",
		CreatedAt: time.Now().Truncate(time.Second),
		Auto:      false,
		Deployments: []profile.SnapshotEntry{
			{
				SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review",
				SourcePath: "/a/b", LinkPath: "/c/d", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: time.Now().Truncate(time.Second),
			},
		},
	}
	if err := store.SaveSnapshot(snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	got, err := store.GetSnapshot("before-switch", false)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if got.Name != "before-switch" {
		t.Errorf("name: got %q", got.Name)
	}
	if len(got.Deployments) != 1 {
		t.Errorf("deployments: got %d", len(got.Deployments))
	}
}

func TestStoreSaveSnapshotDuplicate(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	snap := profile.Snapshot{Version: nd.SchemaVersion, Name: "dup", CreatedAt: time.Now()}
	store.SaveSnapshot(snap)
	if err := store.SaveSnapshot(snap); err == nil {
		t.Error("should reject duplicate snapshot name")
	}
}

func TestStoreGetSnapshotNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	_, err := store.GetSnapshot("nope", false)
	if err == nil {
		t.Error("should error on nonexistent snapshot")
	}
}

func TestStoreSaveSnapshotRejectsPlugins(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	snap := profile.Snapshot{
		Version: nd.SchemaVersion, Name: "bad", CreatedAt: time.Now(),
		Deployments: []profile.SnapshotEntry{
			{AssetType: nd.AssetPlugin, AssetName: "p"},
		},
	}
	if err := store.SaveSnapshot(snap); err == nil {
		t.Error("should reject snapshot with plugin assets")
	}
}

func TestStoreListSnapshots(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	now := time.Now().Truncate(time.Second)
	store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "snap-a", CreatedAt: now,
		Deployments: []profile.SnapshotEntry{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x",
				SourcePath: "/a", LinkPath: "/b", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
		},
	})
	store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "snap-b", CreatedAt: now,
	})

	summaries, err := store.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(summaries))
	}
}

func TestStoreListSnapshotsIncludesAuto(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	now := time.Now().Truncate(time.Second)
	store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "user-snap", CreatedAt: now})
	store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "auto-20260315T140000", CreatedAt: now, Auto: true})

	summaries, err := store.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(summaries))
	}

	autoCount := 0
	for _, s := range summaries {
		if s.Auto {
			autoCount++
		}
	}
	if autoCount != 1 {
		t.Errorf("expected 1 auto snapshot, got %d", autoCount)
	}
}

func TestStoreListSnapshotsEmpty(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	summaries, err := store.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0, got %d", len(summaries))
	}
}

func TestStoreDeleteSnapshot(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "doomed", CreatedAt: time.Now()})
	if err := store.DeleteSnapshot("doomed", false); err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}
	_, err := store.GetSnapshot("doomed", false)
	if err == nil {
		t.Error("snapshot should be deleted")
	}
}

func TestStoreDeleteSnapshotNotFound(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	if err := store.DeleteSnapshot("ghost", false); err == nil {
		t.Error("should error on nonexistent snapshot")
	}
}
