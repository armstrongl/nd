package integration

import (
	"strings"
	"testing"
)

func TestJSONFlag(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "--json", "source", "list")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status"`) {
		t.Errorf("expected JSON status field, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, `"ok"`) {
		t.Errorf("expected status ok, got: %s", result.Stdout)
	}
}

func TestDryRunFlag(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "--dry-run", "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", result.Stdout)
	}
}

func TestYesFlag(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy then uninstall with --yes (should not hang)
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d", result.ExitCode)
	}

	result = runND(t, "--config", configPath, "--yes", "uninstall")
	if result.ExitCode != 0 {
		t.Fatalf("uninstall exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "Removed") {
		t.Errorf("expected 'Removed' in output, got: %s", result.Stdout)
	}
}

func TestVersionOutput(t *testing.T) {
	result := runND(t, "version")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "nd version") {
		t.Errorf("expected version output, got: %s", result.Stdout)
	}
}

func TestHelpOutput(t *testing.T) {
	result := runND(t, "--help")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "coding agent assets") {
		t.Errorf("expected app description in help, got: %s", result.Stdout)
	}
}
