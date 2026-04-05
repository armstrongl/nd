package agent_test

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/config"
)

func stubRegistry(cfg config.Config, lookPath func(string) (string, error), stat func(string) (os.FileInfo, error)) *agent.Registry {
	r := agent.New(cfg)
	r.SetLookPath(lookPath)
	r.SetStat(stat)
	// Skip binary verification by default in tests to avoid hitting real binaries.
	// Tests that need to verify binary detection set their own runCommand.
	r.SetRunCommand(nil)
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

func TestNewRegistryHasBothAgents(t *testing.T) {
	cfg := config.Config{}
	r := agent.New(cfg)
	agents := r.All()
	if len(agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(agents))
	}
	if agents[0].Name != "claude-code" {
		t.Errorf("agent[0] got name %q, want %q", agents[0].Name, "claude-code")
	}
	if agents[1].Name != "copilot" {
		t.Errorf("agent[1] got name %q, want %q", agents[1].Name, "copilot")
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
	if len(result.Agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(result.Agents))
	}
	if !result.Agents[0].Detected {
		t.Error("expected claude-code Detected=true")
	}
	if !result.Agents[0].InPath {
		t.Error("expected claude-code InPath=true")
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

	// 2 agents × 1 detect cycle = 2 lookPath calls
	if callCount != 2 {
		t.Errorf("lookPath called %d times, want 2 (one per agent, idempotent across cycles)", callCount)
	}
}

func TestGetFoundAgent(t *testing.T) {
	r := agent.New(config.Config{})
	a, err := r.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got name %q, want %q", a.Name, "claude-code")
	}
}

func TestGetUnknownAgent(t *testing.T) {
	r := agent.New(config.Config{})
	_, err := r.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown agent")
	}
}

func TestDefaultReturnsConfiguredAgent(t *testing.T) {
	cfg := config.Config{DefaultAgent: "claude-code"}
	r := stubRegistry(cfg, lookPathFound, statFound)
	r.Detect()
	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}

func TestDefaultFallsBackToFirstDetected(t *testing.T) {
	cfg := config.Config{}
	r := stubRegistry(cfg, lookPathFound, statFound)
	r.Detect()
	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}

func TestDefaultErrorsWhenNoneDetected(t *testing.T) {
	cfg := config.Config{}
	r := stubRegistry(cfg, lookPathNotFound, statNotFound)
	r.Detect()
	_, err := r.Default()
	if err == nil {
		t.Error("expected error when no agents detected")
	}
}

func TestDefaultErrorsWhenConfiguredAgentNotDetected(t *testing.T) {
	cfg := config.Config{DefaultAgent: "claude-code"}
	r := stubRegistry(cfg, lookPathNotFound, statNotFound)
	r.Detect()
	_, err := r.Default()
	if err == nil {
		t.Error("expected error when configured default agent is not detected")
	}
}

func TestDefaultAutoDetectsIfNotCalled(t *testing.T) {
	cfg := config.Config{}
	r := stubRegistry(cfg, lookPathFound, statFound)
	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "claude-code" {
		t.Errorf("got %q, want %q", a.Name, "claude-code")
	}
}

func TestDefaultSourceAlias(t *testing.T) {
	cfg := config.Config{}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].SourceAlias != "claude" {
		t.Errorf("got SourceAlias %q, want %q", agents[0].SourceAlias, "claude")
	}
}

func TestSourceAliasConfigOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", SourceAlias: "custom"},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].SourceAlias != "custom" {
		t.Errorf("got SourceAlias %q, want %q", agents[0].SourceAlias, "custom")
	}
}

func TestSourceAliasEmptyOverrideKeepsDefault(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "claude-code", SourceAlias: ""},
		},
	}
	r := agent.New(cfg)
	agents := r.All()
	if agents[0].SourceAlias != "claude" {
		t.Errorf("got SourceAlias %q, want %q (empty override should preserve default)", agents[0].SourceAlias, "claude")
	}
}

func TestCopilotDefaults(t *testing.T) {
	r := agent.New(config.Config{})
	a, err := r.Get("copilot")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(a.GlobalDir, ".copilot") {
		t.Errorf("got GlobalDir %q, want suffix %q", a.GlobalDir, ".copilot")
	}
	if a.ProjectDir != ".github" {
		t.Errorf("got ProjectDir %q, want %q", a.ProjectDir, ".github")
	}
	if a.SourceAlias != "copilot" {
		t.Errorf("got SourceAlias %q, want %q", a.SourceAlias, "copilot")
	}
	if a.Binary != "copilot" {
		t.Errorf("got Binary %q, want %q", a.Binary, "copilot")
	}
	if a.DefaultContextFile != "copilot-instructions.md" {
		t.Errorf("got DefaultContextFile %q, want %q", a.DefaultContextFile, "copilot-instructions.md")
	}
	if !a.ContextInProjectDir {
		t.Error("expected ContextInProjectDir=true for copilot")
	}
	if a.VersionPattern == "" {
		t.Error("expected non-empty VersionPattern for copilot")
	}
}

func TestCopilotConfigOverride(t *testing.T) {
	cfg := config.Config{
		Agents: []config.AgentOverride{
			{Name: "copilot", GlobalDir: "/custom/copilot", ProjectDir: ".custom-gh"},
		},
	}
	r := agent.New(cfg)
	a, err := r.Get("copilot")
	if err != nil {
		t.Fatal(err)
	}
	if a.GlobalDir != "/custom/copilot" {
		t.Errorf("got GlobalDir %q, want %q", a.GlobalDir, "/custom/copilot")
	}
	if a.ProjectDir != ".custom-gh" {
		t.Errorf("got ProjectDir %q, want %q", a.ProjectDir, ".custom-gh")
	}
}

func TestClaudeCodeDefaults(t *testing.T) {
	r := agent.New(config.Config{})
	a, err := r.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if a.Binary != "claude" {
		t.Errorf("got Binary %q, want %q", a.Binary, "claude")
	}
	if a.DefaultContextFile != "" {
		t.Errorf("got DefaultContextFile %q, want empty (no rename)", a.DefaultContextFile)
	}
	if a.ContextInProjectDir {
		t.Error("expected ContextInProjectDir=false for claude-code")
	}
	if a.VersionPattern == "" {
		t.Error("expected non-empty VersionPattern for claude-code")
	}
}

func TestVersionPatternIsValidRegex(t *testing.T) {
	r := agent.New(config.Config{})
	for _, a := range r.All() {
		if a.VersionPattern == "" {
			t.Errorf("agent %q has empty VersionPattern", a.Name)
			continue
		}
		// VersionPattern must compile as valid regex
		_, err := regexp.Compile(a.VersionPattern)
		if err != nil {
			t.Errorf("agent %q VersionPattern %q is not valid regex: %v", a.Name, a.VersionPattern, err)
		}
	}
}

func TestDetectBothAgents(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	result := r.Detect()
	if len(result.Agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(result.Agents))
	}
	for _, a := range result.Agents {
		if !a.Detected {
			t.Errorf("agent %q should be detected", a.Name)
		}
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestDefaultWithCopilotConfigured(t *testing.T) {
	cfg := config.Config{DefaultAgent: "copilot"}
	r := stubRegistry(cfg, lookPathFound, statFound)
	r.Detect()
	a, err := r.Default()
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "copilot" {
		t.Errorf("got %q, want %q", a.Name, "copilot")
	}
}

// --- Binary verification tests (Unit 4) ---

func TestDetectBinaryVerified(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statNotFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		return []byte("Claude Code CLI v1.2.3"), nil
	})
	result := r.Detect()
	claude := result.Agents[0]
	if !claude.InPath {
		t.Error("expected InPath=true when binary found and version matches")
	}
	if !claude.Detected {
		t.Error("expected Detected=true")
	}
}

func TestDetectBinaryNameCollision(t *testing.T) {
	// Binary exists but version output doesn't match — name collision
	r := stubRegistry(config.Config{}, lookPathFound, statNotFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		return []byte("some-other-tool v3.0"), nil
	})
	result := r.Detect()
	claude := result.Agents[0]
	if claude.InPath {
		t.Error("expected InPath=false when version doesn't match (name collision)")
	}
	// No dir either, so not detected
	if claude.Detected {
		t.Error("expected Detected=false (no match, no dir)")
	}
}

func TestDetectBinaryCollisionFallsBackToDir(t *testing.T) {
	// Binary doesn't match version, but directory exists
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		return []byte("not-the-right-tool"), nil
	})
	result := r.Detect()
	claude := result.Agents[0]
	if claude.InPath {
		t.Error("expected InPath=false when version doesn't match")
	}
	if !claude.Detected {
		t.Error("expected Detected=true (dir exists even though binary doesn't match)")
	}
}

func TestDetectVersionCommandTimeout(t *testing.T) {
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("signal: killed")
	})
	result := r.Detect()
	claude := result.Agents[0]
	if claude.InPath {
		t.Error("expected InPath=false when version command fails")
	}
	if !claude.Detected {
		t.Error("expected Detected=true (dir exists as fallback)")
	}
}

func TestDetectVersionOnStderr(t *testing.T) {
	// Version string appears in combined output (could be stderr)
	r := stubRegistry(config.Config{}, lookPathFound, statNotFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		return []byte("error: something\nGitHub Copilot CLI v2.0\n"), nil
	})
	result := r.Detect()
	copilot := result.Agents[1]
	if !copilot.InPath {
		t.Error("expected copilot InPath=true when version found in combined output")
	}
}

func TestDetectNoRunCommandDefaultsToNoVerification(t *testing.T) {
	// When runCommand is nil (not set), binary found via lookPath means InPath=true (legacy behavior)
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	// Don't call SetRunCommand — nil means skip verification
	result := r.Detect()
	claude := result.Agents[0]
	if !claude.InPath {
		t.Error("expected InPath=true with nil runCommand (legacy behavior)")
	}
}

func TestDetectIdempotentWithRunCommand(t *testing.T) {
	runCount := 0
	r := stubRegistry(config.Config{}, lookPathFound, statFound)
	r.SetRunCommand(func(name string, args ...string) ([]byte, error) {
		runCount++
		return []byte("claude code"), nil
	})
	r.Detect()
	r.Detect()
	r.Detect()
	// 2 agents × 1 detect cycle = 2 runCommand calls
	if runCount != 2 {
		t.Errorf("runCommand called %d times, want 2 (one per agent, idempotent across cycles)", runCount)
	}
}
