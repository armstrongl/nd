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
		"agents":   {"go-dev/"},
		"rules":    {"no-emojis.md"},
		"commands": {"build-project.md"},
	})

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
	}
	if withMeta.Type != nd.AssetContext {
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
	}
	if withoutMeta.Meta != nil {
		t.Error("Meta should be nil for asset without _meta.yaml")
	}
	if withoutMeta.ContextFile == nil {
		t.Fatal("ContextFile should still be set")
	}
}

func TestScanWithManifest(t *testing.T) {
	root := t.TempDir()

	// Non-conventional layout with manifest
	os.MkdirAll(filepath.Join(root, "go-skills", "skills"), 0o755)
	os.MkdirAll(filepath.Join(root, "go-skills", "skills", "review"), 0o755)
	os.WriteFile(filepath.Join(root, "go-skills", "skills", "review", "SKILL.md"), []byte("# review"), 0o644)
	os.MkdirAll(filepath.Join(root, "custom-agents"), 0o755)
	os.MkdirAll(filepath.Join(root, "custom-agents", "builder"), 0o755)

	// Also has conventional skills/ that should be IGNORED when manifest exists
	os.MkdirAll(filepath.Join(root, "skills"), 0o755)
	os.MkdirAll(filepath.Join(root, "skills", "ignored"), 0o755)

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

	// Create skills
	os.MkdirAll(filepath.Join(root, "skills", "keep"), 0o755)
	os.MkdirAll(filepath.Join(root, "skills", "experimental"), 0o755)

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
