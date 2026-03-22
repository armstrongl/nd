package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

// --- helpers ---

// makeSkillDir creates a skill directory with a SKILL.md file.
func makeSkillDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# Skill: "+name), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}

// makeFile creates a file with given content and returns its path.
func makeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", p, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// makeContextDir creates a context directory with CLAUDE.md and _meta.yaml.
func makeContextDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir context %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Context: "+name), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_meta.yaml"), []byte("description: "+name), 0o644); err != nil {
		t.Fatalf("write _meta.yaml: %v", err)
	}
	return dir
}

// makeTestHookDir creates a hook directory with hooks.json and optional scripts.
func makeTestHookDir(t *testing.T, parent, name string, hooksContent map[string]any, scripts map[string]string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir hook %s: %v", dir, err)
	}
	data, err := json.Marshal(hooksContent)
	if err != nil {
		t.Fatalf("marshal hooks.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), data, 0o644); err != nil {
		t.Fatalf("write hooks.json: %v", err)
	}
	for filename, content := range scripts {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o755); err != nil {
			t.Fatalf("write script %s: %v", filename, err)
		}
	}
	return dir
}

// readJSON reads and parses a JSON file into a map.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return m
}

// fileExists returns true if path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads a file and returns its content as a string.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// --- Test: Export skills only ---

func TestPluginExporter_SkillsOnly(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	skillDir := makeSkillDir(t, src, "go-test")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		Version:   "1.0.0",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetSkill, Name: "go-test", Path: skillDir, IsDir: true},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify skill directory was copied
	skillOut := filepath.Join(outDir, "skills", "go-test", "SKILL.md")
	if !fileExists(skillOut) {
		t.Fatalf("skill SKILL.md not found at %s", skillOut)
	}
	content := readFile(t, skillOut)
	if content != "# Skill: go-test" {
		t.Fatalf("skill content = %q, want %q", content, "# Skill: go-test")
	}

	// Verify plugin.json
	pj := readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"))
	if pj["name"] != "my-plugin" {
		t.Fatalf("plugin.json name = %v, want %q", pj["name"], "my-plugin")
	}

	// Verify result
	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
	if result.PluginName != "my-plugin" {
		t.Fatalf("PluginName = %q, want %q", result.PluginName, "my-plugin")
	}
	if result.PluginDir != outDir {
		t.Fatalf("PluginDir = %q, want %q", result.PluginDir, outDir)
	}
}

// --- Test: Export agents only ---

func TestPluginExporter_AgentsOnly(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "code-reviewer.md", "# Agent: code-reviewer")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "code-reviewer.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	agentOut := filepath.Join(outDir, "agents", "code-reviewer.md")
	if !fileExists(agentOut) {
		t.Fatalf("agent file not found at %s", agentOut)
	}
	content := readFile(t, agentOut)
	if content != "# Agent: code-reviewer" {
		t.Fatalf("agent content = %q", content)
	}

	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
}

// --- Test: Export commands only ---

func TestPluginExporter_CommandsOnly(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	cmdPath := makeFile(t, src, "deploy.md", "# Command: deploy")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetCommand, Name: "deploy.md", Path: cmdPath},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmdOut := filepath.Join(outDir, "commands", "deploy.md")
	if !fileExists(cmdOut) {
		t.Fatalf("command file not found at %s", cmdOut)
	}
	content := readFile(t, cmdOut)
	if content != "# Command: deploy" {
		t.Fatalf("command content = %q", content)
	}

	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
}

// --- Test: Export output-styles with outputStyles in plugin.json ---

func TestPluginExporter_OutputStylesWithPluginJSON(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	osPath := makeFile(t, src, "concise.md", "# Output Style: concise")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetOutputStyle, Name: "concise.md", Path: osPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists
	osOut := filepath.Join(outDir, "output-styles", "concise.md")
	if !fileExists(osOut) {
		t.Fatalf("output-style file not found at %s", osOut)
	}

	// Verify plugin.json has outputStyles field
	pj := readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"))
	osField, ok := pj["outputStyles"]
	if !ok {
		t.Fatal("plugin.json missing outputStyles field")
	}
	if osField != "./output-styles/" {
		t.Fatalf("outputStyles = %q, want %q", osField, "./output-styles/")
	}
}

// --- Test: No output-styles → no outputStyles in plugin.json ---

func TestPluginExporter_NoOutputStylesNoField(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "my-agent.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "my-agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pj := readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"))
	if _, ok := pj["outputStyles"]; ok {
		t.Fatal("plugin.json should NOT have outputStyles field when no output-style assets")
	}
}

// --- Test: Export hooks ---

func TestPluginExporter_Hooks(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")

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
	hookDir := makeTestHookDir(t, src, "lint-hook", hooksContent, map[string]string{
		"lint.sh": "#!/bin/bash\necho lint",
	})

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetHook, Name: "lint-hook", Path: hookDir, IsDir: true},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hooks.json was written
	hooksJSON := filepath.Join(outDir, "hooks", "hooks.json")
	if !fileExists(hooksJSON) {
		t.Fatalf("hooks.json not found at %s", hooksJSON)
	}

	// Verify scripts were copied under scripts/<hook-name>/
	scriptOut := filepath.Join(outDir, "scripts", "lint-hook", "lint.sh")
	if !fileExists(scriptOut) {
		t.Fatalf("script not found at %s", scriptOut)
	}

	// Verify the hooks.json has rewritten commands
	hj := readJSON(t, hooksJSON)
	hooks := hj["hooks"].(map[string]any)
	preToolUse := hooks["PreToolUse"].([]any)
	group := preToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)
	handler := innerHooks[0].(map[string]any)
	cmd := handler["command"].(string)
	expected := "${CLAUDE_PLUGIN_ROOT}/scripts/lint-hook/lint.sh --strict"
	if cmd != expected {
		t.Fatalf("hooks.json command = %q, want %q", cmd, expected)
	}

	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
}

// --- Test: Export rules (file) → extras ---

func TestPluginExporter_RulesFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	rulePath := makeFile(t, src, "go-conventions.md", "# Rule: go conventions")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetRule, Name: "go-conventions.md", Path: rulePath},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Rule should be in extras/rules/
	ruleOut := filepath.Join(outDir, "extras", "rules", "go-conventions.md")
	if !fileExists(ruleOut) {
		t.Fatalf("rule file not found at %s", ruleOut)
	}
	content := readFile(t, ruleOut)
	if content != "# Rule: go conventions" {
		t.Fatalf("rule content = %q", content)
	}

	// Should be tracked as BundledExtras, not CopiedAssets
	if len(result.BundledExtras) != 1 {
		t.Fatalf("BundledExtras = %d, want 1", len(result.BundledExtras))
	}
	if len(result.CopiedAssets) != 0 {
		t.Fatalf("CopiedAssets = %d, want 0 (rules go to extras)", len(result.CopiedAssets))
	}

	// README should mention extras
	readme := readFile(t, filepath.Join(outDir, "README.md"))
	if !strings.Contains(readme, "Extras") {
		t.Fatal("README should contain Extras section for rules")
	}
	if !strings.Contains(readme, "go-conventions.md") {
		t.Fatal("README should list the rule file")
	}
}

// --- Test: Export rules (directory) ---

func TestPluginExporter_RulesDir(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	ruleDir := filepath.Join(src, "go-rules")
	os.MkdirAll(ruleDir, 0o755)
	makeFile(t, ruleDir, "naming.md", "# Naming rules")
	makeFile(t, ruleDir, "formatting.md", "# Formatting rules")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetRule, Name: "go-rules", Path: ruleDir, IsDir: true},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Rule directory should be under extras/rules/
	ruleOutDir := filepath.Join(outDir, "extras", "rules", "go-rules")
	if !fileExists(filepath.Join(ruleOutDir, "naming.md")) {
		t.Fatal("naming.md not found in extras/rules/go-rules/")
	}
	if !fileExists(filepath.Join(ruleOutDir, "formatting.md")) {
		t.Fatal("formatting.md not found in extras/rules/go-rules/")
	}

	if len(result.BundledExtras) != 1 {
		t.Fatalf("BundledExtras = %d, want 1", len(result.BundledExtras))
	}
}

// --- Test: Export context ---

func TestPluginExporter_Context(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	ctxDir := makeContextDir(t, src, "go-project-rules")
	// The path points to the CLAUDE.md file
	ctxFile := filepath.Join(ctxDir, "CLAUDE.md")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:        "my-plugin",
		Description: "Test plugin",
		OutputDir:   outDir,
		Assets: []AssetRef{
			{Type: nd.AssetContext, Name: "go-project-rules", Path: ctxFile},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Context should be under extras/context/
	ctxOut := filepath.Join(outDir, "extras", "context", "go-project-rules")
	if !fileExists(filepath.Join(ctxOut, "CLAUDE.md")) {
		t.Fatal("CLAUDE.md not found in extras/context/go-project-rules/")
	}
	if !fileExists(filepath.Join(ctxOut, "_meta.yaml")) {
		t.Fatal("_meta.yaml not found in extras/context/go-project-rules/")
	}

	if len(result.BundledExtras) != 1 {
		t.Fatalf("BundledExtras = %d, want 1", len(result.BundledExtras))
	}

	// README should mention context
	readme := readFile(t, filepath.Join(outDir, "README.md"))
	if !strings.Contains(readme, "Context") {
		t.Fatal("README should contain Context section for context assets")
	}
	if !strings.Contains(readme, "go-project-rules") {
		t.Fatal("README should list the context folder")
	}
}

// --- Test: No extras → no extras section in README ---

func TestPluginExporter_NoExtrasNoSection(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "helper.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "helper.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	readme := readFile(t, filepath.Join(outDir, "README.md"))
	if strings.Contains(readme, "Extras") {
		t.Fatal("README should NOT have Extras section when no extras are bundled")
	}
}

// --- Test: Mix of all types ---

func TestPluginExporter_MixedAssets(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)

	// Skill
	skillDir := makeSkillDir(t, src, "test-skill")
	// Agent
	agentPath := makeFile(t, src, "my-agent.md", "# Agent")
	// Command
	cmdPath := makeFile(t, src, "my-cmd.md", "# Command")
	// Output style
	osPath := makeFile(t, src, "brief.md", "# Output Style")
	// Hook
	hookContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo hi"},
					},
				},
			},
		},
	}
	hookDir := makeTestHookDir(t, src, "my-hook", hookContent, nil)
	// Rule
	rulePath := makeFile(t, src, "my-rule.md", "# Rule")
	// Context
	ctxDir := makeContextDir(t, src, "my-context")
	ctxFile := filepath.Join(ctxDir, "CLAUDE.md")

	outDir := filepath.Join(tmp, "output", "mixed-plugin")
	cfg := ExportConfig{
		Name:        "mixed-plugin",
		Version:     "2.0.0",
		Description: "A plugin with everything",
		Author:      Author{Name: "Test Author", Email: "test@example.com"},
		Homepage:    "https://example.com",
		Repository:  "https://github.com/test/repo",
		License:     "MIT",
		Keywords:    []string{"test", "mixed"},
		OutputDir:   outDir,
		Assets: []AssetRef{
			{Type: nd.AssetSkill, Name: "test-skill", Path: skillDir, IsDir: true},
			{Type: nd.AssetAgent, Name: "my-agent.md", Path: agentPath},
			{Type: nd.AssetCommand, Name: "my-cmd.md", Path: cmdPath},
			{Type: nd.AssetOutputStyle, Name: "brief.md", Path: osPath},
			{Type: nd.AssetHook, Name: "my-hook", Path: hookDir, IsDir: true},
			{Type: nd.AssetRule, Name: "my-rule.md", Path: rulePath},
			{Type: nd.AssetContext, Name: "my-context", Path: ctxFile},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check all locations exist
	checks := []string{
		filepath.Join(outDir, "skills", "test-skill", "SKILL.md"),
		filepath.Join(outDir, "agents", "my-agent.md"),
		filepath.Join(outDir, "commands", "my-cmd.md"),
		filepath.Join(outDir, "output-styles", "brief.md"),
		filepath.Join(outDir, "hooks", "hooks.json"),
		filepath.Join(outDir, "extras", "rules", "my-rule.md"),
		filepath.Join(outDir, "extras", "context", "my-context", "CLAUDE.md"),
		filepath.Join(outDir, ".claude-plugin", "plugin.json"),
		filepath.Join(outDir, "README.md"),
	}
	for _, p := range checks {
		if !fileExists(p) {
			t.Fatalf("expected file not found: %s", p)
		}
	}

	// 5 non-extra assets (skill, agent, command, output-style, hook)
	if len(result.CopiedAssets) != 5 {
		t.Fatalf("CopiedAssets = %d, want 5", len(result.CopiedAssets))
	}
	// 2 extras (rule, context)
	if len(result.BundledExtras) != 2 {
		t.Fatalf("BundledExtras = %d, want 2", len(result.BundledExtras))
	}
}

// --- Test: plugin.json contains all metadata fields ---

func TestPluginExporter_PluginJSONAllFields(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "full-meta")
	cfg := ExportConfig{
		Name:        "full-meta",
		Version:     "3.2.1",
		Description: "Full metadata test",
		Author:      Author{Name: "Jane Doe", Email: "jane@example.com"},
		Homepage:    "https://jane.dev",
		Repository:  "https://github.com/jane/plugin",
		License:     "Apache-2.0",
		Keywords:    []string{"go", "testing", "lint"},
		OutputDir:   outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pj := readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"))

	if pj["name"] != "full-meta" {
		t.Fatalf("name = %v", pj["name"])
	}
	if pj["version"] != "3.2.1" {
		t.Fatalf("version = %v", pj["version"])
	}
	if pj["description"] != "Full metadata test" {
		t.Fatalf("description = %v", pj["description"])
	}
	author := pj["author"].(map[string]any)
	if author["name"] != "Jane Doe" {
		t.Fatalf("author.name = %v", author["name"])
	}
	if author["email"] != "jane@example.com" {
		t.Fatalf("author.email = %v", author["email"])
	}
	if pj["homepage"] != "https://jane.dev" {
		t.Fatalf("homepage = %v", pj["homepage"])
	}
	if pj["repository"] != "https://github.com/jane/plugin" {
		t.Fatalf("repository = %v", pj["repository"])
	}
	if pj["license"] != "Apache-2.0" {
		t.Fatalf("license = %v", pj["license"])
	}
	keywords := pj["keywords"].([]any)
	if len(keywords) != 3 {
		t.Fatalf("keywords len = %d, want 3", len(keywords))
	}
}

// --- Test: plugin.json omits empty optional fields ---

func TestPluginExporter_PluginJSONOmitsEmpty(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "minimal-meta")
	cfg := ExportConfig{
		Name:      "minimal-meta",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read raw JSON to check absence of optional fields
	data, err := os.ReadFile(filepath.Join(outDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}
	raw := string(data)

	// These fields should be absent when empty
	for _, field := range []string{"homepage", "repository", "license", "keywords", "outputStyles"} {
		if strings.Contains(raw, `"`+field+`"`) {
			t.Fatalf("plugin.json should not contain %q when empty", field)
		}
	}

	// author should be absent when name is empty
	if strings.Contains(raw, `"author"`) {
		t.Fatalf("plugin.json should not contain author when author name is empty")
	}
}

// --- Test: Source path missing → warning ---

func TestPluginExporter_MissingSourceWarning(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	// Create one valid asset
	agentPath := makeFile(t, src, "valid-agent.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "valid-agent.md", Path: agentPath},
			{Type: nd.AssetSkill, Name: "missing-skill", Path: "/nonexistent/path/skill", IsDir: true},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a warning about the missing source
	if len(result.Warnings) == 0 {
		t.Fatal("expected at least one warning for missing source path")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "missing-skill") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning mentioning 'missing-skill', got: %v", result.Warnings)
	}

	// Valid asset should still be exported
	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
}

// --- Test: All assets missing → error ---

func TestPluginExporter_AllAssetsMissing(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetSkill, Name: "gone1", Path: "/nonexistent/1", IsDir: true},
			{Type: nd.AssetAgent, Name: "gone2.md", Path: "/nonexistent/2"},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err == nil {
		t.Fatal("expected error when all assets are missing")
	}
	if !strings.Contains(err.Error(), "all assets failed") {
		t.Fatalf("error = %q, want mention of 'all assets failed'", err.Error())
	}
}

// --- Test: Plugin-type asset → error ---

func TestPluginExporter_PluginTypeError(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetPlugin, Name: "some-plugin", Path: "/some/path", IsDir: true},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err == nil {
		t.Fatal("expected error for plugin-type asset")
	}
	if !strings.Contains(err.Error(), "plugins cannot be exported") {
		t.Fatalf("error = %q, want 'plugins cannot be exported'", err.Error())
	}
}

// --- Test: Output dir exists without Overwrite → error ---

func TestPluginExporter_OutputDirExistsNoOverwrite(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "existing-output")
	os.MkdirAll(outDir, 0o755)

	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Overwrite: false,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err == nil {
		t.Fatal("expected error when output dir exists and Overwrite is false")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want mention of 'already exists'", err.Error())
	}
}

// --- Test: Output dir exists with Overwrite → succeeds ---

func TestPluginExporter_OutputDirExistsWithOverwrite(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "existing-output")
	os.MkdirAll(outDir, 0o755)
	// Put a stale file to prove it gets cleaned
	makeFile(t, outDir, "stale.txt", "stale content")

	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Overwrite: true,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	result, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stale file should be gone
	if fileExists(filepath.Join(outDir, "stale.txt")) {
		t.Fatal("stale file should have been removed on overwrite")
	}

	// New content should exist
	if !fileExists(filepath.Join(outDir, "agents", "agent.md")) {
		t.Fatal("agent.md not found after overwrite")
	}

	if len(result.CopiedAssets) != 1 {
		t.Fatalf("CopiedAssets = %d, want 1", len(result.CopiedAssets))
	}
}

// --- Test: Version defaults to 1.0.0 ---

func TestPluginExporter_VersionDefault(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		// Version not set — should default to 1.0.0
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pj := readJSON(t, filepath.Join(outDir, ".claude-plugin", "plugin.json"))
	if pj["version"] != "1.0.0" {
		t.Fatalf("version = %v, want %q", pj["version"], "1.0.0")
	}
}

// --- Test: README generation ---

func TestPluginExporter_READMEContent(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)

	skillDir := makeSkillDir(t, src, "my-skill")
	agentPath := makeFile(t, src, "my-agent.md", "# Agent")
	rulePath := makeFile(t, src, "my-rule.md", "# Rule")

	outDir := filepath.Join(tmp, "output", "readme-plugin")
	cfg := ExportConfig{
		Name:        "readme-plugin",
		Description: "A test plugin for README generation",
		OutputDir:   outDir,
		Assets: []AssetRef{
			{Type: nd.AssetSkill, Name: "my-skill", Path: skillDir, IsDir: true},
			{Type: nd.AssetAgent, Name: "my-agent.md", Path: agentPath},
			{Type: nd.AssetRule, Name: "my-rule.md", Path: rulePath},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	readme := readFile(t, filepath.Join(outDir, "README.md"))

	// Title
	if !strings.Contains(readme, "# readme-plugin") {
		t.Fatal("README should contain plugin name as title")
	}

	// Description
	if !strings.Contains(readme, "A test plugin for README generation") {
		t.Fatal("README should contain description")
	}

	// Installation
	if !strings.Contains(readme, "/plugin install") {
		t.Fatal("README should contain installation instruction")
	}

	// Assets listing — should mention skills and agents sections
	if !strings.Contains(readme, "my-skill") {
		t.Fatal("README should list skill asset")
	}
	if !strings.Contains(readme, "my-agent.md") {
		t.Fatal("README should list agent asset")
	}

	// Extras section
	if !strings.Contains(readme, "Extras") {
		t.Fatal("README should have Extras section for rules")
	}
	if !strings.Contains(readme, "my-rule.md") {
		t.Fatal("README extras should mention the rule")
	}
}

// --- Test: Hook merging failure aborts export ---

func TestPluginExporter_HookMergingFailureAbortsExport(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "sources")
	os.MkdirAll(src, 0o755)

	// Create a valid agent so we have at least one non-hook asset
	agentPath := makeFile(t, src, "agent.md", "# Agent")

	// Create a hook dir with invalid hooks.json (unrecognized event name)
	hookDir := filepath.Join(src, "bad-hook")
	os.MkdirAll(hookDir, 0o755)
	os.WriteFile(filepath.Join(hookDir, "hooks.json"), []byte(`{"hooks": {"InvalidEvent": []}}`), 0o644)

	outDir := filepath.Join(tmp, "output", "my-plugin")
	cfg := ExportConfig{
		Name:      "my-plugin",
		OutputDir: outDir,
		Assets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md", Path: agentPath},
			{Type: nd.AssetHook, Name: "bad-hook", Path: hookDir, IsDir: true},
		},
	}

	exp := &PluginExporter{}
	_, err := exp.Export(cfg)
	if err == nil {
		t.Fatal("expected error when hook merging fails")
	}
	if !strings.Contains(err.Error(), "hook merging failed") {
		t.Fatalf("error = %q, want mention of 'hook merging failed'", err.Error())
	}
}

// --- Test: internal helper generatePluginJSON ---

func TestGeneratePluginJSON(t *testing.T) {
	cfg := ExportConfig{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      Author{Name: "Author", Email: "a@b.com"},
		Homepage:    "https://example.com",
		Repository:  "https://github.com/test/repo",
		License:     "MIT",
		Keywords:    []string{"test"},
	}

	data, err := generatePluginJSON(cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if m["name"] != "test-plugin" {
		t.Fatalf("name = %v", m["name"])
	}
	if m["outputStyles"] != "./output-styles/" {
		t.Fatalf("outputStyles = %v", m["outputStyles"])
	}

	// Without output styles
	data2, err := generatePluginJSON(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(data2), "outputStyles") {
		t.Fatal("should not contain outputStyles when hasOutputStyles is false")
	}
}

func TestGeneratePluginJSON_UsesIndent(t *testing.T) {
	cfg := ExportConfig{
		Name:    "indent-test",
		Version: "1.0.0",
	}
	data, err := generatePluginJSON(cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be pretty-printed with 2-space indent
	if !strings.Contains(string(data), "  ") {
		t.Fatal("plugin.json should use 2-space indentation")
	}
}

// --- Test: internal helper generateREADME ---

func TestGenerateREADME_NoExtras(t *testing.T) {
	cfg := ExportConfig{
		Name:        "my-plugin",
		Description: "My cool plugin",
	}
	result := &ExportResult{
		CopiedAssets: []AssetRef{
			{Type: nd.AssetSkill, Name: "my-skill"},
			{Type: nd.AssetAgent, Name: "my-agent.md"},
		},
		BundledExtras: nil,
	}
	readme := generateREADME(cfg, result)

	if !strings.Contains(readme, "# my-plugin") {
		t.Fatal("README should have title")
	}
	if !strings.Contains(readme, "My cool plugin") {
		t.Fatal("README should have description")
	}
	if strings.Contains(readme, "Extras") {
		t.Fatal("README should NOT have Extras section when no extras")
	}
}

func TestGenerateREADME_WithExtras(t *testing.T) {
	cfg := ExportConfig{
		Name: "extras-plugin",
	}
	result := &ExportResult{
		CopiedAssets: []AssetRef{
			{Type: nd.AssetAgent, Name: "agent.md"},
		},
		BundledExtras: []AssetRef{
			{Type: nd.AssetRule, Name: "go-conventions.md"},
			{Type: nd.AssetContext, Name: "go-project-rules"},
		},
	}
	readme := generateREADME(cfg, result)

	if !strings.Contains(readme, "Extras") {
		t.Fatal("README should have Extras section")
	}
	if !strings.Contains(readme, "go-conventions.md") {
		t.Fatal("README should mention rule in extras")
	}
	if !strings.Contains(readme, "go-project-rules") {
		t.Fatal("README should mention context in extras")
	}
	if !strings.Contains(readme, "manual deployment") {
		t.Fatal("README extras should mention manual deployment")
	}
}
