package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all integration tests
	tmp, err := os.MkdirTemp("", "nd-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "nd")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../.")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type runResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func runND(t *testing.T, args ...string) runResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run nd: %v", err)
		}
	}

	return runResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

func setupIntegrationEnv(t *testing.T) (configPath string, srcDir string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath = filepath.Join(configDir, "config.yaml")

	srcDir = filepath.Join(tmp, "my-source")
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "skills", "greeting", "SKILL.md"), []byte("# Greeting"), 0o644)
	os.MkdirAll(filepath.Join(srcDir, "commands"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "commands", "hello.md"), []byte("# Hello"), 0o644)

	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)

	cfg := strings.Join([]string{
		"version: 1",
		"default_scope: global",
		"default_agent: claude-code",
		"symlink_strategy: absolute",
		"sources:",
		"  - id: my-source",
		"    type: local",
		"    path: " + srcDir,
		"agents:",
		"  - name: claude-code",
		"    global_dir: " + agentDir,
	}, "\n") + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	return configPath, srcDir
}
