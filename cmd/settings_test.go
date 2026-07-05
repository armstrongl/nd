package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsEditCmd_NoConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "nonexistent", "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "settings", "edit"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config doesn't exist")
	}
	if !strings.Contains(err.Error(), "init") {
		t.Errorf("expected error to suggest 'nd init', got: %v", err)
	}
}

func TestSettingsEditCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "settings", "edit"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
}

func TestSettingsBare_DelegatesToEdit(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Bare `nd settings` should behave exactly like `nd settings edit`.
	// Use --dry-run so no real editor is launched.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var bare bytes.Buffer
	rootCmd.SetOut(&bare)
	rootCmd.SetErr(&bare)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "settings"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	var edit bytes.Buffer
	rootCmd2.SetOut(&edit)
	rootCmd2.SetErr(&edit)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "settings", "edit"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	if bare.String() != edit.String() {
		t.Errorf("bare `nd settings` output %q != `nd settings edit` output %q", bare.String(), edit.String())
	}
	if !strings.Contains(bare.String(), "dry-run") {
		t.Errorf("expected 'dry-run' in bare output, got: %s", bare.String())
	}
	if strings.Contains(bare.String(), "Usage:") || strings.Contains(bare.String(), "Available Commands:") {
		t.Errorf("bare `nd settings` should not print Cobra usage, got:\n%s", bare.String())
	}
}

func TestSettingsBare_UnknownSubcommand(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "settings", "bogus"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
	}
}

// settingsEditGuardCase drives `settings edit` under a non-interactive signal and
// asserts it returns an actionable error instead of exec-ing an editor (hanging).
func settingsEditGuardCase(t *testing.T, interactive bool, args ...string) {
	t.Helper()
	// Force the terminal state so the editor is never exec'd even if the guard
	// regresses; the guard returns before the exec block in every case here.
	setTestTerminal(t, interactive)
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append([]string{"--config", configPath}, args...))

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected error for non-interactive settings edit (args=%v)", args)
	}
	if !strings.Contains(err.Error(), "requires an interactive terminal") {
		t.Errorf("expected 'requires an interactive terminal' error, got: %v", err)
	}
}

func TestSettingsEditCmd_JSON_Errors(t *testing.T) {
	// TTY is true so only --json trips the guard; the editor must not run.
	settingsEditGuardCase(t, true, "--json", "settings", "edit")
}

func TestSettingsEditCmd_Quiet_Errors(t *testing.T) {
	// TTY is true so only --quiet trips the guard; the editor must not run.
	settingsEditGuardCase(t, true, "--quiet", "settings", "edit")
}

func TestSettingsEditCmd_NonTTY_Errors(t *testing.T) {
	settingsEditGuardCase(t, false, "settings", "edit")
}
