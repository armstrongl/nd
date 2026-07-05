package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/output"
)

// testInitAgent returns an agent pointing at a temp directory for safe testing.
func testInitAgent(t *testing.T, tmp string) *agent.Agent {
	t.Helper()
	agentDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(agentDir, 0o755)
	return &agent.Agent{
		Name:           "claude-code",
		GlobalDir:      agentDir,
		ProjectDir:     ".claude",
		Binary:         "claude",
		SupportedTypes: nd.DeployableAssetTypes(),
		VersionPattern: `(?i)claude`,
		Detected:       true,
		InPath:         true,
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

func TestInitCmd_Interactive_DefaultYesDeploys(t *testing.T) {
	fakeTerminal(t, true)
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")
	agentDir := filepath.Join(tmp, ".claude")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetIn(strings.NewReader("\n")) // Enter => default Yes
	rootCmd.SetArgs([]string{"--config", configPath, "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[Y/n/list]") {
		t.Errorf("expected interactive prompt '[Y/n/list]', got: %s", got)
	}
	if !strings.Contains(got, "Deployed") {
		t.Errorf("expected deploy after Enter (default Yes), got: %s", got)
	}

	skillsDir := filepath.Join(agentDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("expected skills dir at %s: %v", skillsDir, err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one skill symlink deployed")
	}
}

func TestInitCmd_Interactive_ListThenDeploys(t *testing.T) {
	fakeTerminal(t, true)
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")
	agentDir := filepath.Join(tmp, ".claude")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetIn(strings.NewReader("list\n\n")) // list, then Enter => deploy
	rootCmd.SetArgs([]string{"--config", configPath, "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()

	// The list action prints each asset as "<type>/<name>"; built-in assets
	// include skills, so at least one "skills/" line must appear.
	if !strings.Contains(got, "skills/") {
		t.Errorf("expected 'list' to print assets as '<type>/<name>', got: %s", got)
	}
	// Listing must happen before the deploy summary (no deploy until Y/n given).
	listIdx := strings.Index(got, "skills/")
	deployIdx := strings.Index(got, "Deployed")
	if deployIdx < 0 {
		t.Fatalf("expected deploy after list+Enter, got: %s", got)
	}
	if listIdx < 0 || listIdx > deployIdx {
		t.Errorf("expected asset list before deploy summary, got: %s", got)
	}

	skillsDir := filepath.Join(agentDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("expected skills dir at %s: %v", skillsDir, err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one skill symlink deployed")
	}
}

func TestInitCmd_JSON_IncludesBuiltinAssets(t *testing.T) {
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

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map, got %T", resp.Data)
	}

	// The existing count key must be preserved.
	if _, ok := data["builtin_deployed"]; !ok {
		t.Error("expected builtin_deployed count to be preserved")
	}

	// builtin_assets must be a non-empty array of {name, type} objects.
	assetsRaw, ok := data["builtin_assets"]
	if !ok {
		t.Fatal("expected builtin_assets in JSON data")
	}
	arr, ok := assetsRaw.([]interface{})
	if !ok {
		t.Fatalf("expected builtin_assets to be array, got %T", assetsRaw)
	}
	if len(arr) == 0 {
		t.Fatal("expected builtin_assets to be non-empty")
	}
	for i, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			t.Fatalf("builtin_assets[%d] not an object: %T", i, el)
		}
		if name, ok := obj["name"].(string); !ok || name == "" {
			t.Errorf("builtin_assets[%d] missing string name, got: %v", i, obj)
		}
		if typ, ok := obj["type"].(string); !ok || typ == "" {
			t.Errorf("builtin_assets[%d] missing string type, got: %v", i, obj)
		}
	}

	// The count must equal the array length.
	if cnt, ok := data["builtin_deployed"].(float64); ok {
		if int(cnt) != len(arr) {
			t.Errorf("builtin_deployed (%v) != len(builtin_assets) (%d)", cnt, len(arr))
		}
	}
}

func TestInitCmd_NonInteractiveNoYes_Skips(t *testing.T) {
	fakeTerminal(t, false)
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")
	agentDir := filepath.Join(tmp, ".claude")

	app := &App{initAgent: testInitAgent(t, tmp)}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetIn(strings.NewReader("")) // piped stdin, non-TTY
	rootCmd.SetArgs([]string{"--config", configPath, "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Skipped.") {
		t.Errorf("expected 'Skipped.' hint on non-TTY without --yes, got: %s", got)
	}
	if strings.Contains(got, "Deployed") {
		t.Errorf("expected no deploy on non-TTY without --yes, got: %s", got)
	}

	// Nothing should have been deployed into the agent dir.
	skillsDir := filepath.Join(agentDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil && len(entries) > 0 {
		t.Errorf("expected no deployed skills on skip, found %d", len(entries))
	}
}

// testInitRegistry creates a registry where claude-code is detected and copilot is not.
// Agent GlobalDirs are redirected to tmp-based paths for safe testing.
func testInitRegistry(t *testing.T, tmp string) *agent.Registry {
	t.Helper()
	claudeDir := filepath.Join(tmp, ".claude")
	copilotDir := filepath.Join(tmp, ".copilot")
	os.MkdirAll(claudeDir, 0o755)

	cfg := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: claudeDir},
			{Name: "copilot", GlobalDir: copilotDir},
		},
	}
	reg := agent.New(cfg)

	// Override the registry's lookup functions so claude-code is detected
	// and copilot is not.
	reg.SetLookPath(func(name string) (string, error) {
		if name == "claude" {
			return "/usr/local/bin/claude", nil
		}
		return "", fmt.Errorf("not found: %s", name)
	})
	reg.SetStat(func(name string) (os.FileInfo, error) {
		if name == claudeDir {
			return os.Stat(claudeDir) // real stat on the dir we created
		}
		return nil, os.ErrNotExist
	})
	// Skip binary verification since we use fake lookPath results
	reg.SetRunCommand(nil)
	return reg
}

// testInitRegistryBothDetected creates a registry where both agents are detected.
// Agent GlobalDirs are redirected to tmp-based paths for safe testing.
func testInitRegistryBothDetected(t *testing.T, tmp string) *agent.Registry {
	t.Helper()
	claudeDir := filepath.Join(tmp, ".claude")
	copilotDir := filepath.Join(tmp, ".copilot")
	os.MkdirAll(claudeDir, 0o755)
	os.MkdirAll(copilotDir, 0o755)

	cfg := config.Config{
		Version:         1,
		DefaultScope:    nd.ScopeGlobal,
		DefaultAgent:    "claude-code",
		SymlinkStrategy: nd.SymlinkAbsolute,
		Agents: []config.AgentOverride{
			{Name: "claude-code", GlobalDir: claudeDir},
			{Name: "copilot", GlobalDir: copilotDir},
		},
	}
	reg := agent.New(cfg)

	reg.SetLookPath(func(name string) (string, error) {
		if name == "claude" || name == "copilot" {
			return "/usr/local/bin/" + name, nil
		}
		return "", fmt.Errorf("not found: %s", name)
	})
	reg.SetStat(func(name string) (os.FileInfo, error) {
		if name == claudeDir || name == copilotDir {
			return os.Stat(name)
		}
		return nil, os.ErrNotExist
	})
	// Skip binary verification since we use fake lookPath results
	reg.SetRunCommand(nil)
	return reg
}

func TestInitCmd_DisplaysAgentDetection(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	reg := testInitRegistry(t, tmp)

	app := &App{
		initAgent:    testInitAgent(t, tmp),
		initRegistry: reg,
	}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()

	// Must display "Detected agents:" line
	if !strings.Contains(got, "Detected agents:") {
		t.Errorf("expected 'Detected agents:' in output, got: %s", got)
	}

	// claude-code should be marked as detected (checkmark)
	if !strings.Contains(got, "claude-code") {
		t.Errorf("expected 'claude-code' in agent detection output, got: %s", got)
	}

	// copilot should be listed (with cross mark since not detected)
	if !strings.Contains(got, "copilot") {
		t.Errorf("expected 'copilot' in agent detection output, got: %s", got)
	}
}

func TestInitCmd_DisplaysAgentDetection_BothDetected(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	reg := testInitRegistryBothDetected(t, tmp)

	app := &App{
		initAgent:    testInitAgent(t, tmp),
		initRegistry: reg,
	}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()

	// Both agents should show checkmarks
	if !strings.Contains(got, "Detected agents:") {
		t.Errorf("expected 'Detected agents:' in output, got: %s", got)
	}

	// The detection line should appear BEFORE the deploy message
	detIdx := strings.Index(got, "Detected agents:")
	deployIdx := strings.Index(got, "Deployed")
	if detIdx >= 0 && deployIdx >= 0 && detIdx > deployIdx {
		t.Errorf("expected agent detection to appear before deploy message")
	}
}

func TestInitCmd_AgentDetection_QuietMode(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	reg := testInitRegistry(t, tmp)

	app := &App{
		initAgent:    testInitAgent(t, tmp),
		initRegistry: reg,
	}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "--quiet", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()

	// In quiet mode, no detection display
	if strings.Contains(got, "Detected agents:") {
		t.Errorf("expected no agent detection in quiet mode, got: %s", got)
	}
}

func TestInitCmd_AgentDetection_JSONMode(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	reg := testInitRegistryBothDetected(t, tmp)

	app := &App{
		initAgent:    testInitAgent(t, tmp),
		initRegistry: reg,
	}
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

	// JSON response should include agents_detected
	agentsRaw, ok := data["agents_detected"]
	if !ok {
		t.Fatalf("expected agents_detected in JSON data, got: %v", data)
	}

	agents, ok := agentsRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("expected agents_detected to be map, got %T", agentsRaw)
	}

	// Both agents should be true
	if v, ok := agents["claude-code"]; !ok || v != true {
		t.Errorf("expected claude-code: true in agents_detected, got: %v", agents)
	}
	if v, ok := agents["copilot"]; !ok || v != true {
		t.Errorf("expected copilot: true in agents_detected, got: %v", agents)
	}
}

func TestInitCmd_DeployBuiltinAssets_UsesRegistry(t *testing.T) {
	// Verify that when initRegistry is provided, deployBuiltinAssets uses it
	// instead of creating a fresh registry with hardcoded config.
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, ".config", "nd", "config.yaml")

	claudeDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(claudeDir, 0o755)

	reg := testInitRegistry(t, tmp)

	app := &App{
		initRegistry: reg,
	}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "init"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	// Should deploy to claude-code (the default agent from the registry)
	if !strings.Contains(got, "Deployed") {
		t.Errorf("expected deploy message, got: %s", got)
	}

	// Verify symlinks in the claude dir
	skillsDir := filepath.Join(claudeDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("expected skills dir at %s: %v", skillsDir, err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one skill symlink deployed")
	}
}
