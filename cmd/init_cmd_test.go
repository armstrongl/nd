package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larah/nd/internal/output"
)

func TestInitCmd_WithYes(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Initialized") {
		t.Errorf("expected 'Initialized' in output, got: %s", got)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestInitCmd_DirectoryStructure(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	configPath := filepath.Join(configDir, "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify expected directories exist
	expectedDirs := []string{
		"profiles",
		"snapshots",
		"state",
	}
	for _, dir := range expectedDirs {
		path := filepath.Join(configDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %q not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", dir)
		}
	}
}

func TestInitCmd_AlreadyExists(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
}

func TestInitCmd_JSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "--json", "init"})

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
