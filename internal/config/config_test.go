package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
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

func TestConfigDefaultDeployAgentsRoundTrip(t *testing.T) {
	c := config.Config{
		Version:             1,
		DefaultScope:        nd.ScopeGlobal,
		DefaultAgent:        "claude-code",
		DefaultDeployAgents: []string{"claude-code", "copilot"},
		SymlinkStrategy:     nd.SymlinkAbsolute,
		Sources:             []config.SourceEntry{},
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), "default_deploy_agents") {
		t.Errorf("expected default_deploy_agents in YAML, got:\n%s", data)
	}

	var got config.Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.DefaultDeployAgents) != 2 ||
		got.DefaultDeployAgents[0] != "claude-code" ||
		got.DefaultDeployAgents[1] != "copilot" {
		t.Errorf("default_deploy_agents round-trip: got %v", got.DefaultDeployAgents)
	}
}

func TestConfigDefaultDeployAgentsOmittedWhenEmpty(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources:         []config.SourceEntry{},
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if contains(string(data), "default_deploy_agents") {
		t.Errorf("empty default_deploy_agents should be omitted from YAML, got:\n%s", data)
	}
}

func TestConfigValidateDefaultDeployAgentsValid(t *testing.T) {
	c := config.Config{
		Version:             1,
		DefaultScope:        nd.ScopeGlobal,
		DefaultAgent:        "claude-code",
		SymlinkStrategy:     nd.SymlinkAbsolute,
		DefaultDeployAgents: []string{"claude-code", "copilot"},
	}
	errs := c.Validate()
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid default_deploy_agents, got %v", errs)
	}
}

func TestConfigValidateDefaultDeployAgentsUnknown(t *testing.T) {
	c := config.Config{
		Version:             1,
		DefaultScope:        nd.ScopeGlobal,
		DefaultAgent:        "claude-code",
		SymlinkStrategy:     nd.SymlinkAbsolute,
		DefaultDeployAgents: []string{"claude-code", "bogus"},
	}
	errs := c.Validate()
	found := false
	for _, e := range errs {
		if e.Field == "default_deploy_agents" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a ValidationError with Field 'default_deploy_agents', got %v", errs)
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

func TestConfigValidateEmptySourceID(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Sources: []config.SourceEntry{
			{ID: "", Type: nd.SourceLocal, Path: "/a"},
		},
	}
	errs := c.Validate()
	var found bool
	for _, e := range errs {
		if e.Field == "sources[0].id" && e.Message == "must not be empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a sources[0].id must-not-be-empty error, got %v", errs)
	}
}

func TestValidationErrorNoFile(t *testing.T) {
	ve := config.ValidationError{Field: "version", Message: "must be >= 1"}
	if got := ve.Error(); got != "field version: must be >= 1" {
		t.Errorf("no-file Error(): got %q", got)
	}
}

func TestValidationErrorWithFile(t *testing.T) {
	ve := config.ValidationError{
		File: "config.yaml", Line: 5, Field: "sources[0].path", Message: "path does not exist",
	}
	want := "config.yaml:5: field sources[0].path: path does not exist"
	if got := ve.Error(); got != want {
		t.Errorf("with-file Error(): got %q, want %q", got, want)
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

func TestConfigRecencyDaysYAML(t *testing.T) {
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		RecencyDays:     14,
	}

	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(data), "recency_days") {
		t.Errorf("set recency_days should appear in YAML, got:\n%s", data)
	}

	var got config.Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.RecencyDays != 14 {
		t.Errorf("recency_days round-trip: got %d, want 14", got.RecencyDays)
	}

	// A newly added field must not trip validation.
	for _, e := range got.Validate() {
		if e.Field == "recency_days" {
			t.Errorf("recency_days should not be validated, got error: %v", e)
		}
	}
}

func TestConfigRecencyDaysOmitEmpty(t *testing.T) {
	// Unset (zero) recency_days should be omitted from YAML.
	c := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
	}
	data, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if contains(string(data), "recency_days") {
		t.Error("unset recency_days should be omitted from YAML")
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
