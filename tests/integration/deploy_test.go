package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployAndStatus(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "Deployed") {
		t.Errorf("expected 'Deployed' in output, got: %s", result.Stdout)
	}

	// Status should show the deployment
	result = runND(t, "--config", configPath, "status")
	if result.ExitCode != 0 {
		t.Fatalf("status exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "greeting") {
		t.Errorf("expected 'greeting' in status, got: %s", result.Stdout)
	}
}

func TestDeployNotFound(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "deploy", "nonexistent")
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit code for nonexistent asset")
	}
}

func TestDeployDryRun(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "--dry-run", "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", result.Stdout)
	}

	// Verify nothing was actually deployed (no symlinks created)
	// Status should show nothing
	result2 := runND(t, "--config", configPath, "status")
	if strings.Contains(result2.Stdout, "greeting") {
		t.Error("dry-run should not have created any deployments")
	}
}

func TestDeployJSON(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	result := runND(t, "--config", configPath, "--json", "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, `"status"`) {
		t.Errorf("expected JSON envelope, got: %s", result.Stdout)
	}
}

func TestDeployCreateSymlink(t *testing.T) {
	configPath, _ := setupIntegrationEnv(t)

	// Deploy
	result := runND(t, "--config", configPath, "deploy", "greeting")
	if result.ExitCode != 0 {
		t.Fatalf("deploy exit code %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	// Verify symlink exists — agentDir is a sibling of .config/nd
	tmp := filepath.Dir(filepath.Dir(filepath.Dir(configPath))) // go up from .config/nd/config.yaml
	agentDir := filepath.Join(tmp, ".claude")
	symlinkPath := filepath.Join(agentDir, "skills", "greeting")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not found at %s: %v", symlinkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %s, got mode %v", symlinkPath, info.Mode())
	}
}
