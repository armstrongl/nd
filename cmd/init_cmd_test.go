package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/output"
)

// testInitAgent returns an agent pointing at a temp directory for safe testing.
func testInitAgent(t *testing.T, tmp string) *agent.Agent {
	t.Helper()
	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)
	return &agent.Agent{
		Name:       "claude-code",
		GlobalDir:  agentDir,
		ProjectDir: ".claude",
		Detected:   true,
		InPath:     true,
	}
}

func TestInitCmd_WithYes(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Initialized") {
		t.Errorf("expected 'Initialized' in output, got: %s", got)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestInitCmd_WithYes_DeploysBuiltinAssets(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")
	agentDir := filepath.Join(tmp, ".claude")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()

	// Verify "Deployed N built-in asset(s)" message
	if !strings.Contains(got, "Deployed") || !strings.Contains(got, "built-in asset") {
		t.Errorf("expected deploy message in output, got: %s", got)
	}

	// Verify symlinks exist in the test agent dir for skills
	skillsDir := filepath.Join(agentDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("expected skills dir at %s: %v", skillsDir, err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one skill symlink deployed")
	}

	// Verify at least one skill is a symlink
	foundSymlink := false
	for _, e := range entries {
		linkPath := filepath.Join(skillsDir, e.Name())
		if info, lerr := os.Lstat(linkPath); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			foundSymlink = true
			break
		}
	}
	if !foundSymlink {
		t.Error("expected at least one symlink in skills dir")
	}

	// Verify commands dir has symlinks too
	commandsDir := filepath.Join(agentDir, "commands")
	cmdEntries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("expected commands dir at %s: %v", commandsDir, err)
	}
	if len(cmdEntries) == 0 {
		t.Error("expected at least one command symlink deployed")
	}

	// Verify agents dir has symlinks
	agentsSubDir := filepath.Join(agentDir, "agents")
	agentEntries, err := os.ReadDir(agentsSubDir)
	if err != nil {
		t.Fatalf("expected agents dir at %s: %v", agentsSubDir, err)
	}
	if len(agentEntries) == 0 {
		t.Error("expected at least one agent symlink deployed")
	}

	// Verify deployment state file was written
	statePath := filepath.Join(tmp, ".config", "nd", "state", "deployments.yaml")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("deployment state file not created: %v", err)
	}
}

func TestInitCmd_DirectoryStructure(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "nd")
	configPath := filepath.Join(configDir, "config.yaml")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify expected directories exist
	expectedDirs := []string{
		"profiles",
		"snapshots",
		"state",
	}
	for _, dir := range expectedDirs {
		path := filepath.Join(configDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %q not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", dir)
		}
	}
}

func TestInitCmd_AlreadyExists(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
}

func TestInitCmd_JSON(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "--json", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}

	// JSON mode with --yes should also deploy builtins
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map, got %T", resp.Data)
	}
	if _, hasDeployed := data["builtin_deployed"]; !hasDeployed {
		t.Errorf("expected builtin_deployed in JSON data, got: %v", data)
	}
}

func TestInitCmd_JSON_IncludesBuiltinDeployCount(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "--json", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map, got %T", resp.Data)
	}

	count, ok := data["builtin_deployed"]
	if !ok {
		t.Fatal("expected builtin_deployed in JSON response")
	}

	// JSON numbers decode as float64
	countFloat, ok := count.(float64)
	if !ok {
		t.Fatalf("expected builtin_deployed to be number, got %T", count)
	}
	if countFloat < 1 {
		t.Errorf("expected at least 1 deployed asset, got %v", countFloat)
	}
}
