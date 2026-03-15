package agent_test

import (
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
