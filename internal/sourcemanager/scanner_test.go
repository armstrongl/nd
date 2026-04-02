package sourcemanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
)

// makeSourceTree creates a source directory with assets in conventional layout.
func makeSourceTree(t *testing.T, assets map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	for dir, entries := range assets {
		dirPath := filepath.Join(root, dir)
		os.MkdirAll(dirPath, 0o755)
		for _, entry := range entries {
			if entry[len(entry)-1] == '/' {
				// Directory entry — strip trailing slash for the name
				name := entry[:len(entry)-1]
				assetDir := filepath.Join(dirPath, name)
				os.MkdirAll(assetDir, 0o755)
				// Create a placeholder file inside
				os.WriteFile(filepath.Join(assetDir, "SKILL.md"), []byte("# skill"), 0o644)
			} else {
				os.WriteFile(filepath.Join(dirPath, entry), []byte("# "+entry), 0o644)
			}
		}
	}
	return root
}

func TestScanConventionBasic(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills":   {"review/", "deploy/"},
		"rules":    {"no-emojis.md"},
		"commands": {"build-project.md"},
	})
	// Agents are .md files, not directories
	os.MkdirAll(filepath.Join(root, "agents"), 0o755)
	os.WriteFile(filepath.Join(root, "agents", "go-dev.md"), []byte("# agent"), 0o644)

	result := sourcemanager.ScanSource("test-source", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Assets) != 5 {
		t.Errorf("expected 5 assets, got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}

	// Check types
	typeCount := make(map[nd.AssetType]int)
	for _, a := range result.Assets {
		typeCount[a.Type]++
	}
	if typeCount[nd.AssetSkill] != 2 {
		t.Errorf("skills: got %d, want 2", typeCount[nd.AssetSkill])
	}
	if typeCount[nd.AssetAgent] != 1 {
		t.Errorf("agents: got %d, want 1", typeCount[nd.AssetAgent])
	}
	if typeCount[nd.AssetRule] != 1 {
		t.Errorf("rules: got %d, want 1", typeCount[nd.AssetRule])
	}
}

func TestScanConventionSkipsExcluded(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})
	// Add excluded dirs that should be ignored
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	os.MkdirAll(filepath.Join(root, "node_modules"), 0o755)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (only skills/review), got %d", len(result.Assets))
	}
}

func TestScanConventionEmptySource(t *testing.T) {
	root := t.TempDir()
	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 0 {
		t.Errorf("expected 0 assets, got %d", len(result.Assets))
	}
	if len(result.Errors) > 0 {
		t.Errorf("empty source should not produce errors: %v", result.Errors)
	}
}

func TestScanConventionUnavailableSource(t *testing.T) {
	result := sourcemanager.ScanSource("test", "/nonexistent/source")
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unavailable source")
	}
}

func TestScanConventionAssetIdentity(t *testing.T) {
	root := makeSourceTree(t, map[string][]string{
		"skills": {"review/"},
	})

	result := sourcemanager.ScanSource("my-source", root)
	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Assets))
	}
	a := result.Assets[0]
	if a.SourceID != "my-source" {
		t.Errorf("source id: got %q", a.SourceID)
	}
	if a.Name != "review" {
		t.Errorf("name: got %q", a.Name)
	}
	if a.Type != nd.AssetSkill {
		t.Errorf("type: got %q", a.Type)
	}
	if !a.IsDir {
		t.Error("skills should be directories")
	}
}

func TestScanContextAssets(t *testing.T) {
	root := t.TempDir()

	// Create context folder structure
	ctx1 := filepath.Join(root, "context", "go-project-rules")
	os.MkdirAll(ctx1, 0o755)
	os.WriteFile(filepath.Join(ctx1, "CLAUDE.md"), []byte("# Go rules"), 0o644)
	os.WriteFile(filepath.Join(ctx1, "_meta.yaml"), []byte("description: Go project rules\ntags:\n  - go\n"), 0o644)

	ctx2 := filepath.Join(root, "context", "web-frontend")
	os.MkdirAll(ctx2, 0o755)
	os.WriteFile(filepath.Join(ctx2, "CLAUDE.md"), []byte("# Web rules"), 0o644)
	// No _meta.yaml for this one

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 2 {
		t.Fatalf("expected 2 context assets, got %d", len(result.Assets))
	}

	// Find the one with metadata
	var withMeta, withoutMeta *asset.Asset
	for i := range result.Assets {
		if result.Assets[i].Name == "go-project-rules" {
			withMeta = &result.Assets[i]
		}
		if result.Assets[i].Name == "web-frontend" {
			withoutMeta = &result.Assets[i]
		}
	}

	if withMeta == nil {
		t.Fatal("go-project-rules not found")
	} else if withMeta.Type != nd.AssetContext {
		t.Errorf("type: got %q", withMeta.Type)
	}
	if withMeta.ContextFile == nil {
		t.Fatal("ContextFile should be set")
	}
	if withMeta.ContextFile.FolderName != "go-project-rules" {
		t.Errorf("folder: got %q", withMeta.ContextFile.FolderName)
	}
	if withMeta.ContextFile.FileName != "CLAUDE.md" {
		t.Errorf("file: got %q", withMeta.ContextFile.FileName)
	}
	if withMeta.Meta == nil {
		t.Fatal("Meta should be set for asset with _meta.yaml")
	}
	if withMeta.Meta.Description != "Go project rules" {
		t.Errorf("description: got %q", withMeta.Meta.Description)
	}

	if withoutMeta == nil {
		t.Fatal("web-frontend not found")
	} else {
		if withoutMeta.Meta != nil {
			t.Error("Meta should be nil for asset without _meta.yaml")
		}
		if withoutMeta.ContextFile == nil {
			t.Fatal("ContextFile should still be set")
		}
	}
}

func TestScanWithManifest(t *testing.T) {
	root := t.TempDir()

	// Non-conventional layout with manifest
	os.MkdirAll(filepath.Join(root, "go-skills", "skills", "review"), 0o755)
	os.WriteFile(filepath.Join(root, "go-skills", "skills", "review", "SKILL.md"), []byte("# review"), 0o644)
	os.MkdirAll(filepath.Join(root, "custom-agents"), 0o755)
	os.WriteFile(filepath.Join(root, "custom-agents", "builder.md"), []byte("# builder"), 0o644)

	// Also has conventional skills/ that should be IGNORED when manifest exists
	os.MkdirAll(filepath.Join(root, "skills", "ignored"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "ignored", "SKILL.md"), []byte("# ignored"), 0o644)

	manifest := `version: 1
paths:
  skills:
    - go-skills/skills
  agents:
    - custom-agents
`
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), []byte(manifest), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("errors: %v", result.Errors)
	}
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 assets (1 skill + 1 agent), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}

	// The conventional skills/ignored should NOT be discovered
	for _, a := range result.Assets {
		if a.Name == "ignored" {
			t.Error("conventional skills/ignored should not be discovered when manifest exists")
		}
	}
}

func TestScanManifestExclude(t *testing.T) {
	root := t.TempDir()

	// Create skills with SKILL.md (valid structure)
	os.MkdirAll(filepath.Join(root, "skills", "keep"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "keep", "SKILL.md"), []byte("# skill"), 0o644)
	os.MkdirAll(filepath.Join(root, "skills", "experimental"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "experimental", "SKILL.md"), []byte("# skill"), 0o644)

	manifest := `version: 1
paths:
  skills:
    - skills
exclude:
  - experimental
`
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), []byte(manifest), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (excluded experimental), got %d", len(result.Assets))
	}
	for _, a := range result.Assets {
		if a.Name == "experimental" {
			t.Error("excluded asset should not be discovered")
		}
	}
}

func TestScanManifestSizeLimit(t *testing.T) {
	root := t.TempDir()
	// Create a manifest larger than 1MB
	data := make([]byte, 1024*1024+1)
	for i := range data {
		data[i] = 'x'
	}
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), data, 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) == 0 {
		t.Error("expected error for oversized manifest")
	}
}

func TestScanContextLocalOnly(t *testing.T) {
	root := t.TempDir()

	ctx := filepath.Join(root, "context", "local-rules")
	os.MkdirAll(ctx, 0o755)
	os.WriteFile(filepath.Join(ctx, "CLAUDE.local.md"), []byte("# Local"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(result.Assets))
	}
	if result.Assets[0].ContextFile.FileName != "CLAUDE.local.md" {
		t.Errorf("file: got %q", result.Assets[0].ContextFile.FileName)
	}
}

func TestScanValidatesSkillStructure(t *testing.T) {
	root := t.TempDir()

	// Valid skill: directory with SKILL.md
	os.MkdirAll(filepath.Join(root, "skills", "valid-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "valid-skill", "SKILL.md"), []byte("# skill"), 0o644)

	// Invalid: directory without SKILL.md
	os.MkdirAll(filepath.Join(root, "skills", "empty-dir"), 0o755)

	// Invalid: plain file (not a directory)
	os.WriteFile(filepath.Join(root, "skills", "CLAUDE.md"), []byte("# readme"), 0o644)

	// Invalid: directory with wrong marker file
	os.MkdirAll(filepath.Join(root, "skills", "wrong-marker"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "wrong-marker", "README.md"), []byte("# readme"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 valid skill, got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s (dir=%v)", a.Type, a.Name, a.IsDir)
		}
	}
	if len(result.Assets) > 0 && result.Assets[0].Name != "valid-skill" {
		t.Errorf("expected valid-skill, got %q", result.Assets[0].Name)
	}
}

func TestScanValidatesAgentStructure(t *testing.T) {
	root := t.TempDir()

	os.MkdirAll(filepath.Join(root, "agents"), 0o755)
	// Valid: .md file
	os.WriteFile(filepath.Join(root, "agents", "go-dev.md"), []byte("# agent"), 0o644)
	// Invalid: directory (agents are files)
	os.MkdirAll(filepath.Join(root, "agents", "not-an-agent"), 0o755)
	// Invalid: non-.md file
	os.WriteFile(filepath.Join(root, "agents", "notes.txt"), []byte("notes"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 valid agent, got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s (dir=%v)", a.Type, a.Name, a.IsDir)
		}
	}
	if len(result.Assets) > 0 && result.Assets[0].Name != "go-dev.md" {
		t.Errorf("expected go-dev.md, got %q", result.Assets[0].Name)
	}
}

func TestScanValidatesCommandStructure(t *testing.T) {
	root := t.TempDir()

	os.MkdirAll(filepath.Join(root, "commands"), 0o755)
	// Valid: .md file
	os.WriteFile(filepath.Join(root, "commands", "build.md"), []byte("# cmd"), 0o644)
	// Invalid: directory
	os.MkdirAll(filepath.Join(root, "commands", "subdir"), 0o755)
	// Invalid: non-.md file
	os.WriteFile(filepath.Join(root, "commands", "script.sh"), []byte("#!/bin/bash"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	cmdAssets := 0
	for _, a := range result.Assets {
		if a.Type == nd.AssetCommand {
			cmdAssets++
			if a.Name != "build.md" {
				t.Errorf("expected build.md, got %q", a.Name)
			}
		}
	}
	if cmdAssets != 1 {
		t.Errorf("expected 1 valid command, got %d", cmdAssets)
	}
}

func TestScanValidatesHookStructure(t *testing.T) {
	root := t.TempDir()

	// Valid: directory with hooks.json
	os.MkdirAll(filepath.Join(root, "hooks", "pre-commit"), 0o755)
	os.WriteFile(filepath.Join(root, "hooks", "pre-commit", "hooks.json"), []byte("{}"), 0o644)

	// Invalid: directory without hooks.json
	os.MkdirAll(filepath.Join(root, "hooks", "empty-hook"), 0o755)

	// Invalid: file (hooks are directories)
	os.WriteFile(filepath.Join(root, "hooks", "not-a-hook.md"), []byte("# nope"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	hookAssets := 0
	for _, a := range result.Assets {
		if a.Type == nd.AssetHook {
			hookAssets++
			if a.Name != "pre-commit" {
				t.Errorf("expected pre-commit, got %q", a.Name)
			}
		}
	}
	if hookAssets != 1 {
		t.Errorf("expected 1 valid hook, got %d", hookAssets)
	}
}

func TestScanValidatesPluginStructure(t *testing.T) {
	root := t.TempDir()

	// Valid: directory with .claude-plugin/ subdirectory
	os.MkdirAll(filepath.Join(root, "plugins", "my-plugin", ".claude-plugin"), 0o755)

	// Invalid: directory without .claude-plugin/
	os.MkdirAll(filepath.Join(root, "plugins", "not-a-plugin"), 0o755)

	result := sourcemanager.ScanSource("test", root)
	pluginAssets := 0
	for _, a := range result.Assets {
		if a.Type == nd.AssetPlugin {
			pluginAssets++
			if a.Name != "my-plugin" {
				t.Errorf("expected my-plugin, got %q", a.Name)
			}
		}
	}
	if pluginAssets != 1 {
		t.Errorf("expected 1 valid plugin, got %d", pluginAssets)
	}
}

func TestScanGroupingFolderNesting(t *testing.T) {
	root := t.TempDir()

	// Skills organized in grouping folders (like ai-toolbox)
	os.MkdirAll(filepath.Join(root, "skills", "claude", "better-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "claude", "better-skill", "SKILL.md"), []byte("# skill"), 0o644)
	os.MkdirAll(filepath.Join(root, "skills", "claude", "cc-assets"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "claude", "cc-assets", "SKILL.md"), []byte("# skill"), 0o644)
	os.MkdirAll(filepath.Join(root, "skills", "codex", "code-review"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "codex", "code-review", "SKILL.md"), []byte("# skill"), 0o644)

	// Agents organized in grouping folders
	os.MkdirAll(filepath.Join(root, "agents", "claude"), 0o755)
	os.WriteFile(filepath.Join(root, "agents", "claude", "go-dev.md"), []byte("# agent"), 0o644)
	os.WriteFile(filepath.Join(root, "agents", "claude", "research.md"), []byte("# agent"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	typeCount := make(map[nd.AssetType]int)
	names := make(map[string]bool)
	for _, a := range result.Assets {
		typeCount[a.Type]++
		names[a.Name] = true
	}

	if typeCount[nd.AssetSkill] != 3 {
		t.Errorf("skills: got %d, want 3", typeCount[nd.AssetSkill])
	}
	if typeCount[nd.AssetAgent] != 2 {
		t.Errorf("agents: got %d, want 2", typeCount[nd.AssetAgent])
	}

	// Grouping folders should NOT appear as assets
	if names["claude"] {
		t.Error("grouping folder 'claude' should not appear as an asset")
	}
	if names["codex"] {
		t.Error("grouping folder 'codex' should not appear as an asset")
	}

	// Actual assets should be found
	for _, expected := range []string{"better-skill", "cc-assets", "code-review", "go-dev.md", "research.md"} {
		if !names[expected] {
			t.Errorf("expected asset %q not found", expected)
		}
	}
}

func TestScanNestingDepthLimit(t *testing.T) {
	root := t.TempDir()

	// Two levels deep — only one level of nesting should be scanned
	os.MkdirAll(filepath.Join(root, "skills", "group1", "group2", "deep-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "group1", "group2", "deep-skill", "SKILL.md"), []byte("# skill"), 0o644)

	// One level deep — should be found
	os.MkdirAll(filepath.Join(root, "skills", "group1", "shallow-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "group1", "shallow-skill", "SKILL.md"), []byte("# skill"), 0o644)

	result := sourcemanager.ScanSource("test", root)

	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (only shallow-skill), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	if len(result.Assets) > 0 && result.Assets[0].Name != "shallow-skill" {
		t.Errorf("expected shallow-skill, got %q", result.Assets[0].Name)
	}
}

func TestScanNestingWithManifest(t *testing.T) {
	root := t.TempDir()

	// Manifest-based scan should also support nesting
	os.MkdirAll(filepath.Join(root, "my-skills", "claude", "review"), 0o755)
	os.WriteFile(filepath.Join(root, "my-skills", "claude", "review", "SKILL.md"), []byte("# skill"), 0o644)

	manifest := `version: 1
paths:
  skills:
    - my-skills
`
	os.WriteFile(filepath.Join(root, "nd-source.yaml"), []byte(manifest), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("errors: %v", result.Errors)
	}
	if len(result.Assets) != 1 {
		t.Errorf("expected 1 skill via nesting in manifest, got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	if len(result.Assets) > 0 {
		if result.Assets[0].Name != "review" {
			t.Errorf("expected review, got %q", result.Assets[0].Name)
		}
	}
}

func TestScanSkipsSymlinksInAssetDir(t *testing.T) {
	root := t.TempDir()

	// Create output-styles/ with real .md files
	osDir := filepath.Join(root, "output-styles")
	os.MkdirAll(osDir, 0o755)
	os.WriteFile(filepath.Join(osDir, "concise.md"), []byte("# concise"), 0o644)
	os.WriteFile(filepath.Join(osDir, "verbose.md"), []byte("# verbose"), 0o644)

	// Create a symlink to a real .md file — should be skipped
	os.Symlink(filepath.Join(osDir, "concise.md"), filepath.Join(osDir, "link-to-concise.md"))

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	// Only real files should be discovered, not the symlink
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 assets (concise.md + verbose.md), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	for _, a := range result.Assets {
		if a.Name == "link-to-concise.md" {
			t.Error("symlink link-to-concise.md should not be discovered")
		}
	}
}

func TestScanSkipsSymlinkDirectories(t *testing.T) {
	root := t.TempDir()

	// Create skills/ with a real skill dir
	realSkill := filepath.Join(root, "skills", "real-skill")
	os.MkdirAll(realSkill, 0o755)
	os.WriteFile(filepath.Join(realSkill, "SKILL.md"), []byte("# skill"), 0o644)

	// Create a symlink to the real skill dir — should be skipped
	os.Symlink(realSkill, filepath.Join(root, "skills", "link-to-skill"))

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (only real-skill), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	for _, a := range result.Assets {
		if a.Name == "link-to-skill" {
			t.Error("symlink directory link-to-skill should not be discovered")
		}
	}
}

func TestScanSkipsBrokenSymlinks(t *testing.T) {
	root := t.TempDir()

	// Create commands/ with a real .md file
	cmdDir := filepath.Join(root, "commands")
	os.MkdirAll(cmdDir, 0o755)
	os.WriteFile(filepath.Join(cmdDir, "build.md"), []byte("# build"), 0o644)

	// Create a broken symlink (target does not exist)
	os.Symlink("/nonexistent/target.md", filepath.Join(cmdDir, "broken-link.md"))

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if len(result.Assets) != 1 {
		t.Errorf("expected 1 asset (only build.md), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	for _, a := range result.Assets {
		if a.Name == "broken-link.md" {
			t.Error("broken symlink broken-link.md should not be discovered")
		}
	}
}

func TestScanContextSkipsSymlinks(t *testing.T) {
	root := t.TempDir()

	// Create context/ with a real context folder
	realCtx := filepath.Join(root, "context", "go-rules")
	os.MkdirAll(realCtx, 0o755)
	os.WriteFile(filepath.Join(realCtx, "CLAUDE.md"), []byte("# Go rules"), 0o644)

	// Create a symlink to the real context folder — should be skipped
	os.Symlink(realCtx, filepath.Join(root, "context", "link-to-rules"))

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if len(result.Assets) != 1 {
		t.Errorf("expected 1 context asset (only go-rules), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}
	for _, a := range result.Assets {
		if a.Name == "link-to-rules" {
			t.Error("symlink directory link-to-rules should not be discovered as context")
		}
	}
}

func TestScanFindContextFileSkipsSymlinks(t *testing.T) {
	root := t.TempDir()

	// Create context folder with a real .md file
	ctxFolder := filepath.Join(root, "context", "my-rules")
	os.MkdirAll(ctxFolder, 0o755)
	os.WriteFile(filepath.Join(ctxFolder, "CLAUDE.md"), []byte("# Rules"), 0o644)

	// Add a symlink .md file that sorts BEFORE CLAUDE.md alphabetically.
	// Without the symlink check, findContextFile would return this first.
	os.Symlink(filepath.Join(ctxFolder, "CLAUDE.md"), filepath.Join(ctxFolder, "AAA-LINK.md"))

	result := sourcemanager.ScanSource("test", root)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if len(result.Assets) != 1 {
		t.Fatalf("expected 1 context asset, got %d", len(result.Assets))
	}

	// The context file chosen should be the real file, not the symlink
	a := result.Assets[0]
	if a.ContextFile == nil {
		t.Fatal("ContextFile should be set")
	}
	if a.ContextFile.FileName == "AAA-LINK.md" {
		t.Error("findContextFile should not select symlink AAA-LINK.md")
	}
	if a.ContextFile.FileName != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %q", a.ContextFile.FileName)
	}
}

func TestScanMixedValidAndGrouping(t *testing.T) {
	root := t.TempDir()

	// Mix of direct valid skills and grouped skills
	os.MkdirAll(filepath.Join(root, "skills", "direct-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "direct-skill", "SKILL.md"), []byte("# skill"), 0o644)

	os.MkdirAll(filepath.Join(root, "skills", "claude", "grouped-skill"), 0o755)
	os.WriteFile(filepath.Join(root, "skills", "claude", "grouped-skill", "SKILL.md"), []byte("# skill"), 0o644)

	result := sourcemanager.ScanSource("test", root)
	if len(result.Assets) != 2 {
		t.Errorf("expected 2 skills (1 direct + 1 grouped), got %d", len(result.Assets))
		for _, a := range result.Assets {
			t.Logf("  %s/%s", a.Type, a.Name)
		}
	}

	names := make(map[string]bool)
	for _, a := range result.Assets {
		names[a.Name] = true
	}
	if !names["direct-skill"] {
		t.Error("direct-skill not found")
	}
	if !names["grouped-skill"] {
		t.Error("grouped-skill not found")
	}
}
