package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
)

func TestConfigYAMLRoundTrip(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "my-assets", Type: nd.SourceLocal, Path: "/Users/dev/assets"},
		},
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got config.Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DefaultScope != nd.ScopeGlobal {
		t.Errorf("scope: got %q, want %q", got.DefaultScope, nd.ScopeGlobal)
	}
	if got.DefaultAgent != "claude-code" {
		t.Errorf("agent: got %q", got.DefaultAgent)
	}
	if len(got.Sources) != 1 || got.Sources[0].ID != "my-assets" {
		t.Errorf("sources: got %+v", got.Sources)
	}
}

func TestProjectConfigPointerSemantics(t *testing.T) {
	// Unset fields should not appear in YAML
	pc := config.ProjectConfig{Version: 1}
	data, err := yaml.Marshal(&pc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if contains(s, "default_scope") {
		t.Error("unset default_scope should not appear in YAML")
	}
	if contains(s, "default_agent") {
		t.Error("unset default_agent should not appear in YAML")
	}
}

func TestValidationErrorImplementsError(t *testing.T) {
	ve := config.ValidationError{
		File: "config.yaml", Line: 5, Field: "sources[0].path", Message: "path does not exist",
	}
	if ve.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
