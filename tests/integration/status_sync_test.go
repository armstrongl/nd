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
