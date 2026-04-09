package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd_Help(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"--help"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() == 0 {
		t.Error("expected help output")
	}
}

func TestRootCmd_Version(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"version"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCmd_InvalidScope(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"version", "--scope", "invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
}

func TestRootCmd_MutualExclusion(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"version", "--verbose", "--quiet"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for --verbose + --quiet")
	}
}

func TestScopeFlagCompletion(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "list", "--scope", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "global") || !strings.Contains(got, "project") {
		t.Errorf("expected 'global' and 'project' in scope completions, got:\n%s", got)
	}
}

func TestRootCmd_VersionFlag(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)
	rootCmd.SetArgs([]string{"--version"})

	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "nd version") {
		t.Errorf("expected version info in output, got: %s", got)
	}
}

func TestNeedsInit_ExemptCommands(t *testing.T) {
	exempt := []string{"nd", "init", "version", "completion", "help"}
	for _, name := range exempt {
		cmd := &cobra.Command{Use: name}
		if needsInit(cmd) {
			t.Errorf("needsInit(%q) = true, want false", name)
		}
	}
}

func TestNeedsInit_NonExemptCommands(t *testing.T) {
	nonExempt := []string{"deploy", "list", "status", "doctor", "remove", "source"}
	for _, name := range nonExempt {
		cmd := &cobra.Command{Use: name}
		if !needsInit(cmd) {
			t.Errorf("needsInit(%q) = false, want true", name)
		}
	}
}

func TestFirstRunPrompt_VersionSkipped(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "nonexistent", "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "not initialized") {
		t.Error("version command should not trigger init prompt")
	}
}

func TestFirstRunPrompt_NonInteractive_ShowsHint(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "nonexistent", "config.yaml")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	// Pipe stdin so isTerminal() returns false
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"--config", configPath, "list"})

	// Command will likely fail after the hint (no config), but the hint should appear
	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "not initialized") {
		t.Errorf("expected 'not initialized' warning, got: %s", got)
	}
	if !strings.Contains(got, "nd init") {
		t.Errorf("expected 'nd init' hint, got: %s", got)
	}
}

func TestFirstRunPrompt_YesFlag_RunsInit(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	// --yes auto-accepts the init prompt; list will then run (with empty sources)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "list"})

	_ = rootCmd.Execute()

	// Config should now exist
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected config file to be created: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Initialized") {
		t.Errorf("expected 'Initialized' in output, got: %s", got)
	}
}

func TestWithExitCode(t *testing.T) {
	err := withExitCode(2, &exitError{code: 1, err: nil})
	code, ok := exitCodeFromError(err)
	if !ok {
		t.Fatal("expected to extract exit code")
	}
	if code != 2 {
		t.Errorf("got code %d, want 2", code)
	}
}
