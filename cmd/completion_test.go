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

// runBareCompletion executes bare `nd completion` (no subcommand) and returns
// the captured output plus any error.
func runBareCompletion(t *testing.T) (string, error) {
	t.Helper()
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion"})
	err := rootCmd.Execute()
	return out.String(), err
}

// runExplicitCompletion executes `nd completion <shell>` and returns the output.
func runExplicitCompletion(t *testing.T, shell string) string {
	t.Helper()
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", shell})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("nd completion %s failed: %v", shell, err)
	}
	return out.String()
}

// TestCompletionShellDetection verifies that bare `nd completion` auto-detects
// the shell from $SHELL and emits the matching script (byte-identical to the
// explicit subcommand), and errors listing the supported shells otherwise.
func TestCompletionShellDetection(t *testing.T) {
	tests := []struct {
		name      string
		shell     string // value for $SHELL
		shellName string // detected shell subcommand name (bash/zsh/fish)
		marker    string
	}{
		{name: "zsh", shell: "/bin/zsh", shellName: "zsh", marker: "#compdef"},
		{name: "bash", shell: "/bin/bash", shellName: "bash", marker: "__nd"},
		{name: "fish", shell: "/usr/local/bin/fish", shellName: "fish", marker: "complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shell)

			got, err := runBareCompletion(t)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want := runExplicitCompletion(t, tt.shellName); got != want {
				t.Errorf("detected %s output does not match explicit `nd completion %s`", tt.shellName, tt.shellName)
			}
			if !strings.Contains(got, tt.marker) {
				t.Errorf("expected %s completion to contain %q, got:\n%s", tt.shellName, tt.marker, got[:min(200, len(got))])
			}
		})
	}

	errCases := []struct {
		name  string
		shell string
		unset bool
	}{
		{name: "unset", unset: true},
		{name: "csh", shell: "/bin/csh"},
	}

	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv registers a cleanup that restores the original value; for
			// the unset case we then remove it so detection sees no $SHELL.
			t.Setenv("SHELL", tt.shell)
			if tt.unset {
				if err := os.Unsetenv("SHELL"); err != nil {
					t.Fatal(err)
				}
			}

			got, err := runBareCompletion(t)
			if err == nil {
				t.Fatalf("expected error for %s, got output:\n%s", tt.name, got)
			}
			for _, want := range []string{"bash", "zsh", "fish"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q should mention supported shell %q", err.Error(), want)
				}
			}
		})
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
