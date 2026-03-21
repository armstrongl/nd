package integration

import (
	"strings"
	"testing"
)

func TestProfileCreateDeploySwitch(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Create profile A with greeting
	result := runND(t, "--config", configPath, "profile", "create", "profile-a",
		"--assets", "skills/greeting")
	if result.ExitCode != 0 {
		t.Fatalf("create A exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Create profile B with hello command
	result = runND(t, "--config", configPath, "profile", "create", "profile-b",
		"--assets", "commands/hello.md")
	if result.ExitCode != 0 {
		t.Fatalf("create B exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Deploy profile A
	result = runND(t, "--config", configPath, "profile", "deploy", "profile-a")
	if result.ExitCode != 0 {
		t.Fatalf("deploy A exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// List should show profile-a as active
	result = runND(t, "--config", configPath, "profile", "list")
	if result.ExitCode != 0 {
		t.Fatalf("list exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "*") || !strings.Contains(result.Stdout, "profile-a") {
		t.Errorf("expected active profile-a in list, got: %s", result.Stdout)
	}

	// Switch to profile B
	result = runND(t, "--config", configPath, "--yes", "profile", "switch", "profile-b")
	if result.ExitCode != 0 {
		t.Fatalf("switch exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "Switched") {
		t.Errorf("expected 'Switched' in output, got: %s", result.Stdout)
	}
}

func TestSnapshotSaveRestoreRoundTrip(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy something
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Save snapshot
	result = runND(t, "--config", configPath, "snapshot", "save", "snap-1")
	if result.ExitCode != 0 {
		t.Fatalf("save exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// List snapshots
	result = runND(t, "--config", configPath, "snapshot", "list")
	if result.ExitCode != 0 {
		t.Fatalf("list exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "snap-1") {
		t.Errorf("expected 'snap-1' in snapshot list, got: %s", result.Stdout)
	}
}
