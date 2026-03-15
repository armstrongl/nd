package asset_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/asset"
)

func TestCachedIndexYAMLRoundTrip(t *testing.T) {
	ci := asset.CachedIndex{
		Version:   1,
		SourceID:  "my-src",
		BuiltAt:   time.Now().Truncate(time.Second),
		SourceMod: time.Now().Add(-time.Hour).Truncate(time.Second),
	}
	data, err := yaml.Marshal(&ci)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got asset.CachedIndex
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SourceID != "my-src" {
		t.Errorf("source_id: got %q", got.SourceID)
	}
}

func TestCachedIndexIsStale(t *testing.T) {
	ci := asset.CachedIndex{
		SourceMod: time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC),
	}
	newer := time.Date(2026, 3, 14, 11, 0, 0, 0, time.UTC)
	if !ci.IsStale(newer) {
		t.Error("should be stale when source is newer")
	}
	older := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	if ci.IsStale(older) {
		t.Error("should not be stale when source is older")
	}
}
