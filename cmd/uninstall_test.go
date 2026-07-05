package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/output"
)

func TestUninstallCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Dry-run uninstall
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "uninstall"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("dry-run uninstall failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in output, got: %s", got)
	}
}

func TestUninstallCmd_DryRun_Empty(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "uninstall"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No deployments") {
		t.Errorf("expected 'No deployments' in output, got: %s", got)
	}
}

func TestUninstallCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "--dry-run", "uninstall"})
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

func TestUninstallCmd_MultiAgent(t *testing.T) {
	configPath, _ := setupTwoAgentDeployEnv(t)

	// Deploy greeting to both agents.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "--agents", "claude-code,copilot", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("multi-agent deploy failed: %v\n%s", err, out.String())
	}

	// Uninstall removes every recorded deployment across agents.
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "uninstall"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("uninstall failed: %v\n%s", err, out.String())
	}
	if got := out.String(); strings.Count(got, "Removed") < 2 {
		t.Errorf("expected both agents' deployments removed, got: %s", got)
	}

	// Status should report no remaining deployments for any agent.
	out.Reset()
	app3 := &App{}
	rootCmd3 := NewRootCmd(app3)
	rootCmd3.SetOut(&out)
	rootCmd3.SetErr(&out)
	rootCmd3.SetArgs([]string{"--config", configPath, "status"})
	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(out.String(), "No deployments") {
		t.Errorf("expected 'No deployments' after uninstall, got: %s", out.String())
	}
}

func TestUninstallCmd_WithYes(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Uninstall with --yes
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "uninstall"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Removed") || !strings.Contains(got, "greeting") {
		t.Errorf("expected removal confirmation in output, got: %s", got)
	}
}
