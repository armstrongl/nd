package agent_test

import (
	"testing"

	"github.com/larah/nd/internal/agent"
	"github.com/larah/nd/internal/nd"
)

func claudeCode() agent.Agent {
	return agent.Agent{
		Name:       "claude-code",
		GlobalDir:  "/Users/dev/.claude",
		ProjectDir: ".claude",
		Detected:   true,
		InPath:     true,
	}
}

func TestDeployPathSkillGlobal(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeGlobal, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathSkillProject(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeProject, "/Users/dev/myapp", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/myapp/.claude/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathContextGlobal(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetContext, "go-rules", nd.ScopeGlobal, "", nd.ContextCLAUDE)
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/CLAUDE.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathContextProjectRoot(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetContext, "go-rules", nd.ScopeProject, "/Users/dev/myapp", nd.ContextCLAUDE)
	if err != nil {
		t.Fatal(err)
	}
	// Context files deploy to project root, not inside .claude/
	want := "/Users/dev/myapp/CLAUDE.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathLocalOnlyContextRejectsGlobal(t *testing.T) {
	a := claudeCode()
	_, err := a.DeployPath(nd.AssetContext, "local-rules", nd.ScopeGlobal, "", nd.ContextCLAUDELocal)
	if err == nil {
		t.Error("should reject global scope for .local.md context files")
	}
}

func TestDeployPathAgentFile(t *testing.T) {
	a := claudeCode()
	got, err := a.DeployPath(nd.AssetAgent, "go-specialist.md", nd.ScopeGlobal, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.claude/agents/go-specialist.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
