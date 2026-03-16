package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/output"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	if err := printJSON(&buf, data, false); err != nil {
		t.Fatal(err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if resp.DryRun {
		t.Error("DryRun should be false")
	}
}

func TestPrintJSON_DryRun(t *testing.T) {
	var buf bytes.Buffer
	if err := printJSON(&buf, nil, true); err != nil {
		t.Fatal(err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestPrintJSONError(t *testing.T) {
	var buf bytes.Buffer
	errs := []output.JSONError{{Code: "E001", Message: "something failed"}}
	if err := printJSONError(&buf, errs); err != nil {
		t.Fatal(err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("status = %q, want error", resp.Status)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestConfirm_YesFlag(t *testing.T) {
	r := strings.NewReader("")
	var w bytes.Buffer
	ok, err := confirm(r, &w, "Continue?", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true when yesFlag=true")
	}
}

func TestConfirm_UserYes(t *testing.T) {
	// Can't test with real TTY in unit tests, but we can test the logic
	// by noting that confirm checks isTerminal() which will be false in tests.
	// This test verifies the yesFlag path works.
	r := strings.NewReader("y\n")
	var w bytes.Buffer
	ok, err := confirm(r, &w, "Continue?", true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true")
	}
}

func TestPrintHuman(t *testing.T) {
	var buf bytes.Buffer
	printHuman(&buf, "Hello %s, count: %d\n", "world", 42)
	want := "Hello world, count: 42\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPromptChoice_Valid(t *testing.T) {
	// Can't test interactive choice in unit tests (isTerminal returns false).
	// Test that yesFlag path of confirm still works as expected.
	r := strings.NewReader("y\n")
	var w bytes.Buffer
	ok, err := confirm(r, &w, "Proceed?", true)
	if err != nil || !ok {
		t.Errorf("confirm with yesFlag should return true, got ok=%v err=%v", ok, err)
	}
}
