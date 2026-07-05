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

	// With no user sources, the builtin source is still present
	got := out.String()
	if strings.Contains(got, "No sources") {
		t.Errorf("builtin source should always be listed, got 'No sources' message")
	}
	if !strings.Contains(got, "builtin") {
		t.Errorf("expected builtin source in output, got: %s", got)
	}
}

func TestSourceBare_DelegatesToList(t *testing.T) {
	_, configPath := setupTestConfig(t)

	// Bare `nd source` should behave exactly like `nd source list`.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var bare bytes.Buffer
	rootCmd.SetOut(&bare)
	rootCmd.SetErr(&bare)
	rootCmd.SetArgs([]string{"--config", configPath, "source"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	var list bytes.Buffer
	rootCmd2.SetOut(&list)
	rootCmd2.SetErr(&list)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "list"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if bare.String() != list.String() {
		t.Errorf("bare `nd source` output %q != `nd source list` output %q", bare.String(), list.String())
	}
	if strings.Contains(bare.String(), "Usage:") || strings.Contains(bare.String(), "Available Commands:") {
		t.Errorf("bare `nd source` should not print Cobra usage, got:\n%s", bare.String())
	}
	// The builtin source is always listed, confirming the list path ran.
	if !strings.Contains(bare.String(), "builtin") {
		t.Errorf("expected builtin source in bare output, got: %s", bare.String())
	}
}

func TestSourceBare_JSON(t *testing.T) {
	_, configPath := setupTestConfig(t)

	// `--json` must behave the same on the bare parent as on the explicit list.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "source"})
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

func TestSourceBare_UnknownSubcommand(t *testing.T) {
	_, configPath := setupTestConfig(t)

	// An unrecognized subcommand must still error, not be swallowed by the
	// delegating RunE.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "source", "bogus"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected 'unknown command' error, got: %v", err)
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
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "source", "remove", "my-skills"})

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
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "source", "remove", "nonexistent"})

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
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "--yes", "source", "remove", "my-skills"})

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

func TestSourceRemoveCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "__complete", "source", "remove", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "my-source") {
		t.Errorf("expected 'my-source' in source remove completions, got:\n%s", got)
	}
}

func TestSourceRemove_WithYes(t *testing.T) {
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

	// Remove with --yes (new flag, replaces --force)
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "source", "remove", "my-skills"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove with --yes failed: %v", err)
	}

	got2 := out.String()
	if !strings.Contains(got2, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", got2)
	}
}

// TestSourceRemove_PurgesDeployments deploys an asset from a source, then
// removes that source with --yes, and asserts both the symlink and the state
// entry are gone (the removeSourceDeployments -> RemoveBulk cleanup path).
func TestSourceRemove_PurgesDeployments(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy greeting from the pre-registered my-source.
	runND(t, configPath, "deploy", "greeting")

	linkPath := filepath.Join(envAgentDir(configPath), "skills", "greeting")
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("setup: expected deployed symlink at %s: %v", linkPath, err)
	}

	// Remove the source and all its deployed assets.
	runND(t, configPath, "--yes", "source", "remove", "my-source")

	// (a) The symlink under the agent dir is gone.
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Errorf("expected symlink removed after source remove (err=%v)", err)
	}

	// (b) No state entry for that source remains.
	data, err := os.ReadFile(envStateFile(configPath))
	if err != nil {
		if os.IsNotExist(err) {
			return // no state file is acceptable
		}
		t.Fatalf("read state file: %v", err)
	}
	if strings.Contains(string(data), "my-source") || strings.Contains(string(data), "greeting") {
		t.Errorf("state should not reference the removed source's deployment, got: %s", data)
	}
}

func TestSourceRemove_NonTTY_NoYes_Errors(t *testing.T) {
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

	// Remove without --yes in non-TTY — confirm reads EOF → error
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "remove", "my-skills"})

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("expected error when confirm reads EOF in non-TTY")
	}
}

func TestSourceRemove_DeployedAssets_NonTTY_Actionable(t *testing.T) {
	setTestTerminal(t, false)
	configPath, _ := setupDeployEnv(t)

	// Deploy an asset from my-source so the source has deployed assets.
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Remove the source without --yes in a non-TTY: must return the actionable
	// "use --yes to remove" error, NOT the opaque promptChoice message.
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "remove", "my-source"})

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("expected error removing a source with deployed assets in non-TTY without --yes")
	}
	if !strings.Contains(err.Error(), "use --yes to remove") {
		t.Errorf("expected actionable 'use --yes to remove' error, got: %v", err)
	}
	if strings.Contains(err.Error(), "interactive choice required") {
		t.Errorf("got opaque promptChoice error instead of actionable one: %v", err)
	}
}

func TestSourceRemove_ForceAlias(t *testing.T) {
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

	// Remove with hidden --force alias (backwards compat)
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "source", "remove", "--force", "my-skills"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove with --force alias failed: %v", err)
	}

	got2 := out.String()
	if !strings.Contains(got2, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", got2)
	}
}
