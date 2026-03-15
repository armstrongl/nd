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

func TestConfigValidateValid(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources:         []config.SourceEntry{},
	}
	errs := c.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestConfigValidateInvalidVersion(t *testing.T) {
	c := config.Config{Version: 0}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for version 0")
	}
}

func TestConfigValidateInvalidScope(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    "invalid",
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for invalid scope")
	}
}

func TestConfigValidateEmptyAgent(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for empty agent")
	}
}

func TestConfigValidateInvalidSymlinkStrategy(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: "invalid",
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for invalid symlink strategy")
	}
}

func TestConfigValidateFutureVersion(t *testing.T) {
	c := config.Config{
		Version:         99,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for future schema version")
	}
}

func TestConfigValidateDuplicateSourceIDs(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "dup", Type: nd.SourceLocal, Path: "/a"},
			{ID: "dup", Type: nd.SourceLocal, Path: "/b"},
		},
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for duplicate source IDs")
	}
}

func TestConfigValidateSourceMissingPath(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "s1", Type: nd.SourceLocal, Path: ""},
		},
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for empty source path")
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

func TestConfigValidateInvalidSourceType(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "s1", Type: "sftp", Path: "/some/path"},
		},
	}
	errs := c.Validate()
	if len(errs) == 0 {
		t.Error("expected error for invalid source type")
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
