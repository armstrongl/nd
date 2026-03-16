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

// setupTestConfig creates a temp directory with a config file and returns the path.
func setupTestConfig(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources: []\n"), 0o644)
	return tmp, configPath
}

func TestSourceAddLocal(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "my-skills") {
		t.Errorf("output should contain source name, got: %s", got)
	}
}

func TestSourceAddLocal_WithAlias(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(srcDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", "--alias", "work-skills", srcDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "work-skills") {
		t.Errorf("output should contain alias, got: %s", got)
	}
}

func TestSourceAddLocal_JSON(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(srcDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "source", "add", srcDir})

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

func TestSourceAddLocal_NotFound(t *testing.T) {
	_, configPath := setupTestConfig(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", "/nonexistent/path"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestSourceAddLocal_Duplicate(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(srcDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Second add — need fresh root because Cobra reuses command state
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "add", srcDir})

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate source")
	}
}

func TestSourceList(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "list"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "my-skills") {
		t.Errorf("list should contain source name, got: %s", got)
	}
}

func TestSourceList_Empty(t *testing.T) {
	_, configPath := setupTestConfig(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "list"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No sources") {
		t.Errorf("empty list should say 'No sources', got: %s", got)
	}
}

func TestSourceList_JSON(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "source", "list"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("list --json failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestSourceRemove_Force(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(srcDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "remove", "--force", "my-skills"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", got)
	}

	// Verify source is gone
	out.Reset()
	app3 := &App{}
	rootCmd3 := NewRootCmd(app3)
	rootCmd3.SetOut(&out)
	rootCmd3.SetErr(&out)
	rootCmd3.SetArgs([]string{"--config", configPath, "source", "list"})
	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("list after remove failed: %v", err)
	}
	if strings.Contains(out.String(), "my-skills") {
		t.Error("source should be removed from list")
	}
}

func TestSourceRemove_NotFound(t *testing.T) {
	_, configPath := setupTestConfig(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "remove", "--force", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestSourceRemove_JSON(t *testing.T) {
	tmp, configPath := setupTestConfig(t)

	srcDir := filepath.Join(tmp, "my-skills")
	os.MkdirAll(srcDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "add", srcDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "source", "remove", "--force", "my-skills"})

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
