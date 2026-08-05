package profile_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
	"github.com/armstrongl/nd/internal/state"
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

// TestStoreCreateProfileConcurrent is a regression test for the TOCTOU race
// where two concurrent CreateProfile calls with the same name could both pass
// the existence check and both write, silently overwriting one another. With
// the file lock in place, exactly one call must succeed and every other must
// fail with an "already exists" error, leaving one intact file on disk.
func TestStoreCreateProfileConcurrent(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	p := profile.Profile{
		Version:   nd.SchemaVersion,
		Name:      "race",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal},
		},
	}

	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			err := store.CreateProfile(p)
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()

	nilCount := 0
	for _, err := range errs {
		if err == nil {
			nilCount++
			continue
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error from losing writer, got: %v", err)
		}
	}
	if nilCount != 1 {
		t.Fatalf("expected exactly 1 successful CreateProfile, got %d (of %d)", nilCount, n)
	}

	// The single persisted profile must be intact and readable.
	got, err := store.GetProfile("race")
	if err != nil {
		t.Fatalf("GetProfile after race: %v", err)
	}
	if got.Name != "race" || len(got.Assets) != 1 {
		t.Errorf("persisted profile corrupted: name=%q assets=%d", got.Name, len(got.Assets))
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
	_ = store.CreateProfile(p)

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
	_ = store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "alpha", CreatedAt: now, UpdatedAt: now})
	_ = store.CreateProfile(profile.Profile{
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
	} else if beta.AssetCount != 2 {
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
	_ = store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "doomed", CreatedAt: now, UpdatedAt: now})

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
	_ = store.CreateProfile(profile.Profile{Version: nd.SchemaVersion, Name: "evolving", CreatedAt: now, UpdatedAt: now})

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
	_ = store.SaveSnapshot(snap)
	if err := store.SaveSnapshot(snap); err == nil {
		t.Error("should reject duplicate snapshot name")
	}
}

// TestStoreSaveSnapshotConcurrent is a regression test for the TOCTOU race in
// SaveSnapshot: concurrent same-name saves must not both write. Exactly one
// call succeeds; every other returns an "already exists" error.
func TestStoreSaveSnapshotConcurrent(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	snap := profile.Snapshot{
		Version:   nd.SchemaVersion,
		Name:      "race-snap",
		CreatedAt: time.Now().Truncate(time.Second),
		Deployments: []profile.SnapshotEntry{
			{
				SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review",
				SourcePath: "/a/b", LinkPath: "/c/d", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: time.Now().Truncate(time.Second),
			},
		},
	}

	const n = 8
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			err := store.SaveSnapshot(snap)
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}()
	}
	wg.Wait()

	nilCount := 0
	for _, err := range errs {
		if err == nil {
			nilCount++
			continue
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error from losing writer, got: %v", err)
		}
	}
	if nilCount != 1 {
		t.Fatalf("expected exactly 1 successful SaveSnapshot, got %d (of %d)", nilCount, n)
	}

	// The single persisted snapshot must be intact and readable.
	got, err := store.GetSnapshot("race-snap", false)
	if err != nil {
		t.Fatalf("GetSnapshot after race: %v", err)
	}
	if len(got.Deployments) != 1 {
		t.Errorf("persisted snapshot corrupted: deployments=%d, want 1", len(got.Deployments))
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
	_ = store.SaveSnapshot(profile.Snapshot{
		Version: nd.SchemaVersion, Name: "snap-a", CreatedAt: now,
		Deployments: []profile.SnapshotEntry{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x",
				SourcePath: "/a", LinkPath: "/b", Scope: nd.ScopeGlobal,
				Origin: nd.OriginManual, DeployedAt: now},
		},
	})
	_ = store.SaveSnapshot(profile.Snapshot{
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
	_ = store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "user-snap", CreatedAt: now})
	_ = store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "auto-20260315T140000", CreatedAt: now, Auto: true})

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

	_ = store.SaveSnapshot(profile.Snapshot{Version: nd.SchemaVersion, Name: "doomed", CreatedAt: time.Now()})
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

// --- Auto-snapshot and pruning tests ---

func TestStoreAutoSnapshot(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	deps := []profile.SnapshotEntry{
		{
			SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/a/b", LinkPath: "/c/d", Scope: nd.ScopeGlobal,
			Origin: nd.OriginManual, DeployedAt: time.Now().Truncate(time.Second),
		},
	}

	snap, err := store.AutoSnapshot(deps)
	if err != nil {
		t.Fatalf("AutoSnapshot: %v", err)
	}
	if !snap.Auto {
		t.Error("auto should be true")
	}
	if len(snap.Deployments) != 1 {
		t.Errorf("deployments: got %d", len(snap.Deployments))
	}

	// Should be retrievable as auto
	got, err := store.GetSnapshot(snap.Name, true)
	if err != nil {
		t.Fatalf("GetSnapshot auto: %v", err)
	}
	if got.Name != snap.Name {
		t.Errorf("name mismatch: %q vs %q", got.Name, snap.Name)
	}
}

func TestStoreAutoSnapshotEmptyDeployments(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	snap, err := store.AutoSnapshot(nil)
	if err != nil {
		t.Fatalf("AutoSnapshot with nil: %v", err)
	}
	if len(snap.Deployments) != 0 {
		t.Errorf("deployments: got %d", len(snap.Deployments))
	}
}

func TestStorePruneAutoSnapshots(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	// Create 7 auto-snapshots with distinct names
	for i := 0; i < 7; i++ {
		name := fmt.Sprintf("auto-20260315T14%02d00", i)
		snap := profile.Snapshot{
			Version: nd.SchemaVersion, Name: name,
			CreatedAt: time.Now().Truncate(time.Second), Auto: true,
		}
		if err := store.SaveSnapshot(snap); err != nil {
			t.Fatalf("save auto snapshot %d: %v", i, err)
		}
	}

	if err := store.PruneAutoSnapshots(5); err != nil {
		t.Fatalf("PruneAutoSnapshots: %v", err)
	}

	// List auto snapshots via directory
	entries, _ := os.ReadDir(filepath.Join(snapshotsDir, "auto"))
	if len(entries) != 5 {
		t.Errorf("expected 5 auto snapshots after prune, got %d", len(entries))
	}
}

func TestStorePruneAutoSnapshotsNoOp(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	// Only 2 auto snapshots, keep=5 => no pruning
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("auto-20260315T14%02d00", i)
		_ = store.SaveSnapshot(profile.Snapshot{
			Version: nd.SchemaVersion, Name: name,
			CreatedAt: time.Now(), Auto: true,
		})
	}

	if err := store.PruneAutoSnapshots(5); err != nil {
		t.Fatalf("PruneAutoSnapshots: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(snapshotsDir, "auto"))
	if len(entries) != 2 {
		t.Errorf("expected 2 auto snapshots, got %d", len(entries))
	}
}

func TestStoreAutoSaveImplementsSnapshotSaver(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	deps := []state.Deployment{
		{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "review",
			SourcePath: "/a/b", LinkPath: "/c/d", Scope: nd.ScopeGlobal,
			Origin: nd.OriginManual, DeployedAt: time.Now().Truncate(time.Second)},
	}

	if err := store.AutoSave(deps); err != nil {
		t.Fatalf("AutoSave: %v", err)
	}

	// Should have created an auto-snapshot
	summaries, _ := store.ListSnapshots()
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

func TestStoreAutoSavePrunes(t *testing.T) {
	profilesDir, snapshotsDir := tempDirs(t)
	store := profile.NewStore(profilesDir, snapshotsDir)

	// Create 6 auto snapshots directly
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("auto-20260315T14%02d00", i)
		_ = store.SaveSnapshot(profile.Snapshot{
			Version: nd.SchemaVersion, Name: name,
			CreatedAt: time.Now().Truncate(time.Second), Auto: true,
		})
	}

	// AutoSave creates one more and should prune to 5
	_ = store.AutoSave(nil)

	entries, _ := os.ReadDir(filepath.Join(snapshotsDir, "auto"))
	if len(entries) != 5 {
		t.Errorf("expected 5 auto snapshots after prune, got %d", len(entries))
	}
}
