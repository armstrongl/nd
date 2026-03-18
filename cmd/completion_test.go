package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "__nd") {
		t.Errorf("expected bash completion to contain '__nd' function, got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionZsh(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "#compdef") || !strings.Contains(got, "nd") {
		t.Errorf("expected zsh completion header, got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionFish(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "fish"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "complete") {
		t.Errorf("expected fish completion to contain 'complete', got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionHidden(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--help"})

	_ = rootCmd.Execute()

	got := out.String()
	if strings.Contains(got, "completion") {
		t.Errorf("completion command should be hidden from help, but found in:\n%s", got)
	}
}

func TestCompletionNoSubcommand(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion"})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "bash") || !strings.Contains(got, "zsh") || !strings.Contains(got, "fish") {
		t.Errorf("expected usage showing bash, zsh, fish, got:\n%s", got)
	}
}

func TestCompletionBashInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".local", "share", "bash-completion", "completions")
	os.MkdirAll(installDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "bash", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "nd"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "__nd") {
		t.Errorf("installed file missing bash completion content")
	}
}

func TestCompletionZshInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".zfunc")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "zsh", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "_nd"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "#compdef") {
		t.Errorf("installed file missing zsh completion content")
	}
}

func TestCompletionFishInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".config", "fish", "completions")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "fish", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "nd.fish"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "complete") {
		t.Errorf("installed file missing fish completion content")
	}
}

func TestCompletionInstallUnwritable(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "bash", "--install", "--install-dir", "/nonexistent/readonly/path"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unwritable path")
	}
}
