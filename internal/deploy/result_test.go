package deploy_test

import (
	"encoding/json"
	"testing"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
)

func TestResultJSONRoundTrip(t *testing.T) {
	r := deploy.Result{
		AssetID:  asset.Identity{SourceID: "src", Type: nd.AssetSkill, Name: "review"},
		Success:  true,
		Action:   deploy.ActionCreated,
		LinkPath: "/Users/dev/.claude/skills/review",
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var got deploy.Result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.AssetID.Name != "review" {
		t.Errorf("asset name: got %q", got.AssetID.Name)
	}
	if !got.Success {
		t.Error("should be success")
	}
}
