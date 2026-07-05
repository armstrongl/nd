package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

// TestApp_SourceManager_MergesProjectConfigInGlobalScope verifies that
// App.SourceManager() resolves the project root on demand and merges
// .nd/config.yaml when cwd is inside a project, even when the app is in the
// default global scope with an empty ProjectRoot (taskmd k63tsg). Without the
// on-demand resolution, sourcemanager.New would be called with "" and skip the
// project-config merge, leaving DefaultScope at the global default.
func TestApp_SourceManager_MergesProjectConfigInGlobalScope(t *testing.T) {
	projectDir := t.TempDir()
	// A .git/ directory marks the project root for nd.FindProjectRoot.
	if err := os.MkdirAll(filepath.Join(projectDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	// Project config overrides the global default_scope (global -> project).
	ndDir := filepath.Join(projectDir, ".nd")
	if err := os.MkdirAll(ndDir, 0o755); err != nil {
		t.Fatalf("mkdir .nd: %v", err)
	}
	projectConfig := "version: 1\ndefault_scope: project\n"
	if err := os.WriteFile(filepath.Join(ndDir, "config.yaml"), []byte(projectConfig), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	// Run from inside the project so on-demand resolution finds projectDir.
	t.Chdir(projectDir)

	// Global config lives elsewhere and does not exist -> defaults (global scope).
	globalConfig := filepath.Join(t.TempDir(), "config.yaml")

	app := &App{
		ConfigPath: globalConfig,
		Scope:      nd.ScopeGlobal, // default launch scope; ProjectRoot intentionally empty
	}

	sm, err := app.SourceManager()
	if err != nil {
		t.Fatalf("SourceManager: %v", err)
	}
	if got := sm.Config().DefaultScope; got != nd.ScopeProject {
		t.Errorf("project .nd/config.yaml not merged: DefaultScope = %q, want %q", got, nd.ScopeProject)
	}
	// The resolve-on-demand path should have cached the project root.
	if root := app.GetProjectRoot(); root == "" {
		t.Error("GetProjectRoot() = \"\", want a resolved project root")
	}
}

// TestApp_GetProjectRoot_NoProjectReturnsEmpty verifies that GetProjectRoot
// swallows the FindProjectRoot error and returns "" when cwd is not inside a
// project, while ResolveProjectRoot still surfaces the genuine error referencing
// the missing .git/ or .claude/ markers (taskmd k63tsg).
func TestApp_GetProjectRoot_NoProjectReturnsEmpty(t *testing.T) {
	dir := t.TempDir() // no .git/ or .claude/ ancestor markers
	t.Chdir(dir)

	app := &App{
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
		Scope:      nd.ScopeGlobal,
	}

	if root := app.GetProjectRoot(); root != "" {
		t.Errorf("GetProjectRoot() = %q, want \"\" outside a project", root)
	}

	_, err := app.ResolveProjectRoot()
	if err == nil {
		t.Fatal("ResolveProjectRoot() error = nil, want FindProjectRoot error")
	}
	if !strings.Contains(err.Error(), "looked for .git/ or .claude/") {
		t.Errorf("error %q does not reference the missing project markers", err.Error())
	}
}
