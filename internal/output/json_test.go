package output_test

import (
	"encoding/json"
	"testing"

	"github.com/larah/nd/internal/output"
)

func TestJSONResponseOK(t *testing.T) {
	r := output.JSONResponse{
		Status: "ok",
		Data:   map[string]int{"count": 3},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var got output.JSONResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "ok" {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestJSONResponseError(t *testing.T) {
	r := output.JSONResponse{
		Status: "error",
		Errors: []output.JSONError{
			{Code: "INVALID_CONFIG", Message: "bad config", Field: "sources[0].path"},
		},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("should produce non-empty JSON")
	}
}
