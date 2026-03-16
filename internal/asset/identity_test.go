package asset_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

func TestIdentityString(t *testing.T) {
	id := asset.Identity{SourceID: "my-src", Type: nd.AssetSkill, Name: "review"}
	got := id.String()
	want := "my-src:skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIdentityAsMapKey(t *testing.T) {
	id1 := asset.Identity{SourceID: "a", Type: nd.AssetSkill, Name: "x"}
	id2 := asset.Identity{SourceID: "a", Type: nd.AssetSkill, Name: "x"}
	id3 := asset.Identity{SourceID: "b", Type: nd.AssetSkill, Name: "x"}

	m := map[asset.Identity]bool{id1: true}
	if !m[id2] {
		t.Error("identical identities should match as map keys")
	}
	if m[id3] {
		t.Error("different identities should not match")
	}
}
