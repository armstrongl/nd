package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/larah/nd/internal/output"
)

func TestDoctorCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "doctor"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Config") {
		t.Errorf("expected 'Config' check in output, got: %s", got)
	}
	if !strings.Contains(got, "Git") {
		t.Errorf("expected 'Git' check in output, got: %s", got)
	}
	if !strings.Contains(got, "pass") {
		t.Errorf("expected summary in output, got: %s", got)
	}
}

func TestDoctorCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "doctor"})

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

func TestDoctorCmd_WithSources(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "doctor"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Source") {
		t.Errorf("expected 'Source' check in output, got: %s", got)
	}
}
