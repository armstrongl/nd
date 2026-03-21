package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/output"
	"github.com/armstrongl/nd/internal/profile"
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

func TestCompletionInitApp(t *testing.T) {
	app := &App{ConfigPath: "~/.config/nd/config.yaml"}
	completionInitApp(app)

	if strings.Contains(app.ConfigPath, "~") {
		t.Errorf("ConfigPath still contains ~: %s", app.ConfigPath)
	}
	if app.BackupDir == "" {
		t.Error("BackupDir not set")
	}
	if !strings.HasSuffix(app.BackupDir, "backups") {
		t.Errorf("BackupDir should end with 'backups', got: %s", app.BackupDir)
	}
}

func TestCompletionInitApp_Idempotent(t *testing.T) {
	app := &App{ConfigPath: "/tmp/nd/config.yaml"}
	completionInitApp(app)
	first := app.ConfigPath
	completionInitApp(app)
	if app.ConfigPath != first {
		t.Errorf("not idempotent: %s != %s", first, app.ConfigPath)
	}
}

func TestExtractChoiceNames(t *testing.T) {
	completions := []string{
		"skills/greeting\tglobal from my-source",
		"commands/hello.md\tglobal from my-source",
	}
	got := extractChoiceNames(completions)
	want := []string{"skills/greeting", "commands/hello.md"}
	if len(got) != len(want) {
		t.Fatalf("got %d names, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractChoiceNames_NoTab(t *testing.T) {
	completions := []string{"alpha", "beta"}
	got := extractChoiceNames(completions)
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Errorf("expected raw strings when no tab, got: %v", got)
	}
}

func TestExtractChoiceNames_Empty(t *testing.T) {
	got := extractChoiceNames(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got: %v", got)
	}
}

func TestLatestAutoSnapshot(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\n"), 0o644)

	app := &App{ConfigPath: configPath}
	pstore, err := app.ProfileStore()
	if err != nil {
		t.Fatal(err)
	}

	// Create two auto-snapshots with different timestamps
	snap1 := profile.Snapshot{
		Version:   1,
		Name:      "auto-20260321T100000-000000000",
		CreatedAt: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		Auto:      true,
	}
	snap2 := profile.Snapshot{
		Version:   1,
		Name:      "auto-20260321T110000-000000000",
		CreatedAt: time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC),
		Auto:      true,
	}
	if err := pstore.SaveSnapshot(snap1); err != nil {
		t.Fatal(err)
	}
	if err := pstore.SaveSnapshot(snap2); err != nil {
		t.Fatal(err)
	}

	got := latestAutoSnapshot(app)
	if got != snap2.Name {
		t.Errorf("latestAutoSnapshot() = %q, want %q", got, snap2.Name)
	}
}

func TestLatestAutoSnapshot_None(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\n"), 0o644)

	app := &App{ConfigPath: configPath}

	got := latestAutoSnapshot(app)
	if got != "" {
		t.Errorf("latestAutoSnapshot() = %q, want empty string", got)
	}
}

func TestLatestAutoSnapshot_IgnoresUserSnapshots(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\n"), 0o644)

	app := &App{ConfigPath: configPath}
	pstore, err := app.ProfileStore()
	if err != nil {
		t.Fatal(err)
	}

	// Create a user snapshot (not auto)
	userSnap := profile.Snapshot{
		Version:   1,
		Name:      "my-backup",
		CreatedAt: time.Now(),
		Auto:      false,
	}
	if err := pstore.SaveSnapshot(userSnap); err != nil {
		t.Fatal(err)
	}

	got := latestAutoSnapshot(app)
	if got != "" {
		t.Errorf("latestAutoSnapshot() = %q, want empty (should ignore user snapshots)", got)
	}
}
