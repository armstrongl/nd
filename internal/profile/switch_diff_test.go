package profile_test

import (
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/profile"
)

func TestComputeSwitchDiff(t *testing.T) {
	now := time.Now()
	current := &profile.Profile{
		Version: 1, Name: "current", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "b", Scope: nd.ScopeGlobal},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "c", Scope: nd.ScopeProject},
		},
	}
	target := &profile.Profile{
		Version: 1, Name: "target", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a", Scope: nd.ScopeGlobal},  // keep
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "d", Scope: nd.ScopeGlobal},  // deploy
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "c", Scope: nd.ScopeGlobal},  // different scope = remove + deploy
		},
	}

	diff := profile.ComputeSwitchDiff(current, target)

	if len(diff.Keep) != 1 {
		t.Errorf("keep: expected 1, got %d", len(diff.Keep))
	}
	// b is removed, c@project is removed (scope changed)
	if len(diff.Remove) != 2 {
		t.Errorf("remove: expected 2, got %d", len(diff.Remove))
	}
	// d is deployed, c@global is deployed (scope changed)
	if len(diff.Deploy) != 2 {
		t.Errorf("deploy: expected 2, got %d", len(diff.Deploy))
	}
}

func TestComputeSwitchDiffEmpty(t *testing.T) {
	now := time.Now()
	empty := &profile.Profile{Version: 1, Name: "empty", CreatedAt: now, UpdatedAt: now}
	full := &profile.Profile{
		Version: 1, Name: "full", CreatedAt: now, UpdatedAt: now,
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x", Scope: nd.ScopeGlobal},
		},
	}
	diff := profile.ComputeSwitchDiff(empty, full)
	if len(diff.Keep) != 0 {
		t.Errorf("keep: expected 0, got %d", len(diff.Keep))
	}
	if len(diff.Remove) != 0 {
		t.Errorf("remove: expected 0, got %d", len(diff.Remove))
	}
	if len(diff.Deploy) != 1 {
		t.Errorf("deploy: expected 1, got %d", len(diff.Deploy))
	}
}
