package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/larah/nd/internal/output"
)

func TestRemoveCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Remove it
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "remove", "greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", got)
	}
}

func TestRemoveCmd_NotDeployed(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "remove", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-deployed asset")
	}
}

func TestRemoveCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Remove with --dry-run
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "remove", "greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove --dry-run failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
}

func TestRemoveCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Remove with --json
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "remove", "greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove --json failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestRemoveCmd_TypeQualified(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "skills/greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Remove with type/name
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "remove", "skills/greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
}
