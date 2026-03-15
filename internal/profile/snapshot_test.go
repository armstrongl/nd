package profile_test

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/profile"
)

func TestSnapshotYAMLRoundTrip(t *testing.T) {
	s := profile.Snapshot{
		Version:   1,
		Name:      "before-switch",
		CreatedAt: time.Now().Truncate(time.Second),
		Auto:      true,
		Deployments: []profile.SnapshotEntry{
			{
				SourceID:   "src",
				AssetType:  nd.AssetSkill,
				AssetName:  "review",
				SourcePath: "/a/b",
				LinkPath:   "/c/d",
				Scope:      nd.ScopeGlobal,
				Origin:     nd.OriginManual,
				DeployedAt: time.Now().Truncate(time.Second),
			},
		},
	}
	data, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	var got profile.Snapshot
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Auto {
		t.Error("auto should be true")
	}
	if len(got.Deployments) != 1 {
		t.Fatalf("deployments: got %d", len(got.Deployments))
	}
	if got.Deployments[0].DeployedAt.IsZero() {
		t.Error("deployed_at should be preserved in snapshot entries")
	}
}

func TestSnapshotValidateRejectsPlugins(t *testing.T) {
	s := profile.Snapshot{
		Version: 1,
		Name:    "bad",
		Deployments: []profile.SnapshotEntry{
			{AssetType: nd.AssetPlugin, AssetName: "p"},
		},
	}
	errs := s.Validate()
	if len(errs) == 0 {
		t.Error("should reject plugin assets in snapshots")
	}
}
