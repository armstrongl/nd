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

func TestStatusCmd_Empty(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources: []\n"), 0o644)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "status"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No deployments") {
		t.Errorf("expected 'No deployments' message, got: %s", out.String())
	}
}

func TestStatusCmd_WithDeployments(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// First deploy something
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Now check status
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "status"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("status should show deployed asset, got: %s", got)
	}
}

func TestStatusCmd_MultiAgent(t *testing.T) {
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

	// status --json should list one deployment per agent, each labeled.
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "status"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map, got %T", resp.Data)
	}
	deployments, ok := data["deployments"].([]interface{})
	if !ok {
		t.Fatalf("expected deployments array, got %T", data["deployments"])
	}
	if len(deployments) != 2 {
		t.Fatalf("expected 2 deployments (one per agent), got %d: %s", len(deployments), out.String())
	}
	agentsSeen := map[string]bool{}
	for _, d := range deployments {
		dm := d.(map[string]interface{})
		if dm["asset_name"] != "greeting" {
			t.Errorf("expected greeting, got %v", dm["asset_name"])
		}
		agentsSeen[dm["agent"].(string)] = true
	}
	if !agentsSeen["claude-code"] || !agentsSeen["copilot"] {
		t.Errorf("expected status to surface both agents, got: %v", agentsSeen)
	}
}

func TestStatusCmd_JSON(t *testing.T) {
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

	// Check status --json
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "status"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("status --json failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}
