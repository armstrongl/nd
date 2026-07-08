package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

	// Dry-run uninstall must leave symlinks and state untouched.
	linkPath := filepath.Join(envAgentDir(configPath), "skills", "greeting")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Errorf("dry-run uninstall must not remove the symlink at %s: %v", linkPath, err)
	}
	data, err := os.ReadFile(envStateFile(configPath))
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if !strings.Contains(string(data), "greeting") {
		t.Errorf("dry-run uninstall must not drop the state entry, got: %s", data)
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

// deployGreetingForUninstall deploys one asset so uninstall has work to do.
// An explicit --scope keeps deploy from prompting for scope when a test has
// forced an interactive terminal.
func deployGreetingForUninstall(t *testing.T, configPath string) {
	t.Helper()
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--scope", "global", "--yes", "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
}

func TestUninstallCmd_ConfirmYes_ViaStdin(t *testing.T) {
	// Confirms uninstall reads from cmd.InOrStdin() (not raw os.Stdin): a "y"
	// answer on a terminal proceeds with the removal.
	setTestTerminal(t, true)
	configPath, _ := setupDeployEnv(t)
	deployGreetingForUninstall(t, configPath)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetIn(strings.NewReader("y\n"))
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "uninstall"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Removed") || !strings.Contains(got, "greeting") {
		t.Errorf("expected removal after 'y' confirmation via stdin, got: %s", got)
	}
}

func TestUninstallCmd_Quiet_SuppressesAbort(t *testing.T) {
	setTestTerminal(t, true)
	configPath, _ := setupDeployEnv(t)
	deployGreetingForUninstall(t, configPath)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetIn(strings.NewReader("n\n"))
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--quiet", "uninstall"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out.String(), "Aborted") {
		t.Errorf("expected 'Aborted.' to be suppressed under --quiet, got: %s", out.String())
	}
}

func TestUninstallCmd_NonTTY_NoYes_Errors(t *testing.T) {
	setTestTerminal(t, false)
	configPath, _ := setupDeployEnv(t)
	deployGreetingForUninstall(t, configPath)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "uninstall"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when confirming uninstall in non-TTY without --yes")
	}
	if !strings.Contains(err.Error(), "confirmation required") {
		t.Errorf("expected 'confirmation required' error, got: %v", err)
	}
}
