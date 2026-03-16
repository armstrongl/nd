package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/output"
)

func TestPinCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy an asset first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Pin it
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "pin", "greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Pinned") {
		t.Errorf("expected 'Pinned' in output, got: %s", got)
	}
}

func TestUnpinCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy and pin
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "pin", "greeting"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("pin failed: %v", err)
	}

	// Unpin
	out.Reset()
	app3 := &App{}
	rootCmd3 := NewRootCmd(app3)
	rootCmd3.SetOut(&out)
	rootCmd3.SetErr(&out)
	rootCmd3.SetArgs([]string{"--config", configPath, "unpin", "greeting"})

	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("unpin failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Unpinned") {
		t.Errorf("expected 'Unpinned' in output, got: %s", got)
	}
}

func TestPinCmd_NotDeployed(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "pin", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-deployed asset")
	}
}

func TestPinCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Pin with JSON
	out.Reset()
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--json", "pin", "greeting"})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("pin --json failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}
