package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/output"
)

func TestProfileCreateCmd(t *testing.T) {
	configPath, srcDir := setupDeployEnv(t)
	_ = srcDir

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "test-profile",
		"--assets", "skills/greeting", "--description", "A test profile"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "test-profile") {
		t.Errorf("expected profile name in output, got: %s", got)
	}
}

func TestProfileCreateCmd_FromCurrent(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy an asset first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Now create profile from current deployments
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "create", "from-current-test", "--from-current"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "from-current-test") {
		t.Errorf("expected profile name in output, got: %s", got)
	}
}

func TestProfileCreateCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "profile", "create", "json-profile",
		"--assets", "skills/greeting"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestProfileCreateCmd_Duplicate(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create once
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "dup-test", "--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// Create again — should fail
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "create", "dup-test", "--assets", "skills/greeting"})
	if err := rootCmd2.Execute(); err == nil {
		t.Fatal("expected error for duplicate profile")
	}
}

func TestProfileDeleteCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "delete-me", "--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Delete it
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "profile", "delete", "delete-me"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Deleted") {
		t.Errorf("expected 'Deleted' in output, got: %s", got)
	}
}

func TestProfileDeleteCmd_NotFound(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "profile", "delete", "nonexistent"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestProfileListCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create a profile
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "list-test", "--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// List profiles
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "list"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "list-test") {
		t.Errorf("expected profile name in output, got: %s", got)
	}
}

func TestProfileListCmd_Empty(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No profiles") {
		t.Errorf("expected 'No profiles' in output, got: %s", got)
	}
}

func TestProfileListCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "profile", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestProfileDeployCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "deploy-test",
		"--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Deploy profile
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "deploy", "deploy-test"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "deploy-test") {
		t.Errorf("expected profile name in output, got: %s", got)
	}
}

func TestProfileDeployCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "dryrun-test",
		"--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Deploy with dry-run
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "profile", "deploy", "dryrun-test"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("dry-run deploy failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
}

func TestProfileAddAssetCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile with one asset
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "add-asset-test",
		"--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Add another asset
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "add-asset", "add-asset-test", "commands/hello"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("add-asset failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "hello") {
		t.Errorf("expected asset name in output, got: %s", got)
	}
}

func TestProfileAddAssetCmd_Duplicate(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile with greeting
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "dup-asset-test",
		"--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Try adding the same asset again — should fail
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "profile", "add-asset", "dup-asset-test", "skills/greeting"})
	if err := rootCmd2.Execute(); err == nil {
		t.Fatal("expected error for duplicate asset")
	}
}

func TestProfileAddAssetCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create profile
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "dryrun-add-test",
		"--assets", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Add asset with --dry-run
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "profile", "add-asset", "dryrun-add-test", "commands/hello"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("dry-run add-asset failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}

	// Verify the asset was NOT actually added
	app3 := &App{}
	rootCmd3 := NewRootCmd(app3)
	out.Reset()
	rootCmd3.SetOut(&out)
	rootCmd3.SetErr(&out)
	rootCmd3.SetArgs([]string{"--config", configPath, "--json", "profile", "list"})
	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	// Profile should still have only 1 asset (the original greeting)
	if strings.Contains(out.String(), `"asset_count":2`) {
		t.Error("dry-run should not have persisted the asset addition")
	}
}
