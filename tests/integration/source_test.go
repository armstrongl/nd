package integration

import (
	"strings"
	"testing"
)

func TestSourceAddList(t *testing.T) {
	configPath, srcDir := setupIntegrationEnv(t)
	_ = srcDir

	// Source already registered via config — list should show it
	result := runND(t, "--config", configPath, "source", "list")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "my-source") {
		t.Errorf("expected 'my-source' in output, got: %s", result.Stdout)
	}
}

func TestSourceListJSON(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "--json", "source", "list")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status"`) {
		t.Errorf("expected JSON envelope in output, got: %s", result.Stdout)
	}
}
