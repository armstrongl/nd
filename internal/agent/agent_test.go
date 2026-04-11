package agent_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/nd"
)

func claudeCode() agent.Agent {
	return agent.Agent{
		Name:               "claude-code",
		GlobalDir:          "/Users/dev/.claude",
		ProjectDir:         ".claude",
		SourceAlias:        "claude",
		Binary:             "claude",
		SupportedTypes:     nd.DeployableAssetTypes(),
		DefaultContextFile: "",
		ContextInProjectDir: false,
		VersionPattern:     `(?i)claude`,
		Detected:           true,
		InPath:             true,
	}
}

func copilot() agent.Agent {
	return agent.Agent{
		Name:               "copilot",
		GlobalDir:          "/Users/dev/.copilot",
		ProjectDir:         ".github",
		SourceAlias:        "copilot",
		Binary:             "copilot",
		SupportedTypes:     []nd.AssetType{nd.AssetSkill, nd.AssetAgent, nd.AssetContext},
		DefaultContextFile: "copilot-instructions.md",
		ContextInProjectDir: true,
		VersionPattern:     `(?i)copilot|github\.copilot`,
		Detected:           true,
		InPath:             true,
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

// --- SupportsType tests ---

func TestSupportsTypeClaudeCodeAllDeployable(t *testing.T) {
	a := claudeCode()
	for _, at := range nd.DeployableAssetTypes() {
		if !a.SupportsType(at) {
			t.Errorf("claude-code should support %q", at)
		}
	}
}

func TestSupportsTypeCopilotAccepts(t *testing.T) {
	a := copilot()
	accepted := []nd.AssetType{nd.AssetSkill, nd.AssetAgent, nd.AssetContext}
	for _, at := range accepted {
		if !a.SupportsType(at) {
			t.Errorf("copilot should support %q", at)
		}
	}
}

func TestSupportsTypeCopilotRejects(t *testing.T) {
	a := copilot()
	rejected := []nd.AssetType{nd.AssetCommand, nd.AssetOutputStyle, nd.AssetRule, nd.AssetHook}
	for _, at := range rejected {
		if a.SupportsType(at) {
			t.Errorf("copilot should NOT support %q", at)
		}
	}
}

func TestSupportsTypeEmptySliceRejectsAll(t *testing.T) {
	a := agent.Agent{Name: "empty"}
	for _, at := range nd.AllAssetTypes() {
		if a.SupportsType(at) {
			t.Errorf("agent with empty SupportedTypes should not support %q", at)
		}
	}
}

// --- Copilot deploy path tests ---

func TestDeployPathCopilotContextProjectInsideProjectDir(t *testing.T) {
	a := copilot()
	got, err := a.DeployPath(nd.AssetContext, "rules", nd.ScopeProject, "/Users/dev/myapp", "copilot-instructions.md")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/myapp/.github/copilot-instructions.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathCopilotContextGlobal(t *testing.T) {
	a := copilot()
	got, err := a.DeployPath(nd.AssetContext, "rules", nd.ScopeGlobal, "", "copilot-instructions.md")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.copilot/copilot-instructions.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathCopilotSkillGlobal(t *testing.T) {
	a := copilot()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeGlobal, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/.copilot/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeployPathCopilotSkillProject(t *testing.T) {
	a := copilot()
	got, err := a.DeployPath(nd.AssetSkill, "review", nd.ScopeProject, "/Users/dev/myapp", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "/Users/dev/myapp/.github/skills/review"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
