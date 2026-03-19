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

func setupDeployEnv(t *testing.T) (configPath string, srcDir string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	// Pre-create state dir so file lock works
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath = filepath.Join(configDir, "config.yaml")

	srcDir = filepath.Join(tmp, "my-source")
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "skills", "greeting", "SKILL.md"), []byte("# Greeting"), 0o644)
	os.MkdirAll(filepath.Join(srcDir, "commands", "hello"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "commands", "hello", "command.md"), []byte("# Hello"), 0o644)

	// Create agent deploy target dir
	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)

	// Write config with the source pre-registered and agent global_dir override
	cfg := "version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources:\n  - id: my-source\n    type: local\n    path: " + srcDir + "\nagents:\n  - name: claude-code\n    global_dir: " + agentDir + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	return configPath, srcDir
}

func TestDeployCmd_SingleAsset(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Deployed") {
		t.Errorf("expected 'Deployed' in output, got: %s", got)
	}
}

func TestDeployCmd_TypeQualified(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "skills/greeting"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeployCmd_NotFound(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
}

func TestDeployCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "deploy", "greeting"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
}

func TestDeployCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "deploy", "greeting"})

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

func TestDeployCmd_Multiple(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting", "hello"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "greeting") || !strings.Contains(got, "hello") {
		t.Errorf("expected both assets in output, got: %s", got)
	}
}

func TestDeployCmd_NoArgs_NonTTY(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no args and non-TTY")
	}
	if !strings.Contains(err.Error(), "requires at least one asset") {
		t.Errorf("expected 'requires at least one asset' in error, got: %v", err)
	}
}

func TestDeployCmd_RelativeFlag(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "--relative", "greeting"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Resolve the agent global_dir from the config to find the deployed symlink
	// configPath = <tmp>/.config/nd/config.yaml → three Dir() calls to reach tmp
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	agentDir := filepath.Join(tmp, ".claude")
	linkPath := filepath.Join(agentDir, "skills", "greeting")

	target, readErr := os.Readlink(linkPath)
	if readErr != nil {
		t.Fatalf("expected symlink at %s: %v", linkPath, readErr)
	}
	// A relative symlink target must not be absolute
	if filepath.IsAbs(target) {
		t.Errorf("expected relative symlink, got absolute target: %q", target)
	}
}

func TestDeployCmd_AbsoluteFlag(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "--absolute", "greeting"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath)))
	agentDir := filepath.Join(tmp, ".claude")
	linkPath := filepath.Join(agentDir, "skills", "greeting")

	target, readErr := os.Readlink(linkPath)
	if readErr != nil {
		t.Fatalf("expected symlink at %s: %v", linkPath, readErr)
	}
	// An absolute symlink target must be an absolute path
	if !filepath.IsAbs(target) {
		t.Errorf("expected absolute symlink, got relative target: %q", target)
	}
}

func TestDeployCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "__complete", "deploy", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in completions, got:\n%s", got)
	}
}
