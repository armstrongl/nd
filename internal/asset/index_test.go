package asset_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

func makeAsset(source string, at nd.AssetType, name string) asset.Asset {
	return asset.Asset{
		Identity:   asset.Identity{SourceID: source, Type: at, Name: name},
		SourcePath: "/fake/" + name,
		IsDir:      at.IsDirectory(),
	}
}

func TestNewIndex(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "review"),
		makeAsset("src1", nd.AssetAgent, "go-dev"),
		makeAsset("src2", nd.AssetSkill, "deploy"),
	}
	idx := asset.NewIndex(assets)
	if len(idx.All()) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(idx.All()))
	}
}

func TestIndexLookup(t *testing.T) {
	assets := []asset.Asset{makeAsset("src1", nd.AssetSkill, "review")}
	idx := asset.NewIndex(assets)
	got := idx.Lookup(asset.Identity{SourceID: "src1", Type: nd.AssetSkill, Name: "review"})
	if got == nil {
		t.Fatal("expected to find asset")
	} else if got.Name != "review" {
		t.Errorf("got name %q", got.Name)
	}

	missing := idx.Lookup(asset.Identity{SourceID: "nope", Type: nd.AssetSkill, Name: "nope"})
	if missing != nil {
		t.Error("expected nil for missing asset")
	}
}

func TestIndexByType(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "a"),
		makeAsset("src1", nd.AssetSkill, "b"),
		makeAsset("src1", nd.AssetAgent, "c"),
	}
	idx := asset.NewIndex(assets)
	skills := idx.ByType(nd.AssetSkill)
	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
}

func TestIndexBySource(t *testing.T) {
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "a"),
		makeAsset("src2", nd.AssetSkill, "b"),
	}
	idx := asset.NewIndex(assets)
	src1 := idx.BySource("src1")
	if len(src1) != 1 {
		t.Errorf("expected 1, got %d", len(src1))
	}
}

func TestIndexConflictDetection(t *testing.T) {
	// Same (type, name) from two sources: first source wins
	assets := []asset.Asset{
		makeAsset("src1", nd.AssetSkill, "review"),
		makeAsset("src2", nd.AssetSkill, "review"),
	}
	idx := asset.NewIndex(assets)
	conflicts := idx.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Winner != "src1" {
		t.Errorf("winner should be src1, got %q", conflicts[0].Winner)
	}
	if conflicts[0].Loser != "src2" {
		t.Errorf("loser should be src2, got %q", conflicts[0].Loser)
	}
	// Only the winner should be in the index
	all := idx.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 asset after conflict, got %d", len(all))
	}
}
