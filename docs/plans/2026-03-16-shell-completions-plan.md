# Shell Completions Implementation Plan

> **For agentic workers:** REQUIRED: Use supapowers:subagent-driven-development (if subagents available) or supapowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add shell completion support for Bash, Zsh, and Fish with dynamic completions for asset names, profiles, snapshots, and source IDs (FR-035).

**Architecture:** A hidden `nd completion` parent command with `bash`, `zsh`, and `fish` subcommands that print completion scripts to stdout. An `--install` flag writes to conventional paths. Dynamic completions are wired via Cobra's `ValidArgsFunction` on existing commands. A lightweight `completionInitApp` helper handles App initialization for completion contexts where `PersistentPreRunE` hasn't run.

**Tech Stack:** Cobra (built-in completion generation), Go standard library

**Spec:** `docs/plans/2026-03-16-shell-completions-design.md`

---

## File structure

| File | Action | Responsibility |
|------|--------|----------------|
| `cmd/completion.go` | Create | Hidden `completion` parent + `bash`/`zsh`/`fish` subcommands, `--install` flag logic |
| `cmd/completion_test.go` | Create | Tests for completion output markers and `--install` behavior |
| `cmd/helpers.go` | Modify | Add `completionInitApp(app *App)` helper |
| `cmd/helpers_test.go` | Modify | Add test for `completionInitApp` |
| `cmd/root.go` | Modify | Register `completion` command, register `--scope` flag completion on rootCmd |
| `cmd/deploy.go` | Modify | Add `ValidArgsFunction` + `RegisterFlagCompletionFunc("type")` |
| `cmd/remove.go` | Modify | Add `ValidArgsFunction` |
| `cmd/pin.go` | Modify | Add `ValidArgsFunction` to pin/unpin |
| `cmd/profile.go` | Modify | Add `ValidArgsFunction` to delete/switch/deploy/add-asset |
| `cmd/snapshot.go` | Modify | Add `ValidArgsFunction` to restore/delete |
| `cmd/source.go` | Modify | Add `ValidArgsFunction` to source remove |
| `cmd/list.go` | Modify | Add `RegisterFlagCompletionFunc` for `--type` and `--source` |
| `cmd/sync.go` | Modify | Add `RegisterFlagCompletionFunc` for `--source` |

## Implementation notes

**Import additions for `cmd/helpers.go`:** Across Chunks 1-2, the following imports must be added to `helpers.go`: `"os/user"`, `"path/filepath"`, and `"github.com/spf13/cobra"`. Add them as you encounter the first task that needs them.

**Universal refactoring pattern for Chunk 2:** Several commands (remove, pin, unpin, profile delete/switch/deploy/add-asset, snapshot restore/delete, source remove) currently return `&cobra.Command{...}` directly. Each must be refactored to `cmd := &cobra.Command{...}; cmd.ValidArgsFunction = ...; return cmd` to support adding `ValidArgsFunction`. This structural change is identical for all of them.

**Test helpers:** All completion tests in Chunks 2-3 should use `setupDeployEnv(t)` from `cmd/deploy_test.go` — it provides a fully wired config with a source and assets pre-registered. There are no `setupProfileEnv`, `setupSnapshotEnv`, or `setupSourceEnv` helpers in the codebase.

**Scope-agnostic completions:** The `completeDeployedAssets` helper returns ALL deployed assets regardless of scope, because `PersistentPreRunE` hasn't run during completion so scope defaults to "global". This is acceptable since completions are advisory. Add a code comment noting this.

## Chunk 1: Completion command and helper

### Task 1: Add completionInitApp helper

**Files:**

- Modify: `cmd/helpers.go`
- Modify: `cmd/helpers_test.go`

This helper does lightweight App initialization for completion contexts where `PersistentPreRunE` has not run. It expands `~` in `ConfigPath` and sets `BackupDir`.

- [ ] **Step 1: Write failing test for completionInitApp**

In `cmd/helpers_test.go`, add:

```go
func TestCompletionInitApp(t *testing.T) {
	app := &App{ConfigPath: "~/.config/nd/config.yaml"}
	completionInitApp(app)

	if strings.Contains(app.ConfigPath, "~") {
		t.Errorf("ConfigPath still contains ~: %s", app.ConfigPath)
	}
	if app.BackupDir == "" {
		t.Error("BackupDir not set")
	}
	if !strings.HasSuffix(app.BackupDir, "backups") {
		t.Errorf("BackupDir should end with 'backups', got: %s", app.BackupDir)
	}
}

func TestCompletionInitApp_Idempotent(t *testing.T) {
	app := &App{ConfigPath: "/tmp/nd/config.yaml"}
	completionInitApp(app)
	first := app.ConfigPath
	completionInitApp(app)
	if app.ConfigPath != first {
		t.Errorf("not idempotent: %s != %s", first, app.ConfigPath)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestCompletionInitApp -v`
Expected: FAIL with "undefined: completionInitApp"

- [ ] **Step 3: Write minimal implementation**

In `cmd/helpers.go`, add after the existing imports (add `"os/user"` and `"path/filepath"` to imports):

```go
// completionInitApp does lightweight App initialization for completion contexts.
// PersistentPreRunE is not guaranteed to run during shell completion, so this
// handles the essential setup: expanding ~ in ConfigPath and deriving BackupDir.
// This function is idempotent and safe to call multiple times.
func completionInitApp(app *App) {
	if strings.HasPrefix(app.ConfigPath, "~/") {
		if u, err := user.Current(); err == nil {
			app.ConfigPath = filepath.Join(u.HomeDir, app.ConfigPath[2:])
		}
	}
	if app.BackupDir == "" {
		app.BackupDir = filepath.Join(filepath.Dir(app.ConfigPath), "backups")
	}
}
```

Note: `strings`, `os/user`, and `path/filepath` are already imported in `helpers.go` or need to be added to the import block.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestCompletionInitApp -v`
Expected: PASS (both tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/helpers.go cmd/helpers_test.go
git commit -m "feat(cli): add completionInitApp helper for shell completion contexts"
```

### Task 2: Create completion command with Bash/Zsh/Fish subcommands

**Files:**

- Create: `cmd/completion.go`
- Create: `cmd/completion_test.go`
- Modify: `cmd/root.go:48` (add to `rootCmd.AddCommand`)
- [ ] **Step 1: Write failing tests for completion command**

Create `cmd/completion_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "__nd") {
		t.Errorf("expected bash completion to contain '__nd' function, got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionZsh(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "#compdef") || !strings.Contains(got, "nd") {
		t.Errorf("expected zsh completion header, got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionFish(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"completion", "fish"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "complete") {
		t.Errorf("expected fish completion to contain 'complete', got:\n%s", got[:min(200, len(got))])
	}
}

func TestCompletionHidden(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--help"})

	_ = rootCmd.Execute()

	got := out.String()
	if strings.Contains(got, "completion") {
		t.Errorf("completion command should be hidden from help, but found in:\n%s", got)
	}
}

func TestCompletionNoSubcommand(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion"})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "bash") || !strings.Contains(got, "zsh") || !strings.Contains(got, "fish") {
		t.Errorf("expected usage showing bash, zsh, fish, got:\n%s", got)
	}
}

func TestCompletionBashInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".local", "share", "bash-completion", "completions")
	os.MkdirAll(installDir, 0o755)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "bash", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "nd"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "__nd") {
		t.Errorf("installed file missing bash completion content")
	}
}

func TestCompletionZshInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".zfunc")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "zsh", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "_nd"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "#compdef") {
		t.Errorf("installed file missing zsh completion content")
	}
}

func TestCompletionFishInstallWithDir(t *testing.T) {
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, ".config", "fish", "completions")

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "fish", "--install", "--install-dir", installDir})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(installDir, "nd.fish"))
	if err != nil {
		t.Fatalf("completion file not written: %v", err)
	}
	if !strings.Contains(string(content), "complete") {
		t.Errorf("installed file missing fish completion content")
	}
}

func TestCompletionBashInstallDefault(t *testing.T) {
	// Tests that --install without --install-dir uses default path resolution.
	// We can't test the actual default path (it uses $HOME), but we verify
	// the function doesn't error when UserHomeDir succeeds.
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "bash", "--install"})

	// This will write to the real default dir. It should succeed (creates dir if needed).
	// If it fails with permission error, that's expected on some systems.
	err := rootCmd.Execute()
	if err != nil {
		// Permission errors are acceptable — verify it's not a logic error
		got := err.Error()
		if !strings.Contains(got, "permission") && !strings.Contains(got, "create directory") {
			t.Fatalf("unexpected error (not permission-related): %v", err)
		}
	}
}

func TestCompletionInstallUnwritable(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"completion", "bash", "--install", "--install-dir", "/nonexistent/readonly/path"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unwritable path")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run TestCompletion -v`
Expected: FAIL (completion command doesn't exist yet)

- [ ] **Step 3: Create cmd/completion.go**

```go
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCompletionCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for nd.

Available shells: bash, zsh, fish

Run "nd completion <shell> --help" for shell-specific instructions.`,
		Hidden: true,
	}

	cmd.AddCommand(
		newCompletionBashCmd(app),
		newCompletionZshCmd(app),
		newCompletionFishCmd(app),
	)

	return cmd
}

func newCompletionBashCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate bash completion script for nd.

To install completions:
  nd completion bash --install

Or manually:
  nd completion bash > ~/.local/share/bash-completion/completions/nd

Then restart your shell or source the file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
			}
			dir := installDir
			if dir == "" {
				dir = defaultBashCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenBashCompletionV2(buf, true)
			}, dir, "nd")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

func newCompletionZshCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate zsh completion script for nd.

To install completions:
  nd completion zsh --install

Or manually:
  nd completion zsh > ~/.zfunc/_nd

Then add to ~/.zshrc (if not already present):
  fpath+=~/.zfunc
  autoload -Uz compinit && compinit`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			}
			dir := installDir
			if dir == "" {
				dir = defaultZshCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenZshCompletion(buf)
			}, dir, "_nd")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

func newCompletionFishCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate fish completion script for nd.

To install completions:
  nd completion fish --install

Or manually:
  nd completion fish > ~/.config/fish/completions/nd.fish`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			}
			dir := installDir
			if dir == "" {
				dir = defaultFishCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenFishCompletion(buf, true)
			}, dir, "nd.fish")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

// defaultBashCompletionDir returns the XDG-compliant bash completion dir,
// falling back to ~/.bash_completion.d if the XDG dir doesn't exist.
func defaultBashCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	xdg := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	if info, err := os.Stat(xdg); err == nil && info.IsDir() {
		return xdg
	}
	return filepath.Join(home, ".bash_completion.d")
}

// defaultZshCompletionDir returns ~/.zfunc as the conventional zsh fpath dir.
func defaultZshCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".zfunc")
}

// defaultFishCompletionDir returns the standard fish completions dir.
func defaultFishCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fish", "completions")
}

// installCompletion generates a completion script and writes it to a file.
// Creates the directory if needed.
func installCompletion(cmd *cobra.Command, genFn func(*bytes.Buffer) error, dir, filename string) error {
	if dir == "" {
		return fmt.Errorf("could not determine home directory; use --install-dir to specify a path")
	}

	var buf bytes.Buffer
	if err := genFn(&buf); err != nil {
		return fmt.Errorf("generate completion: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w\n\nYou can install manually instead:\n  nd completion %s > <path>",
			dir, err, cmd.Name())
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w\n\nYou can install manually instead:\n  nd completion %s > <path>",
			path, err, cmd.Name())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Completion script installed to %s\n", path)
	return nil
}
```

- [ ] **Step 4: Register completion command in root.go**

In `cmd/root.go`, add `newCompletionCmd(app)` to the `rootCmd.AddCommand(...)` block:

```go
rootCmd.AddCommand(
    newVersionCmd(app),
    newSourceCmd(app),
    newDeployCmd(app),
    newRemoveCmd(app),
    newListCmd(app),
    newStatusCmd(app),
    newPinCmd(app),
    newUnpinCmd(app),
    newSyncCmd(app),
    newDoctorCmd(app),
    newProfileCmd(app),
    newSnapshotCmd(app),
    newInitCmd(app),
    newSettingsCmd(app),
    newUninstallCmd(app),
    newCompletionCmd(app),
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestCompletion -v`
Expected: All PASS

- [ ] **Step 6: Run full test suite to verify no regressions**

Run: `go test ./... -timeout 120s`
Expected: All packages pass

- [ ] **Step 7: Commit**

```bash
git add cmd/completion.go cmd/completion_test.go cmd/root.go
git commit -m "feat(cli): add hidden completion command for bash, zsh, fish (FR-035)"
```

## Chunk 2: Dynamic completions — positional arguments

### Task 3: Add ValidArgsFunction to deploy command

**Files:**

- Modify: `cmd/deploy.go:17-141` (add ValidArgsFunction to the command)

- [ ] **Step 1: Write failing test**

Add to `cmd/deploy_test.go`:

```go
func TestDeployCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	// Use Cobra's completion debug mechanism
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "deploy", ""})

	_ = rootCmd.Execute()

	got := out.String()
	// Should suggest available asset names from the index
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in completions, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestDeployCmd_Completions -v`
Expected: FAIL — no completions returned (ValidArgsFunction not set)

- [ ] **Step 3: Add ValidArgsFunction to deploy command**

In `cmd/deploy.go`, inside `newDeployCmd`, after the `cmd` variable is created (before `cmd.Flags()`), add:

```go
cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    completionInitApp(app)
    summary, err := app.ScanIndex()
    if err != nil {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }
    var names []string
    for _, a := range summary.Index.All() {
        name := fmt.Sprintf("%s/%s", a.Type, a.Name)
        if toComplete == "" || strings.HasPrefix(name, toComplete) || strings.HasPrefix(a.Name, toComplete) {
            names = append(names, fmt.Sprintf("%s/%s\t%s from %s", a.Type, a.Name, a.Type, a.SourceID))
        }
    }
    return names, cobra.ShellCompDirectiveNoFileComp
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestDeployCmd_Completions -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/deploy.go cmd/deploy_test.go
git commit -m "feat(cli): add dynamic completions for deploy command"
```

### Task 4: Add ValidArgsFunction to remove, pin, and unpin commands

**Files:**

- Modify: `cmd/remove.go:14-123`
- Modify: `cmd/pin.go:10-30`

These three commands all complete from deployed asset names (state file), so they share the same completion pattern.

- [ ] **Step 1: Write failing tests**

Add to `cmd/remove_test.go`:

```go
func TestRemoveCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	// First deploy an asset
	var devNull bytes.Buffer
	rootCmd.SetOut(&devNull)
	rootCmd.SetErr(&devNull)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	_ = rootCmd.Execute()

	// Now test completions for remove
	app2 := &App{ConfigPath: configPath}
	completionInitApp(app2)
	rootCmd2 := NewRootCmd(app2)

	var out bytes.Buffer
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"__complete", "remove", ""})

	_ = rootCmd2.Execute()

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in remove completions, got:\n%s", got)
	}
}
```

Add to `cmd/pin_test.go`:

```go
func TestPinCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	// First deploy
	var devNull bytes.Buffer
	rootCmd.SetOut(&devNull)
	rootCmd.SetErr(&devNull)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	_ = rootCmd.Execute()

	// Test pin completions
	app2 := &App{ConfigPath: configPath}
	completionInitApp(app2)
	rootCmd2 := NewRootCmd(app2)

	var out bytes.Buffer
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"__complete", "pin", ""})

	_ = rootCmd2.Execute()

	got := out.String()
	if !strings.Contains(got, "greeting") {
		t.Errorf("expected 'greeting' in pin completions, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run "TestRemoveCmd_Completions|TestPinCmd_Completions" -v`
Expected: FAIL

- [ ] **Step 3: Add a shared helper and wire ValidArgsFunction**

In `cmd/helpers.go`, add:

```go
// completeDeployedAssets returns names of deployed assets for shell completion.
func completeDeployedAssets(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	eng, err := app.DeployEngine()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := eng.Status()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, e := range entries {
		d := e.Deployment
		name := fmt.Sprintf("%s/%s", d.AssetType, d.AssetName)
		if toComplete == "" || strings.HasPrefix(name, toComplete) || strings.HasPrefix(d.AssetName, toComplete) {
			names = append(names, fmt.Sprintf("%s/%s\t%s from %s", d.AssetType, d.AssetName, d.Scope, d.SourceID))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
```

Add import for `"github.com/spf13/cobra"` to `cmd/helpers.go` imports.

In `cmd/remove.go`, change the command creation to capture the cmd in a variable, then add `ValidArgsFunction` before the return:

```go
func newRemoveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		// ... existing fields unchanged ...
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeDeployedAssets(app, toComplete)
	}
	return cmd
}
```

In `cmd/pin.go`, add `ValidArgsFunction` to both commands. Modify `newPinCmd`:

```go
func newPinCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin <asset>",
		Short: "Pin an asset to prevent profile switches from removing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setAssetOrigin(cmd, app, args[0], nd.OriginPinned, "Pinned")
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeDeployedAssets(app, toComplete)
	}
	return cmd
}
```

Same for `newUnpinCmd`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run "TestRemoveCmd_Completions|TestPinCmd_Completions" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/helpers.go cmd/remove.go cmd/pin.go cmd/remove_test.go cmd/pin_test.go
git commit -m "feat(cli): add dynamic completions for remove, pin, unpin commands"
```

### Task 5: Add ValidArgsFunction to profile subcommands

**Files:**

- Modify: `cmd/profile.go:121-445`

Profile subcommands (`delete`, `switch`, `deploy`, `add-asset`) complete profile names. `add-asset` also completes asset names for its second argument.

- [ ] **Step 1: Write failing test**

Add to `cmd/profile_test.go`:

```go
func TestProfileSwitchCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Create a profile first so completions have something to return
	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	var devNull bytes.Buffer
	rootCmd.SetOut(&devNull)
	rootCmd.SetErr(&devNull)
	rootCmd.SetArgs([]string{"--config", configPath, "profile", "create", "test-profile", "--from-current"})
	_ = rootCmd.Execute()

	// Now test completions
	app2 := &App{ConfigPath: configPath}
	completionInitApp(app2)
	rootCmd2 := NewRootCmd(app2)

	var out bytes.Buffer
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"__complete", "profile", "switch", ""})

	_ = rootCmd2.Execute()

	got := out.String()
	if !strings.Contains(got, "test-profile") {
		t.Errorf("expected 'test-profile' in profile switch completions, got:\n%s", got)
	}
}
```

Note: Uses `setupDeployEnv` from `deploy_test.go` which provides a fully wired config.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestProfileSwitchCmd_Completions -v`
Expected: FAIL

- [ ] **Step 3: Add shared profile completion helper and wire it**

In `cmd/helpers.go`, add:

```go
// completeProfileNames returns profile names for shell completion.
func completeProfileNames(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	pstore, err := app.ProfileStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profiles, err := pstore.ListProfiles()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range profiles {
		if toComplete == "" || strings.HasPrefix(p.Name, toComplete) {
			desc := fmt.Sprintf("%d assets", p.AssetCount)
			if p.Description != "" {
				desc = p.Description
			}
			names = append(names, fmt.Sprintf("%s\t%s", p.Name, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
```

In `cmd/profile.go`, add `ValidArgsFunction` to `newProfileDeleteCmd`, `newProfileSwitchCmd`, `newProfileDeployCmd`:

```go
// In each of these, change from returning &cobra.Command{...} directly
// to capturing in a variable, adding ValidArgsFunction, then returning.
// Example for newProfileDeleteCmd:
func newProfileDeleteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		// ... existing fields unchanged ...
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeProfileNames(app, toComplete)
	}
	return cmd
}
```

For `newProfileAddAssetCmd`, which takes `<profile> <asset>`:

```go
cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeProfileNames(app, toComplete)
	}
	if len(args) == 1 {
		// Complete asset names for second argument
		completionInitApp(app)
		summary, err := app.ScanIndex()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var names []string
		for _, a := range summary.Index.All() {
			name := fmt.Sprintf("%s/%s", a.Type, a.Name)
			if toComplete == "" || strings.HasPrefix(name, toComplete) || strings.HasPrefix(a.Name, toComplete) {
				names = append(names, fmt.Sprintf("%s/%s\t%s from %s", a.Type, a.Name, a.Type, a.SourceID))
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestProfile.*Completions -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/profile.go cmd/profile_test.go cmd/helpers.go
git commit -m "feat(cli): add dynamic completions for profile subcommands"
```

### Task 6: Add ValidArgsFunction to snapshot and source subcommands

**Files:**

- Modify: `cmd/snapshot.go:74-240`
- Modify: `cmd/source.go:89-184`
- [ ] **Step 1: Write failing tests**

Add to `cmd/snapshot_test.go`:

```go
func TestSnapshotRestoreCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "snapshot", "restore", ""})

	_ = rootCmd.Execute()

	// Completions may be empty if no snapshots exist; the key test is no panic/error
}
```

Add to `cmd/source_test.go`:

```go
func TestSourceRemoveCmd_Completions(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{ConfigPath: configPath}
	completionInitApp(app)
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "source", "remove", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "my-source") {
		t.Errorf("expected 'my-source' in source remove completions, got:\n%s", got)
	}
}
```

Note: Uses `setupDeployEnv` from `deploy_test.go` which pre-registers `my-source`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run "TestSnapshotRestoreCmd_Completions|TestSourceRemoveCmd_Completions" -v`
Expected: FAIL

- [ ] **Step 3: Add completion helpers and wire them**

In `cmd/helpers.go`, add:

```go
// completeSnapshotNames returns snapshot names for shell completion.
func completeSnapshotNames(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	pstore, err := app.ProfileStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	snapshots, err := pstore.ListSnapshots()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, s := range snapshots {
		if toComplete == "" || strings.HasPrefix(s.Name, toComplete) {
			desc := fmt.Sprintf("%d deployments", s.DeploymentCount)
			if s.Auto {
				desc += " (auto)"
			}
			names = append(names, fmt.Sprintf("%s\t%s", s.Name, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeSourceIDs returns source IDs for shell completion.
func completeSourceIDs(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	sm, err := app.SourceManager()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, s := range sm.Sources() {
		if toComplete == "" || strings.HasPrefix(s.ID, toComplete) {
			desc := string(s.Type)
			if s.Alias != "" {
				desc = s.Alias
			}
			names = append(names, fmt.Sprintf("%s\t%s", s.ID, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
```

Wire `ValidArgsFunction` in `cmd/snapshot.go` for `newSnapshotRestoreCmd` and `newSnapshotDeleteCmd`:

```go
cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return completeSnapshotNames(app, toComplete)
}
```

Wire in `cmd/source.go` for `newSourceRemoveCmd`:

```go
cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return completeSourceIDs(app, toComplete)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run "TestSnapshot.*Completions|TestSource.*Completions" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/snapshot.go cmd/source.go cmd/helpers.go cmd/snapshot_test.go cmd/source_test.go
git commit -m "feat(cli): add dynamic completions for snapshot and source subcommands"
```

## Chunk 3: Flag completions and final verification

### Task 7: Register flag completion functions

**Files:**

- Modify: `cmd/root.go:19-67` (register `--scope` completion on rootCmd)
- Modify: `cmd/deploy.go:142` (register `--type` completion)
- Modify: `cmd/list.go:113-115` (register `--type` and `--source` completions)
- Modify: `cmd/sync.go:91` (register `--source` completion)
- [ ] **Step 1: Write failing test**

Add to `cmd/root_test.go`:

```go
func TestScopeFlagCompletion(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "list", "--scope", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "global") || !strings.Contains(got, "project") {
		t.Errorf("expected 'global' and 'project' in scope completions, got:\n%s", got)
	}
}
```

Add to `cmd/list_test.go`:

```go
func TestListCmd_TypeFlagCompletion(t *testing.T) {
	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"__complete", "list", "--type", ""})

	_ = rootCmd.Execute()

	got := out.String()
	if !strings.Contains(got, "skills") || !strings.Contains(got, "commands") {
		t.Errorf("expected asset types in type flag completions, got:\n%s", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run "TestScopeFlagCompletion|TestListCmd_TypeFlagCompletion" -v`
Expected: FAIL — no flag completions registered

- [ ] **Step 3: Register flag completion functions**

In `cmd/root.go`, after `rootCmd.MarkFlagsMutuallyExclusive(...)` (line 45), add:

```go
rootCmd.RegisterFlagCompletionFunc("scope", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"global\tDeploy to ~/.claude/", "project\tDeploy to .claude/ in project"}, cobra.ShellCompDirectiveNoFileComp
})
```

In `cmd/deploy.go`, after `cmd.Flags().StringVar(&assetType, "type", ...)`, add:

```go
cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    var types []string
    for _, t := range nd.AllAssetTypes() {
        types = append(types, string(t))
    }
    return types, cobra.ShellCompDirectiveNoFileComp
})
```

In `cmd/list.go`, after the three `cmd.Flags()` calls (lines 113-115), add:

```go
cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    var types []string
    for _, t := range nd.AllAssetTypes() {
        types = append(types, string(t))
    }
    return types, cobra.ShellCompDirectiveNoFileComp
})
cmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return completeSourceIDs(app, toComplete)
})
```

In `cmd/sync.go`, after `cmd.Flags().StringVar(&sourceID, "source", ...)`, add:

```go
cmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return completeSourceIDs(app, toComplete)
})
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run "TestScopeFlagCompletion|TestListCmd_TypeFlagCompletion" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/root.go cmd/deploy.go cmd/list.go cmd/sync.go cmd/root_test.go cmd/list_test.go
git commit -m "feat(cli): register flag completion functions for --scope, --type, --source"
```

### Task 8: Full test suite verification and gofumpt

- [ ] **Step 1: Run gofumpt on all modified files**

Run: `gofumpt -w cmd/completion.go cmd/helpers.go cmd/root.go cmd/deploy.go cmd/remove.go cmd/pin.go cmd/profile.go cmd/snapshot.go cmd/source.go cmd/list.go cmd/sync.go`

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -timeout 120s`
Expected: All 19+ packages pass, 0 failures

- [ ] **Step 3: Run build**

Run: `go build ./...`
Expected: exit 0

- [ ] **Step 4: Run linter if available**

Run: `golangci-lint run ./cmd/... 2>&1 || true`
Expected: No new warnings in modified files

- [ ] **Step 5: Verify completion output manually**

Run: `go run . completion bash | head -5`
Expected: Bash completion script header

Run: `go run . completion zsh | head -5`
Expected: `#compdef nd` header

Run: `go run . completion fish | head -5`
Expected: Fish completion directives

- [ ] **Step 6: Commit formatting changes (if any)**

```bash
git add cmd/completion.go cmd/helpers.go cmd/root.go cmd/deploy.go cmd/remove.go cmd/pin.go cmd/profile.go cmd/snapshot.go cmd/source.go cmd/list.go cmd/sync.go
git commit -m "style(cli): apply gofumpt formatting to completion files"
```

- [ ] **Step 7: Final commit or squash**

If all green, the feature is complete. The branch is ready for PR.
