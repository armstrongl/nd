package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestPlugin creates a plugin directory with a valid .claude-plugin/plugin.json
// and a sample skill directory. Returns the absolute path to the plugin directory.
func createTestPlugin(t *testing.T, dir, name string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0o755); err != nil {
		t.Fatalf("create .claude-plugin dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginDir, "skills", "test-skill"), 0o755); err != nil {
		t.Fatalf("create skills dir: %v", err)
	}
	pluginJSON := map[string]any{
		"name":        name,
		"version":     "1.0.0",
		"description": name + " plugin",
	}
	data, err := json.MarshalIndent(pluginJSON, "", "  ")
	if err != nil {
		t.Fatalf("marshal plugin.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), data, 0o644); err != nil {
		t.Fatalf("write plugin.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "skills", "test-skill", "SKILL.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return pluginDir
}

// readMarketplaceJSON reads and parses the marketplace.json from the output directory.
func readMarketplaceJSON(t *testing.T, outputDir string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outputDir, ".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatalf("read marketplace.json: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse marketplace.json: %v", err)
	}
	return parsed
}

func TestMarketplaceGenerator_SinglePlugin(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir, Description: "Plugin A description", Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	result, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify result fields
	if result.MarketplaceDir != outDir {
		t.Fatalf("MarketplaceDir = %q, want %q", result.MarketplaceDir, outDir)
	}
	if result.MarketplaceName != "my-marketplace" {
		t.Fatalf("MarketplaceName = %q, want %q", result.MarketplaceName, "my-marketplace")
	}
	if result.PluginCount != 1 {
		t.Fatalf("PluginCount = %d, want 1", result.PluginCount)
	}

	// Verify marketplace.json exists and has correct structure
	mj := readMarketplaceJSON(t, outDir)
	if mj["name"] != "my-marketplace" {
		t.Fatalf("marketplace.json name = %v, want %q", mj["name"], "my-marketplace")
	}

	// Verify plugin was copied
	copiedPluginJSON := filepath.Join(outDir, "plugins", "plugin-a", ".claude-plugin", "plugin.json")
	if _, err := os.Stat(copiedPluginJSON); err != nil {
		t.Fatalf("plugin.json not copied to output: %v", err)
	}
	copiedSkill := filepath.Join(outDir, "plugins", "plugin-a", "skills", "test-skill", "SKILL.md")
	if _, err := os.Stat(copiedSkill); err != nil {
		t.Fatalf("SKILL.md not copied to output: %v", err)
	}
}

func TestMarketplaceGenerator_MultiplePlugins(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDirA := createTestPlugin(t, srcDir, "plugin-a")
	pluginDirB := createTestPlugin(t, srcDir, "plugin-b")

	cfg := MarketplaceConfig{
		Name:  "multi-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDirA, Description: "Plugin A", Version: "1.0.0"},
			{Name: "plugin-b", Source: pluginDirB, Description: "Plugin B", Version: "2.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	result, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PluginCount != 2 {
		t.Fatalf("PluginCount = %d, want 2", result.PluginCount)
	}

	// Verify both plugins referenced in marketplace.json
	mj := readMarketplaceJSON(t, outDir)
	plugins, ok := mj["plugins"].([]any)
	if !ok {
		t.Fatal("marketplace.json missing plugins array")
	}
	if len(plugins) != 2 {
		t.Fatalf("plugins array length = %d, want 2", len(plugins))
	}

	// Verify both plugin directories were copied
	for _, name := range []string{"plugin-a", "plugin-b"} {
		pj := filepath.Join(outDir, "plugins", name, ".claude-plugin", "plugin.json")
		if _, err := os.Stat(pj); err != nil {
			t.Fatalf("plugin %s not copied: %v", name, err)
		}
	}
}

func TestMarketplaceGenerator_PluginDirMissingPluginJSON(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	// Create a plugin directory without plugin.json
	badDir := filepath.Join(srcDir, "bad-plugin")
	if err := os.MkdirAll(filepath.Join(badDir, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No plugin.json written

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "bad-plugin", Source: badDir, Description: "Bad plugin", Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected error for missing plugin.json")
	}
	if !strings.Contains(err.Error(), "bad-plugin") {
		t.Fatalf("error should name the plugin directory: %v", err)
	}
	if !strings.Contains(err.Error(), "plugin.json") {
		t.Fatalf("error should mention plugin.json: %v", err)
	}
}

func TestMarketplaceGenerator_PluginDirMalformedPluginJSON(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	// Create a plugin directory with malformed plugin.json
	badDir := filepath.Join(srcDir, "malformed-plugin")
	if err := os.MkdirAll(filepath.Join(badDir, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, ".claude-plugin", "plugin.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "malformed-plugin", Source: badDir, Description: "Malformed plugin", Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected error for malformed plugin.json")
	}
	if !strings.Contains(err.Error(), "malformed-plugin") {
		t.Fatalf("error should name the plugin: %v", err)
	}
}

func TestMarketplaceGenerator_PluginJSONMissingName(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	// Create a plugin directory with plugin.json that has no "name" field
	noNameDir := filepath.Join(srcDir, "no-name-plugin")
	if err := os.MkdirAll(filepath.Join(noNameDir, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	pluginJSON := map[string]any{
		"version":     "1.0.0",
		"description": "Plugin without name",
	}
	data, _ := json.MarshalIndent(pluginJSON, "", "  ")
	if err := os.WriteFile(filepath.Join(noNameDir, ".claude-plugin", "plugin.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "no-name-plugin", Source: noNameDir, Description: "No name", Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected error for plugin.json without name")
	}
	if !strings.Contains(err.Error(), "no-name-plugin") {
		t.Fatalf("error should name the plugin: %v", err)
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("error should mention missing name: %v", err)
	}
}

func TestMarketplaceGenerator_MarketplaceJSONFields(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:        "my-marketplace",
		Description: "A test marketplace",
		Owner:       Author{Name: "Jane Doe", Email: "jane@example.com"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir, Description: "Plugin A description", Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)

	// Check top-level name
	if mj["name"] != "my-marketplace" {
		t.Fatalf("name = %v, want %q", mj["name"], "my-marketplace")
	}

	// Check owner
	owner, ok := mj["owner"].(map[string]any)
	if !ok {
		t.Fatal("marketplace.json missing owner object")
	}
	if owner["name"] != "Jane Doe" {
		t.Fatalf("owner.name = %v, want %q", owner["name"], "Jane Doe")
	}

	// Check plugins array
	plugins, ok := mj["plugins"].([]any)
	if !ok {
		t.Fatal("marketplace.json missing plugins array")
	}
	if len(plugins) != 1 {
		t.Fatalf("plugins length = %d, want 1", len(plugins))
	}

	plugin := plugins[0].(map[string]any)
	if plugin["name"] != "plugin-a" {
		t.Fatalf("plugin name = %v, want %q", plugin["name"], "plugin-a")
	}
	if plugin["description"] != "Plugin A description" {
		t.Fatalf("plugin description = %v, want %q", plugin["description"], "Plugin A description")
	}
	if plugin["version"] != "1.0.0" {
		t.Fatalf("plugin version = %v, want %q", plugin["version"], "1.0.0")
	}
}

func TestMarketplaceGenerator_RelativeSourcePaths(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDirA := createTestPlugin(t, srcDir, "plugin-a")
	pluginDirB := createTestPlugin(t, srcDir, "plugin-b")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDirA},
			{Name: "plugin-b", Source: pluginDirB},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)
	plugins := mj["plugins"].([]any)

	for _, p := range plugins {
		pe := p.(map[string]any)
		name := pe["name"].(string)
		source := pe["source"].(string)
		expected := "./plugins/" + name
		if source != expected {
			t.Fatalf("plugin %q source = %q, want %q", name, source, expected)
		}
	}
}

func TestMarketplaceGenerator_MetadataDescriptionIncluded(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:        "my-marketplace",
		Description: "Generated by nd",
		Owner:       Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)

	metadata, ok := mj["metadata"].(map[string]any)
	if !ok {
		t.Fatal("marketplace.json missing metadata object when description is set")
	}
	if metadata["description"] != "Generated by nd" {
		t.Fatalf("metadata.description = %v, want %q", metadata["description"], "Generated by nd")
	}
}

func TestMarketplaceGenerator_MetadataOmittedWhenNoDescription(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:        "my-marketplace",
		Description: "", // empty
		Owner:       Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)

	if _, ok := mj["metadata"]; ok {
		t.Fatal("marketplace.json should not have metadata when description is empty")
	}
}

func TestMarketplaceGenerator_OutputDirExistsWithoutOverwrite(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	// Pre-create the output directory
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := MarketplaceConfig{
		Name:      "my-marketplace",
		Owner:     Author{Name: "Test Author"},
		Plugins:   []PluginEntry{{Name: "plugin-a", Source: pluginDir}},
		OutputDir: outDir,
		Overwrite: false,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected error when output dir exists and Overwrite is false")
	}
	if !strings.Contains(err.Error(), "exists") {
		t.Fatalf("error should mention that directory exists: %v", err)
	}
}

func TestMarketplaceGenerator_OutputDirExistsWithOverwrite(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	// Pre-create the output directory with some content
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "old-file.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MarketplaceConfig{
		Name:      "my-marketplace",
		Owner:     Author{Name: "Test Author"},
		Plugins:   []PluginEntry{{Name: "plugin-a", Source: pluginDir}},
		OutputDir: outDir,
		Overwrite: true,
	}

	gen := &MarketplaceGenerator{}
	result, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error with Overwrite=true: %v", err)
	}

	if result.PluginCount != 1 {
		t.Fatalf("PluginCount = %d, want 1", result.PluginCount)
	}

	// Verify marketplace.json was created
	if _, err := os.Stat(filepath.Join(outDir, ".claude-plugin", "marketplace.json")); err != nil {
		t.Fatalf("marketplace.json not created: %v", err)
	}
}

func TestMarketplaceGenerator_PluginFilesActuallyCopied(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	// Add an extra file to the plugin
	if err := os.WriteFile(filepath.Join(pluginDir, "README.md"), []byte("# Plugin A"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all files exist in the output
	expectedFiles := []string{
		filepath.Join(outDir, "plugins", "plugin-a", ".claude-plugin", "plugin.json"),
		filepath.Join(outDir, "plugins", "plugin-a", "skills", "test-skill", "SKILL.md"),
		filepath.Join(outDir, "plugins", "plugin-a", "README.md"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file %q not found: %v", f, err)
		}
	}

	// Verify file content was preserved
	content, err := os.ReadFile(filepath.Join(outDir, "plugins", "plugin-a", "skills", "test-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("read copied SKILL.md: %v", err)
	}
	if string(content) != "# Test" {
		t.Fatalf("SKILL.md content = %q, want %q", string(content), "# Test")
	}

	readmeContent, err := os.ReadFile(filepath.Join(outDir, "plugins", "plugin-a", "README.md"))
	if err != nil {
		t.Fatalf("read copied README.md: %v", err)
	}
	if string(readmeContent) != "# Plugin A" {
		t.Fatalf("README.md content = %q, want %q", string(readmeContent), "# Plugin A")
	}
}

func TestMarketplaceGenerator_ValidationError(t *testing.T) {
	gen := &MarketplaceGenerator{}

	// Missing name should fail validation
	cfg := MarketplaceConfig{
		Owner:     Author{Name: "Test Author"},
		Plugins:   []PluginEntry{{Name: "p1", Source: "/some/path"}},
		OutputDir: "/tmp/out",
	}

	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestMarketplaceGenerator_PluginSourceDirDoesNotExist(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "marketplace-out")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "missing-plugin", Source: filepath.Join(tmp, "nonexistent"), Version: "1.0.0"},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err == nil {
		t.Fatal("expected error for nonexistent source directory")
	}
	if !strings.Contains(err.Error(), "missing-plugin") {
		t.Fatalf("error should name the plugin: %v", err)
	}
}

func TestMarketplaceGenerator_MarketplaceJSONIndentation(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Test Author"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read raw file to check indentation
	data, err := os.ReadFile(filepath.Join(outDir, ".claude-plugin", "marketplace.json"))
	if err != nil {
		t.Fatalf("read marketplace.json: %v", err)
	}

	content := string(data)
	// 2-space indent should appear in the formatted JSON
	if !strings.Contains(content, "  ") {
		t.Fatal("marketplace.json should use 2-space indentation")
	}
	// Trailing newline for clean file
	if !strings.HasSuffix(content, "\n") {
		t.Fatal("marketplace.json should end with a newline")
	}
}

func TestMarketplaceGenerator_OwnerWithAllFields(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Jane Doe", Email: "jane@example.com", URL: "https://example.com"},
		Plugins: []PluginEntry{
			{Name: "plugin-a", Source: pluginDir},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)
	owner := mj["owner"].(map[string]any)
	if owner["name"] != "Jane Doe" {
		t.Fatalf("owner.name = %v, want %q", owner["name"], "Jane Doe")
	}
	if owner["email"] != "jane@example.com" {
		t.Fatalf("owner.email = %v, want %q", owner["email"], "jane@example.com")
	}
	if owner["url"] != "https://example.com" {
		t.Fatalf("owner.url = %v, want %q", owner["url"], "https://example.com")
	}
}

func TestMarketplaceGenerator_PluginWithAuthor(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "sources")
	outDir := filepath.Join(tmp, "marketplace-out")

	pluginDir := createTestPlugin(t, srcDir, "plugin-a")

	cfg := MarketplaceConfig{
		Name:  "my-marketplace",
		Owner: Author{Name: "Marketplace Owner"},
		Plugins: []PluginEntry{
			{
				Name:    "plugin-a",
				Source:  pluginDir,
				Version: "1.0.0",
				Author:  &Author{Name: "Plugin Author", Email: "plugin@example.com"},
			},
		},
		OutputDir: outDir,
	}

	gen := &MarketplaceGenerator{}
	_, err := gen.Generate(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mj := readMarketplaceJSON(t, outDir)
	plugins := mj["plugins"].([]any)
	plugin := plugins[0].(map[string]any)

	author, ok := plugin["author"].(map[string]any)
	if !ok {
		t.Fatal("plugin should have author object")
	}
	if author["name"] != "Plugin Author" {
		t.Fatalf("plugin author.name = %v, want %q", author["name"], "Plugin Author")
	}
}
