package agent_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/config"
)

func stubRegistry(cfg config.Config, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error)) *agent.Registry {
	r := agent.New(cfg)
	r.SetLookPath(lookPath)
	r.SetStat(stat)
	return r
}

func lookPathFound(file string) (string, error) {
	return "/usr/local/bin/" + file, nil
}

func lookPathNotFound(file string) (string, error) {
	return "", exec.ErrNotFound
}

func statFound(path string) (os.FileInfo, error) {
	return nil, nil
}

func statNotFound(path string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

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

func TestDetectPathAndDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	result := r.Detect()
	if len(result.Agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(result.Agents))
	}
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true")
	}
	if !result.Agents[0].InPath {
		t.Error("expected InPath=true")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestDetectPathNoDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statNotFound)
	result := r.Detect()
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true when in PATH")
	}
	if !result.Agents[0].InPath {
		t.Error("expected InPath=true")
	}
}

func TestDetectNoPATHWithDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathNotFound, statFound)
	result := r.Detect()
	if !result.Agents[0].Detected {
		t.Error("expected Detected=true when dir exists")
	}
	if result.Agents[0].InPath {
		t.Error("expected InPath=false")
	}
}

func TestDetectNoPATHNoDir(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathNotFound, statNotFound)
	result := r.Detect()
	if result.Agents[0].Detected {
		t.Error("expected Detected=false")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning when no agents detected")
	}
}

func TestDetectIsIdempotent(t *testing.T) {
	callCount := 0
	countingLookPath := func(file string) (string, error) {
		callCount++
		return "/usr/local/bin/" + file, nil
	}
	r := stubRegistry(config.Config{}, countingLookPath, statFound)

	r.Detect()
	r.Detect()
	r.Detect()

	if callCount != 1 {
		t.Errorf("lookPath called %d times, want 1 (idempotent)", callCount)
	}
}
