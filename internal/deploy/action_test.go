package deploy_test

import (
	"encoding/json"
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
)

func TestActionString(t *testing.T) {
	tests := []struct {
		a    deploy.Action
		want string
	}{
		{deploy.ActionCreated, "created"},
		{deploy.ActionRemoved, "removed"},
		{deploy.ActionReplaced, "replaced"},
		{deploy.ActionSkipped, "skipped"},
		{deploy.ActionBackedUp, "backed-up"},
		{deploy.ActionFailed, "failed"},
		{deploy.ActionDryRun, "dry-run"},
		{deploy.Action(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("Action(%d).String() = %q, want %q", tt.a, got, tt.want)
		}
	}
}

func TestActionMarshalJSON(t *testing.T) {
	data, err := json.Marshal(deploy.ActionCreated)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"created"` {
		t.Errorf("got %s", data)
	}
}
