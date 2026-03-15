package asset_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/asset"
)

func TestContextMetaValidateEmpty(t *testing.T) {
	m := asset.ContextMeta{}
	if err := m.Validate(); err == nil {
		t.Error("expected error for empty description")
	}
}

func TestContextMetaValidateValid(t *testing.T) {
	m := asset.ContextMeta{Description: "Go project rules"}
	if err := m.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextMetaYAMLRoundTrip(t *testing.T) {
	meta := asset.ContextMeta{
		Description:    "Go project rules",
		Tags:           []string{"go", "backend"},
		TargetLanguage: "go",
	}
	data, err := yaml.Marshal(&meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got asset.ContextMeta
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Description != "Go project rules" {
		t.Errorf("description: got %q", got.Description)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags: got %d", len(got.Tags))
	}
}
