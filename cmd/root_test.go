package cmd

import (
	"bytes"
	"testing"
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
