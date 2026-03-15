package sourcemanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/sourcemanager"
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
