package state_test

import (
	"testing"
	"time"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

func sampleState() state.DeploymentState {
	return state.DeploymentState{
		Version: 1,
		Deployments: []state.Deployment{
			{SourceID: "s1", AssetType: nd.AssetSkill, AssetName: "a",
				Scope: nd.ScopeGlobal, Origin: nd.OriginManual, DeployedAt: time.Now()},
			{SourceID: "s1", AssetType: nd.AssetAgent, AssetName: "b",
				Scope: nd.ScopeProject, ProjectPath: "/proj", Origin: nd.OriginProfile("go"), DeployedAt: time.Now()},
			{SourceID: "s2", AssetType: nd.AssetSkill, AssetName: "c",
				Scope: nd.ScopeGlobal, Origin: nd.OriginPinned, DeployedAt: time.Now()},
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
