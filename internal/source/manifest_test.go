package source_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/source"
)

func TestManifestYAMLRoundTrip(t *testing.T) {
	m := source.Manifest{
		Version: 1,
		Paths: map[nd.AssetType][]string{
			nd.AssetSkill: {"skills/", "go-skills/skills/"},
			nd.AssetAgent: {"agents/"},
		},
		Exclude: []string{"experimental/"},
	}
	data, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	var got source.Manifest
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Paths[nd.AssetSkill]) != 2 {
		t.Errorf("skills paths: got %d", len(got.Paths[nd.AssetSkill]))
	}
}

func TestManifestValidatePathTraversal(t *testing.T) {
	m := source.Manifest{
		Version: 1,
		Paths: map[nd.AssetType][]string{
			nd.AssetSkill: {"../../../etc/"},
		},
	}
	errs := m.Validate("/Users/dev/source")
	if len(errs) == 0 {
		t.Error("should reject path traversal")
	}
}

func TestManifestValidateTooManyPaths(t *testing.T) {
	paths := make([]string, 1001)
	for i := range paths {
		paths[i] = "dir/"
	}
	m := source.Manifest{
		Version: 1,
		Paths:   map[nd.AssetType][]string{nd.AssetSkill: paths},
	}
	errs := m.Validate("/Users/dev/source")
	if len(errs) == 0 {
		t.Error("should reject >1000 path entries")
	}
}
