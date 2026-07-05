package state_test

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

func TestDeploymentStateYAMLRoundTrip(t *testing.T) {
	s := state.DeploymentState{
		Version:       1,
		ActiveProfile: "go-backend",
		Deployments: []state.Deployment{
			{
				SourceID:   "my-assets",
				AssetType:  nd.AssetSkill,
				AssetName:  "code-review",
				SourcePath: "/Users/dev/assets/skills/code-review",
				LinkPath:   "/Users/dev/.claude/skills/code-review",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
				DeployedAt: time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC),
			},
		},
	}

	data, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var got state.DeploymentState
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ActiveProfile != "go-backend" {
		t.Errorf("active_profile: got %q", got.ActiveProfile)
	}
	if len(got.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(got.Deployments))
	}
	d := got.Deployments[0]
	if d.AssetName != "code-review" {
		t.Errorf("asset_name: got %q", d.AssetName)
	}
	if d.Scope != nd.ScopeGlobal {
		t.Errorf("scope: got %q", d.Scope)
	}
}

func TestDeploymentIdentity(t *testing.T) {
	d := state.Deployment{
		SourceID:  "src",
		AssetType: nd.AssetSkill,
		AssetName: "review",
	}
	id := d.Identity()
	if id.SourceID != "src" || id.Name != "review" {
		t.Errorf("unexpected identity: %+v", id)
	}
}

func TestDeploymentStateValidateValid(t *testing.T) {
	s := state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
			{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "lint", Scope: nd.ScopeProject, ProjectPath: "/p", Origin: nd.OriginManual},
		},
	}
	if errs := s.Validate(); len(errs) != 0 {
		t.Errorf("expected no errors for valid state, got %v", errs)
	}
}

func TestDeploymentStateValidateDuplicateIdentity(t *testing.T) {
	s := state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
			{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.ScopeGlobal, Origin: nd.OriginManual},
		},
	}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for duplicate identity, got none")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "duplicate identity") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a duplicate-identity error, got %v", errs)
	}
}

func TestDeploymentStateValidateInvalidScope(t *testing.T) {
	s := state.DeploymentState{
		Version: nd.SchemaVersion,
		Deployments: []state.Deployment{
			{SourceID: "src", AssetType: nd.AssetSkill, AssetName: "review", Scope: nd.Scope("bogus"), Origin: nd.OriginManual},
		},
	}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Fatal("expected error for invalid scope, got none")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "invalid scope") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an invalid-scope error, got %v", errs)
	}
}
