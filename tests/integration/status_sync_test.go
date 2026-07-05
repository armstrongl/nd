package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRepairsSymlinks(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Break the symlink
	configDir := filepath.Dir(configPath)
	agentDir := filepath.Join(filepath.Dir(configDir), ".claude")
	symlinkPath := filepath.Join(agentDir, "skills", "greeting")
	os.Remove(symlinkPath)

	// Sync should report healthy (repair the link)
	result = runND(t, "--config", configPath, "sync")
	if result.ExitCode != 0 {
		t.Fatalf("sync exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestStatusReflectsDeployAndRemove(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy greeting.
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Status should show the deployment.
	result = runND(t, "--config", configPath, "status")
	if result.ExitCode != 0 {
		t.Fatalf("status exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "greeting") {
		t.Errorf("expected 'greeting' in status, got: %s", result.Stdout)
	}

	// Remove greeting.
	result = runND(t, "--config", configPath, "--yes", "remove", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("remove exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Status should now report no deployments.
	result = runND(t, "--config", configPath, "status")
	if result.ExitCode != 0 {
		t.Fatalf("status exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if strings.Contains(result.Stdout, "greeting") {
		t.Errorf("expected no 'greeting' after remove, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "No deployments") {
		t.Errorf("expected 'No deployments' after remove, got: %s", result.Stdout)
	}
}

func TestDoctorReportsHealth(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "doctor")
	if result.ExitCode != 0 {
		t.Fatalf("doctor exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "Config") {
		t.Errorf("expected 'Config' in doctor output, got: %s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "pass") {
		t.Errorf("expected 'pass' in doctor output, got: %s", result.Stdout)
	}
}
