package doctor_test

import (
	"encoding/json"
	"testing"

	"github.com/armstrongl/nd/internal/doctor"
)

func TestReportJSONRoundTrip(t *testing.T) {
	r := doctor.Report{
		Config: doctor.ConfigCheck{GlobalValid: true, ProjectValid: true},
		Git:    doctor.GitCheck{Available: true, Version: "2.44.0"},
		Summary: doctor.Summary{Pass: 5, Warn: 1, Fail: 0},
	}
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatal(err)
	}
	var got doctor.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Summary.Pass != 5 {
		t.Errorf("pass: got %d", got.Summary.Pass)
	}
	if !got.Git.Available {
		t.Error("git should be available")
	}
}
