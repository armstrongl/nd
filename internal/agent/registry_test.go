package agent_test

import (
	"strings"
	"testing"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/config"
)

func TestNewRegistryHasClaudeCode(t *testing.T) {
	cfg := config.Config{}
	r := agent.New(cfg)
	agents := r.All()
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	if agents[0].Name != "claude-code" {
		t.Errorf("got name %q, want %q", agents[0].Name, "claude-code")
	}
}

func TestNewRegistryAppliesGlobalDirOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: "/custom/global"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].GlobalDir != "/custom/global" {
		t.Errorf("got GlobalDir %q, want %q", agents[0].GlobalDir, "/custom/global")
	}
}

func TestNewRegistryAppliesProjectDirOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", ProjectDir: ".custom-claude"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].ProjectDir != ".custom-claude" {
		t.Errorf("got ProjectDir %q, want %q", agents[0].ProjectDir, ".custom-claude")
	}
}

func TestNewRegistryExpandsTildeInOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: "~/custom-claude"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if strings.HasPrefix(agents[0].GlobalDir, "~") {
		t.Errorf("tilde not expanded: got %q", agents[0].GlobalDir)
	}
	if !strings.HasSuffix(agents[0].GlobalDir, "/custom-claude") {
		t.Errorf("got GlobalDir %q, want suffix %q", agents[0].GlobalDir, "/custom-claude")
	}
}

func TestNewRegistryIgnoresUnknownAgentOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "unknown-agent", GlobalDir: "/somewhere"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if !strings.HasSuffix(agents[0].GlobalDir, ".claude") {
		t.Errorf("expected default GlobalDir, got %q", agents[0].GlobalDir)
	}
}
