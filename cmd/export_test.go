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

func setupExportEnv(t *testing.T) (configPath string, srcDir string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath = filepath.Join(configDir, "config.yaml")

	srcDir = filepath.Join(tmp, "my-source")

	// skills/greeting (directory asset)
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "skills", "greeting", "SKILL.md"), []byte("# Greeting Skill\nSay hello."), 0o644)

	// agents/code-reviewer.md (file asset)
	os.MkdirAll(filepath.Join(srcDir, "agents"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "agents", "code-reviewer.md"), []byte("# Code Reviewer Agent"), 0o644)

	// commands/deploy.md (file asset)
	os.MkdirAll(filepath.Join(srcDir, "commands"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "commands", "deploy.md"), []byte("# Deploy Command"), 0o644)

	// output-styles/concise.md (file asset)
	os.MkdirAll(filepath.Join(srcDir, "output-styles"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "output-styles", "concise.md"), []byte("# Concise Style"), 0o644)

	// rules/go-conventions.md (file asset)
	os.MkdirAll(filepath.Join(srcDir, "rules"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "rules", "go-conventions.md"), []byte("# Go Conventions"), 0o644)

	// hooks/pre-commit-lint (directory asset with hooks.json)
	os.MkdirAll(filepath.Join(srcDir, "hooks", "pre-commit-lint"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "hooks", "pre-commit-lint", "hooks.json"), []byte(`{"hooks":[]}`), 0o644)
	os.WriteFile(filepath.Join(srcDir, "hooks", "pre-commit-lint", "lint.sh"), []byte("#!/bin/sh\necho lint"), 0o644)

	// Create agent deploy target dir
	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)

	cfg := "version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources:\n  - id: my-source\n    type: local\n    path: " + srcDir + "\nagents:\n  - name: claude-code\n    global_dir: " + agentDir + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	return configPath, srcDir
}

func TestExportCmd_Basic(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Exported plugin") {
		t.Errorf("expected 'Exported plugin' in output, got: %s", got)
	}

	// Verify plugin directory was created
	if _, err := os.Stat(filepath.Join(outDir, ".claude-plugin", "plugin.json")); err != nil {
		t.Errorf("expected plugin.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "skills", "greeting", "SKILL.md")); err != nil {
		t.Errorf("expected skill to be copied: %v", err)
	}
}

func TestExportCmd_JSON(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-json")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
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

func TestExportCmd_DryRun(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-dry")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}

	// Verify nothing was actually written
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("expected output dir to not exist during dry-run")
	}
}

func TestExportCmd_NoFlagsNonTTY(t *testing.T) {
	configPath, _ := setupExportEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no flags and non-TTY")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' in error, got: %v", err)
	}
}

func TestExportCmd_InvalidFormat(t *testing.T) {
	configPath, _ := setupExportEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test", "--assets", "invalidformat"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid asset format")
	}
	if !strings.Contains(err.Error(), "invalid asset reference") {
		t.Errorf("expected 'invalid asset reference' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "type/name format") {
		t.Errorf("expected 'type/name format' in error, got: %v", err)
	}
}

func TestExportCmd_PluginTypeRejected(t *testing.T) {
	configPath, _ := setupExportEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test", "--assets", "plugins/foo"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for plugins type")
	}
	if !strings.Contains(err.Error(), "plugins cannot be exported") {
		t.Errorf("expected 'plugins cannot be exported' in error, got: %v", err)
	}
}

func TestExportCmd_AssetNotFound(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-notfound")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test", "--assets", "skills/nonexistent", "--output", outDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestExportCmd_OutputExists(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-exists")

	// Create existing output dir
	os.MkdirAll(outDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when output dir already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

func TestExportCmd_OverwriteExisting(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-overwrite")

	// Create existing output dir
	os.MkdirAll(outDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir, "--overwrite"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error with --overwrite: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Exported plugin") {
		t.Errorf("expected 'Exported plugin' in output, got: %s", got)
	}
}

func TestExportMarketplaceCmd(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))

	// First, export a plugin that we can use as marketplace input
	pluginDir := filepath.Join(tmp, "plugin-out")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", pluginDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to export plugin: %v", err)
	}

	// Now generate a marketplace from the exported plugin
	mktDir := filepath.Join(tmp, "mkt-out")

	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)

	var out2 bytes.Buffer
	rootCmd2.SetOut(&out2)
	rootCmd2.SetErr(&out2)
	rootCmd2.SetArgs([]string{"--config", configPath, "export", "marketplace", "--name", "test-mkt", "--owner", "Test", "--plugins", pluginDir, "--output", mktDir})

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out2.String()
	if !strings.Contains(got, "Generated marketplace") {
		t.Errorf("expected 'Generated marketplace' in output, got: %s", got)
	}

	// Verify marketplace structure
	if _, err := os.Stat(filepath.Join(mktDir, ".claude-plugin", "marketplace.json")); err != nil {
		t.Errorf("expected marketplace.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(mktDir, "plugins", "test-plugin")); err != nil {
		t.Errorf("expected plugin dir in marketplace: %v", err)
	}
}

func TestExportCmd_DryRunJSON(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-dryjson")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "--json", "export", "--name", "test-plugin", "--assets", "skills/greeting", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
	if !resp.DryRun {
		t.Errorf("expected dryRun=true in response")
	}
}

func TestExportCmd_MultipleAssets(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-multi")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting,agents/code-reviewer.md", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Exported plugin") {
		t.Errorf("expected 'Exported plugin' in output, got: %s", got)
	}
	if !strings.Contains(got, "2 asset(s) copied") {
		t.Errorf("expected '2 asset(s) copied' in output, got: %s", got)
	}
}

func TestExportMarketplaceCmd_MissingFlags(t *testing.T) {
	configPath, _ := setupExportEnv(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing name",
			args:    []string{"--config", configPath, "export", "marketplace", "--owner", "Test", "--plugins", "/tmp/fake"},
			wantErr: "--name is required",
		},
		{
			name:    "missing owner",
			args:    []string{"--config", configPath, "export", "marketplace", "--name", "test-mkt", "--plugins", "/tmp/fake"},
			wantErr: "--owner is required",
		},
		{
			name:    "missing plugins",
			args:    []string{"--config", configPath, "export", "marketplace", "--name", "test-mkt", "--owner", "Test"},
			wantErr: "--plugins is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			rootCmd := NewRootCmd(app)
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected %q in error, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestExportCmd_SourceFilter(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-source")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--source", "my-source", "--output", outDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Exported plugin") {
		t.Errorf("expected 'Exported plugin' in output, got: %s", got)
	}
}

func TestExportCmd_SourceFilterNotFound(t *testing.T) {
	configPath, _ := setupExportEnv(t)
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	outDir := filepath.Join(tmp, "out-srcnf")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "export", "--name", "test-plugin", "--assets", "skills/greeting", "--source", "nonexistent-source", "--output", outDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
