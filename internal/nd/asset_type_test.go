package nd_test

import (
	"testing"

	"github.com/larah/nd/internal/nd"
)

func TestAllAssetTypes(t *testing.T) {
	types := nd.AllAssetTypes()
	if len(types) != 8 {
		t.Fatalf("expected 8 asset types, got %d", len(types))
	}
}

func TestDeployableAssetTypes(t *testing.T) {
	types := nd.DeployableAssetTypes()
	for _, at := range types {
		if at == nd.AssetPlugin {
			t.Fatal("plugins should not be in deployable types")
		}
	}
	if len(types) != 7 {
		t.Fatalf("expected 7 deployable types, got %d", len(types))
	}
}

func TestAssetTypeIsDirectory(t *testing.T) {
	tests := []struct {
		at   nd.AssetType
		want bool
	}{
		{nd.AssetSkill, true},
		{nd.AssetPlugin, true},
		{nd.AssetHook, true},
		{nd.AssetAgent, false},
		{nd.AssetCommand, false},
		{nd.AssetOutputStyle, false},
		{nd.AssetRule, false},
		{nd.AssetContext, false},
	}
	for _, tt := range tests {
		if got := tt.at.IsDirectory(); got != tt.want {
			t.Errorf("%s.IsDirectory() = %v, want %v", tt.at, got, tt.want)
		}
	}
}

func TestAssetTypeDeploySubdir(t *testing.T) {
	if nd.AssetContext.DeploySubdir() != "" {
		t.Error("context should return empty deploy subdir")
	}
	if nd.AssetSkill.DeploySubdir() != "skills" {
		t.Errorf("skills should return 'skills', got %q", nd.AssetSkill.DeploySubdir())
	}
}

func TestAssetTypeIsDeployable(t *testing.T) {
	if nd.AssetPlugin.IsDeployable() {
		t.Error("plugins should not be deployable")
	}
	if !nd.AssetSkill.IsDeployable() {
		t.Error("skills should be deployable")
	}
}

func TestAssetTypeRequiresSettingsRegistration(t *testing.T) {
	if !nd.AssetHook.RequiresSettingsRegistration() {
		t.Error("hooks require settings registration")
	}
	if !nd.AssetOutputStyle.RequiresSettingsRegistration() {
		t.Error("output-styles require settings registration")
	}
	if nd.AssetSkill.RequiresSettingsRegistration() {
		t.Error("skills do not require settings registration")
	}
}
