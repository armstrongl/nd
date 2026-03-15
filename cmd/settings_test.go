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
