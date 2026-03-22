//go:build integration

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/export"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
)

// setupExportIntegrationEnv creates a realistic source directory with all asset
// types and returns the config path, source dir, and a temp root.
func setupExportIntegrationEnv(t *testing.T) (configPath, srcDir, tmpRoot string) {
	t.Helper()
	tmpRoot = t.TempDir()
	configDir := filepath.Join(tmpRoot, ".config", "nd")
	os.MkdirAll(configDir, 0o755)
	os.MkdirAll(filepath.Join(configDir, "state"), 0o755)
	configPath = filepath.Join(configDir, "config.yaml")

	srcDir = filepath.Join(tmpRoot, "test-source")

	// Skill directory
	os.MkdirAll(filepath.Join(srcDir, "skills", "greeting"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "skills", "greeting", "SKILL.md"), []byte("# Greeting Skill"), 0o644)

	// Agent file
	os.MkdirAll(filepath.Join(srcDir, "agents"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "agents", "code-reviewer.md"), []byte("# Code Reviewer Agent"), 0o644)

	// Command file
	os.MkdirAll(filepath.Join(srcDir, "commands"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "commands", "deploy.md"), []byte("# Deploy Command"), 0o644)

	// Output style file
	os.MkdirAll(filepath.Join(srcDir, "output-styles"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "output-styles", "concise.md"), []byte("# Concise Style"), 0o644)

	// Hook directory with hooks.json and script
	hookDir := filepath.Join(srcDir, "hooks", "pre-commit-lint")
	os.MkdirAll(hookDir, 0o755)
	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "lint.sh --strict"},
					},
				},
			},
		},
	}
	hooksData, _ := json.Marshal(hooksContent)
	os.WriteFile(filepath.Join(hookDir, "hooks.json"), hooksData, 0o644)
	os.WriteFile(filepath.Join(hookDir, "lint.sh"), []byte("#!/bin/bash\necho lint"), 0o755)

	// Rule file
	os.MkdirAll(filepath.Join(srcDir, "rules"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "rules", "go-conventions.md"), []byte("# Go Conventions"), 0o644)

	// Context directory with CLAUDE.md and _meta.yaml
	ctxDir := filepath.Join(srcDir, "context", "go-project-rules")
	os.MkdirAll(ctxDir, 0o755)
	os.WriteFile(filepath.Join(ctxDir, "CLAUDE.md"), []byte("# Go project rules"), 0o644)
	os.WriteFile(filepath.Join(ctxDir, "_meta.yaml"), []byte("description: Go project conventions"), 0o644)

	// Create agent deploy target dir (needed by source manager)
	agentDir := filepath.Join(tmpRoot, ".claude")
	os.MkdirAll(agentDir, 0o755)

	// Write config with source registered
	cfg := "version: 1\ndefault_scope: global\ndefault_agent: claude-code\nsymlink_strategy: absolute\nsources:\n  - id: test-source\n    type: local\n    path: " + srcDir + "\nagents:\n  - name: claude-code\n    global_dir: " + agentDir + "\n"
	os.WriteFile(configPath, []byte(cfg), 0o644)

	return configPath, srcDir, tmpRoot
}

func TestExportIntegration_FullPluginExport(t *testing.T) {
	configPath, _, tmpRoot := setupExportIntegrationEnv(t)
	outDir := filepath.Join(tmpRoot, "exported-plugin")

	// Export all asset types
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"export",
		"--name", "test-plugin",
		"--description", "Integration test plugin",
		"--version", "2.0.0",
		"--author", "Test Author",
		"--email", "test@example.com",
		"--license", "MIT",
		"--output", outDir,
		"--assets", "skills/greeting,agents/code-reviewer.md,commands/deploy.md,output-styles/concise.md,hooks/pre-commit-lint,rules/go-conventions.md",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("export failed: %v\nOutput: %s", err, out.String())
	}

	// 1. Verify .claude-plugin/plugin.json
	pluginJSONPath := filepath.Join(outDir, ".claude-plugin", "plugin.json")
	pluginData, err := os.ReadFile(pluginJSONPath)
	if err != nil {
		t.Fatalf("plugin.json not found: %v", err)
	}

	var pj map[string]any
	if err := json.Unmarshal(pluginData, &pj); err != nil {
		t.Fatalf("invalid plugin.json: %v", err)
	}
	if pj["name"] != "test-plugin" {
		t.Errorf("plugin.json name = %v, want %q", pj["name"], "test-plugin")
	}
	if pj["version"] != "2.0.0" {
		t.Errorf("plugin.json version = %v, want %q", pj["version"], "2.0.0")
	}
	author, ok := pj["author"].(map[string]any)
	if !ok || author["name"] != "Test Author" {
		t.Errorf("plugin.json author.name = %v", author)
	}
	if pj["license"] != "MIT" {
		t.Errorf("plugin.json license = %v, want %q", pj["license"], "MIT")
	}
	// outputStyles should be present (we exported output-styles)
	if pj["outputStyles"] != "./output-styles/" {
		t.Errorf("plugin.json outputStyles = %v, want %q", pj["outputStyles"], "./output-styles/")
	}

	// 2. Verify skills directory
	skillFile := filepath.Join(outDir, "skills", "greeting", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("skill SKILL.md missing: %v", err)
	}

	// 3. Verify agents file
	agentFile := filepath.Join(outDir, "agents", "code-reviewer.md")
	if _, err := os.Stat(agentFile); err != nil {
		t.Errorf("agent file missing: %v", err)
	}

	// 4. Verify commands file
	cmdFile := filepath.Join(outDir, "commands", "deploy.md")
	if _, err := os.Stat(cmdFile); err != nil {
		t.Errorf("command file missing: %v", err)
	}

	// 5. Verify output-styles file
	osFile := filepath.Join(outDir, "output-styles", "concise.md")
	if _, err := os.Stat(osFile); err != nil {
		t.Errorf("output-style file missing: %v", err)
	}

	// 6. Verify hooks/hooks.json with merged config
	hooksJSON := filepath.Join(outDir, "hooks", "hooks.json")
	hooksData, err := os.ReadFile(hooksJSON)
	if err != nil {
		t.Fatalf("hooks.json missing: %v", err)
	}
	var hj map[string]any
	if err := json.Unmarshal(hooksData, &hj); err != nil {
		t.Fatalf("invalid hooks.json: %v", err)
	}
	hooks := hj["hooks"].(map[string]any)
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("hooks.json missing PreToolUse event")
	}

	// 7. Verify scripts copied
	scriptFile := filepath.Join(outDir, "scripts", "pre-commit-lint", "lint.sh")
	if _, err := os.Stat(scriptFile); err != nil {
		t.Errorf("script lint.sh missing: %v", err)
	}

	// 8. Verify extras/rules
	ruleFile := filepath.Join(outDir, "extras", "rules", "go-conventions.md")
	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf("rule file missing: %v", err)
	}

	// 9. Verify README.md
	readme, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatalf("README.md missing: %v", err)
	}
	readmeStr := string(readme)
	if !strings.Contains(readmeStr, "test-plugin") {
		t.Error("README should mention plugin name")
	}
	if !strings.Contains(readmeStr, "Extras") {
		t.Error("README should have Extras section for rules")
	}
	if !strings.Contains(readmeStr, "go-conventions.md") {
		t.Error("README should list the rule file")
	}

	// 10. Verify oplog entry
	entries := readLogEntries(t, logDir(configPath))
	found := false
	for _, e := range entries {
		if e.Operation == oplog.OpExport {
			found = true
			if e.Succeeded < 1 {
				t.Errorf("oplog export entry succeeded = %d, want >= 1", e.Succeeded)
			}
			break
		}
	}
	if !found {
		t.Error("oplog should contain an export entry")
	}

	// 11. Verify human output
	outStr := out.String()
	if !strings.Contains(outStr, "Exported plugin") {
		t.Errorf("output should contain 'Exported plugin', got: %s", outStr)
	}
}

func TestExportIntegration_JSONOutput(t *testing.T) {
	configPath, _, tmpRoot := setupExportIntegrationEnv(t)
	outDir := filepath.Join(tmpRoot, "json-export")

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"--json",
		"export",
		"--name", "json-plugin",
		"--output", outDir,
		"--assets", "skills/greeting",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("export failed: %v\nOutput: %s", err, out.String())
	}

	// Parse JSON output
	var resp struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nRaw: %s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("JSON status = %q, want %q", resp.Status, "ok")
	}
	if resp.Data["pluginName"] != "json-plugin" {
		t.Errorf("JSON pluginName = %v", resp.Data["pluginName"])
	}
}

func TestExportIntegration_DryRun(t *testing.T) {
	configPath, _, tmpRoot := setupExportIntegrationEnv(t)
	outDir := filepath.Join(tmpRoot, "dryrun-export")

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"--dry-run",
		"export",
		"--name", "dryrun-plugin",
		"--output", outDir,
		"--assets", "skills/greeting,agents/code-reviewer.md",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("export dry-run failed: %v", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "[dry-run]") {
		t.Errorf("dry-run output should contain '[dry-run]', got: %s", outStr)
	}
	if !strings.Contains(outStr, "dryrun-plugin") {
		t.Errorf("dry-run output should mention plugin name")
	}

	// Output directory should NOT exist
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Error("dry-run should not create output directory")
	}
}

func TestExportIntegration_MarketplaceFromExportedPlugin(t *testing.T) {
	configPath, _, tmpRoot := setupExportIntegrationEnv(t)

	// Step 1: Export a plugin
	pluginDir := filepath.Join(tmpRoot, "my-plugin")

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{
		"--config", configPath,
		"export",
		"--name", "my-plugin",
		"--description", "Test plugin for marketplace",
		"--version", "1.2.0",
		"--author", "Test Author",
		"--output", pluginDir,
		"--assets", "skills/greeting,agents/code-reviewer.md",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("export failed: %v\nOutput: %s", err, out.String())
	}

	// Step 2: Generate marketplace from the exported plugin
	mktDir := filepath.Join(tmpRoot, "my-marketplace")

	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	var mktOut bytes.Buffer
	rootCmd2.SetOut(&mktOut)
	rootCmd2.SetErr(&mktOut)
	rootCmd2.SetArgs([]string{
		"--config", configPath,
		"export", "marketplace",
		"--name", "my-marketplace",
		"--owner", "Test Owner",
		"--email", "owner@example.com",
		"--plugins", pluginDir,
		"--output", mktDir,
	})

	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("marketplace generation failed: %v\nOutput: %s", err, mktOut.String())
	}

	// Verify marketplace.json
	mktJSON := filepath.Join(mktDir, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(mktJSON)
	if err != nil {
		t.Fatalf("marketplace.json not found: %v", err)
	}

	var mj map[string]any
	if err := json.Unmarshal(data, &mj); err != nil {
		t.Fatalf("invalid marketplace.json: %v", err)
	}

	if mj["name"] != "my-marketplace" {
		t.Errorf("marketplace name = %v, want %q", mj["name"], "my-marketplace")
	}
	owner, ok := mj["owner"].(map[string]any)
	if !ok || owner["name"] != "Test Owner" {
		t.Errorf("marketplace owner = %v", owner)
	}
	plugins, ok := mj["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("marketplace plugins count = %v", plugins)
	}
	plugin := plugins[0].(map[string]any)
	if plugin["name"] != "my-plugin" {
		t.Errorf("marketplace plugin name = %v", plugin["name"])
	}
	if plugin["source"] != "./plugins/my-plugin" {
		t.Errorf("marketplace plugin source = %v, want %q", plugin["source"], "./plugins/my-plugin")
	}
	if plugin["version"] != "1.2.0" {
		t.Errorf("marketplace plugin version = %v, want %q", plugin["version"], "1.2.0")
	}

	// Verify plugin was copied into marketplace
	copiedPluginJSON := filepath.Join(mktDir, "plugins", "my-plugin", ".claude-plugin", "plugin.json")
	if _, err := os.Stat(copiedPluginJSON); err != nil {
		t.Errorf("plugin not copied into marketplace: %v", err)
	}

	// Verify oplog entry for marketplace
	entries := readLogEntries(t, logDir(configPath))
	found := false
	for _, e := range entries {
		if e.Operation == oplog.OpExportMarketplace {
			found = true
			break
		}
	}
	if !found {
		t.Error("oplog should contain an export-marketplace entry")
	}

	// Verify human output
	mktOutStr := mktOut.String()
	if !strings.Contains(mktOutStr, "Generated marketplace") {
		t.Errorf("marketplace output should contain 'Generated marketplace', got: %s", mktOutStr)
	}
}

func TestExportIntegration_PluginExporterDirectly(t *testing.T) {
	// Test the export package directly (not via CLI) for full structural verification
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "direct-export")

	// Create source assets
	srcDir := filepath.Join(tmp, "sources")

	// Skill
	skillDir := filepath.Join(srcDir, "skills", "test-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test Skill\nA greeting skill."), 0o644)

	// Agent
	os.MkdirAll(filepath.Join(srcDir, "agents"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "agents", "my-agent.md"), []byte("# My Agent"), 0o644)

	// Hook with script
	hookDir := filepath.Join(srcDir, "hooks", "format-hook")
	os.MkdirAll(hookDir, 0o755)
	hookContent := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write|Edit",
					"hooks": []any{
						map[string]any{"type": "command", "command": "format.sh"},
					},
				},
			},
		},
	}
	hooksData, _ := json.Marshal(hookContent)
	os.WriteFile(filepath.Join(hookDir, "hooks.json"), hooksData, 0o644)
	os.WriteFile(filepath.Join(hookDir, "format.sh"), []byte("#!/bin/bash\necho format"), 0o755)

	// Rule
	os.MkdirAll(filepath.Join(srcDir, "rules"), 0o755)
	os.WriteFile(filepath.Join(srcDir, "rules", "naming.md"), []byte("# Naming Rules"), 0o644)

	// Context
	ctxDir := filepath.Join(srcDir, "context", "go-rules")
	os.MkdirAll(ctxDir, 0o755)
	os.WriteFile(filepath.Join(ctxDir, "CLAUDE.md"), []byte("# Go Rules Context"), 0o644)
	os.WriteFile(filepath.Join(ctxDir, "_meta.yaml"), []byte("description: Go project rules"), 0o644)

	cfg := export.ExportConfig{
		Name:        "full-plugin",
		Version:     "3.0.0",
		Description: "Full export integration test",
		Author:      export.Author{Name: "Integration Tester", Email: "test@test.com"},
		License:     "Apache-2.0",
		OutputDir:   outDir,
		Assets: []export.AssetRef{
			{Type: nd.AssetSkill, Name: "test-skill", Path: skillDir, IsDir: true},
			{Type: nd.AssetAgent, Name: "my-agent.md", Path: filepath.Join(srcDir, "agents", "my-agent.md")},
			{Type: nd.AssetHook, Name: "format-hook", Path: hookDir, IsDir: true},
			{Type: nd.AssetRule, Name: "naming.md", Path: filepath.Join(srcDir, "rules", "naming.md")},
			{Type: nd.AssetContext, Name: "go-rules", Path: filepath.Join(ctxDir, "CLAUDE.md")},
		},
	}

	exporter := &export.PluginExporter{}
	result, err := exporter.Export(cfg)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{
		".claude-plugin/plugin.json",
		"README.md",
		"skills/test-skill/SKILL.md",
		"agents/my-agent.md",
		"hooks/hooks.json",
		"scripts/format-hook/format.sh",
		"extras/rules/naming.md",
		"extras/context/go-rules/CLAUDE.md",
		"extras/context/go-rules/_meta.yaml",
	}
	for _, rel := range expectedFiles {
		p := filepath.Join(outDir, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file %q missing: %v", rel, err)
		}
	}

	// Verify hooks.json has rewritten script path
	hooksJSONData, _ := os.ReadFile(filepath.Join(outDir, "hooks", "hooks.json"))
	if !strings.Contains(string(hooksJSONData), "${CLAUDE_PLUGIN_ROOT}/scripts/format-hook/format.sh") {
		t.Error("hooks.json should contain rewritten script path")
	}

	// Verify result counts
	if len(result.CopiedAssets) != 3 { // skill + agent + hook
		t.Errorf("CopiedAssets = %d, want 3", len(result.CopiedAssets))
	}
	if len(result.BundledExtras) != 2 { // rule + context
		t.Errorf("BundledExtras = %d, want 2", len(result.BundledExtras))
	}
}
