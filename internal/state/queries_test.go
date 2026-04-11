package state_test

import (
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

func sampleState() state.DeploymentState {
	return state.DeploymentState{
		Version: 2,
		Deployments: []state.Deployment{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a",
				Agent: "claude-code", Scope: nd.ScopeGlobal, Origin: nd.OriginManual, DeployedAt: time.Now()},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "b",
				Agent: "claude-code", Scope: nd.ScopeProject, ProjectPath: "/proj", Origin: nd.OriginProfile("go"), DeployedAt: time.Now()},
			{SourceID: "s2", AssetType: nd.AssetSkill, AssetName: "c",
				Agent: "copilot", Scope: nd.ScopeGlobal, Origin: nd.OriginPinned, DeployedAt: time.Now()},
		},
	}
}

func TestFindByIdentity(t *testing.T) {
	s := sampleState()
	got := s.FindByIdentity(asset.Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "a"})
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
}

func TestFindByScope(t *testing.T) {
	s := sampleState()
	globals := s.FindByScope(nd.ScopeGlobal)
	if len(globals) != 2 {
		t.Errorf("expected 2 global, got %d", len(globals))
	}
}

func TestFindByOrigin(t *testing.T) {
	s := sampleState()
	pinned := s.FindByOrigin(nd.OriginPinned)
	if len(pinned) != 1 {
		t.Errorf("expected 1 pinned, got %d", len(pinned))
	}
}

func TestFindByProject(t *testing.T) {
	s := sampleState()
	proj := s.FindByProject("/proj")
	if len(proj) != 1 {
		t.Errorf("expected 1, got %d", len(proj))
	}
}

func TestFindByAgent(t *testing.T) {
	s := sampleState()
	claude := s.FindByAgent("claude-code")
	if len(claude) != 2 {
		t.Errorf("expected 2 claude-code deployments, got %d", len(claude))
	}
	copilot := s.FindByAgent("copilot")
	if len(copilot) != 1 {
		t.Errorf("expected 1 copilot deployment, got %d", len(copilot))
	}
}

func TestFindByAgentEmptyTreatsAsClaudeCode(t *testing.T) {
	// Simulate a partially migrated record with empty Agent
	s := state.DeploymentState{
		Version: 2,
		Deployments: []state.Deployment{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "old",
				Agent: "", Scope: nd.ScopeGlobal},
		},
	}
	claude := s.FindByAgent("claude-code")
	if len(claude) != 1 {
		t.Errorf("empty Agent should match claude-code, got %d results", len(claude))
	}
	copilot := s.FindByAgent("copilot")
	if len(copilot) != 0 {
		t.Errorf("empty Agent should NOT match copilot, got %d results", len(copilot))
	}
}

func TestFindByAgentNoMatches(t *testing.T) {
	s := sampleState()
	got := s.FindByAgent("windsurf")
	if len(got) != 0 {
		t.Errorf("expected 0 for unknown agent, got %d", len(got))
	}
}
