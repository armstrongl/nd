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

func TestSettingsEditJSONNoHang(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// `settings edit --json` must return an actionable error immediately instead
	// of exec-ing an interactive editor (which would hang a scripted run).
	result := runND(t, "--config", configPath, "--json", "settings", "edit")
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0; stdout: %s", result.Stdout)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		t.Errorf("expected empty stdout under --json, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "requires an interactive terminal") {
		t.Errorf("expected actionable error on stderr, got: %s", result.Stderr)
	}
}

func TestExportJSONMissingFlags(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// `export --json` with no --name/--assets must error without launching a form
	// and without writing any non-JSON noise to stdout.
	result := runND(t, "--config", configPath, "--json", "export")
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0; stdout: %s", result.Stdout)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		t.Errorf("expected empty stdout under --json, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "--name is required") {
		t.Errorf("expected '--name is required' on stderr, got: %s", result.Stderr)
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
