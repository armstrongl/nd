package deploy_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/deploy"
)

func TestBulkResultHasFailures(t *testing.T) {
	br := deploy.BulkResult{Succeeded: 3, Failed: 0}
	if br.HasFailures() {
		t.Error("should not have failures")
	}
	br.Failed = 1
	if !br.HasFailures() {
		t.Error("should have failures")
	}
}

func TestBulkResultFailedResults(t *testing.T) {
	br := deploy.BulkResult{
		Results: []deploy.Result{
			{Success: true, Action: deploy.ActionCreated},
			{Success: false, Action: deploy.ActionFailed, ErrorMsg: "permission denied"},
			{Success: true, Action: deploy.ActionCreated},
		},
		Succeeded: 2,
		Failed:    1,
	}
	failed := br.FailedResults()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed, got %d", len(failed))
	}
	if failed[0].ErrorMsg != "permission denied" {
		t.Errorf("error: got %q", failed[0].ErrorMsg)
	}
}
