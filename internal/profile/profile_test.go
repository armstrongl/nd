package profile_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/profile"
)

func TestProfileYAMLRoundTrip(t *testing.T) {
	p := profile.Profile{
		Version:   1,
		Name:      "go-backend",
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Assets: []profile.ProfileAsset{
			{SourceID: "my-src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal},
		},
	}
	data, err := yaml.Marshal(&p)
	if err != nil {
		t.Fatal(err)
	}
	var got profile.Profile
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "go-backend" {
		t.Errorf("name: got %q", got.Name)
	}
	if len(got.Assets) != 1 {
		t.Fatalf("assets: got %d", len(got.Assets))
	}
}

func TestProfileValidateRejectsPlugins(t *testing.T) {
	p := profile.Profile{
		Version: 1,
		Name:    "bad",
		Assets: []profile.ProfileAsset{
			{SourceID: "s", AssetType: nd.AssetPlugin, AssetName: "p", Scope: nd.ScopeGlobal},
		},
	}
	errs := p.Validate()
	if len(errs) == 0 {
		t.Error("should reject plugin assets in profiles")
	}
}

func TestProfileAssetIdentity(t *testing.T) {
	pa := profile.ProfileAsset{SourceID: "s", AssetType: nd.AssetSkill, AssetName: "x"}
	id := pa.Identity()
	if id.SourceID != "s" || id.Name != "x" {
		t.Errorf("unexpected identity: %+v", id)
	}
}
