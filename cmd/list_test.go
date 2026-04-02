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

func TestListCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in output, got: %s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", got)
	}
}

func TestListCmd_FilterByType(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "list", "--type", "skills"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in output, got: %s", got)
	}
	if strings.Contains(got, "hello") {
		t.Error("should not contain 'hello' (commands type) when filtering by skills")
	}
}

func TestListCmd_FilterByPattern(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "list", "--pattern", "greet"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in output, got: %s", got)
	}
	if strings.Contains(got, "hello") {
		t.Error("should not contain 'hello' when filtering by 'greet'")
	}
}

func TestListCmd_Empty(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources: []\n"), 0o644)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With no user sources, the builtin source still provides assets
	got := out.String()
	if strings.Contains(got, "No assets") {
		t.Errorf("builtin assets should be listed, got 'No assets' message")
	}
	if !strings.Contains(got, "builtin") {
		t.Errorf("expected builtin source assets in output, got: %s", got)
	}
}

func TestListCmd_TypeFlagCompletion(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "list", "--type", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "skills") || !strings.Contains(got, "commands") {
		t.Errorf("expected asset types in type flag completions, got:\n%s", got)
	}
}

func TestListCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "list"})

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

func TestListCmd_ContextMeta(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath := filepath.Join(configDir, "config.yaml")

	srcDir := filepath.Join(tmp, "my-source")
	contextDir := filepath.Join(srcDir, "context", "my-rules")
	os.MkdirAll(contextDir, 0o755)
	os.WriteFile(filepath.Join(contextDir, "CLAUDE.md"), []byte("# Rules"), 0o644)
	os.WriteFile(filepath.Join(contextDir, "_meta.yaml"), []byte("description: Project-specific coding rules\ntags:\n  - coding\n  - rules\n"), 0o644)

	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)

	cfg := "version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources:\n  - id: my-source\n    type: local\n    path: " + srcDir + "\nagents:\n  - name: claude-code\n    global_dir: " + agentDir + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Project-specific coding rules") {
		t.Errorf("expected meta description in output, got: %s", got)
	}
}

func TestListCmd_ContextMetaJSON(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath := filepath.Join(configDir, "config.yaml")

	srcDir := filepath.Join(tmp, "my-source")
	contextDir := filepath.Join(srcDir, "context", "my-rules")
	os.MkdirAll(contextDir, 0o755)
	os.WriteFile(filepath.Join(contextDir, "CLAUDE.md"), []byte("# Rules"), 0o644)
	os.WriteFile(filepath.Join(contextDir, "_meta.yaml"), []byte("description: Project-specific coding rules\ntags:\n  - coding\n  - rules\n"), 0o644)

	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)

	cfg := "version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources:\n  - id: my-source\n    type: local\n    path: " + srcDir + "\nagents:\n  - name: claude-code\n    global_dir: " + agentDir + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "list"})

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

	// Marshal Data back to JSON and check for the description field
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal data: %v", err)
	}
	if !strings.Contains(string(dataBytes), "Project-specific coding rules") {
		t.Errorf("expected meta description in JSON data, got: %s", string(dataBytes))
	}
}
