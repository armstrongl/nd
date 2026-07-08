package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/oplog"
)

// logDir returns the oplog directory derived from the config path.
func logDir(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "logs")
}

// readLogEntries reads all JSONL entries from the operations.log in logDir.
func readLogEntries(t *testing.T, dir string) []oplog.LogEntry {
	t.Helper()
	path := filepath.Join(dir, "operations.log")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()

	var entries []oplog.LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e oplog.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("invalid log entry: %v", err)
		}
		entries = append(entries, e)
	}
	return entries
}

func TestOplog_DeploySingleWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Operation != oplog.OpDeploy {
		t.Errorf("operation = %q, want %q", entries[0].Operation, oplog.OpDeploy)
	}
	if entries[0].Succeeded != 1 {
		t.Errorf("succeeded = %d, want 1", entries[0].Succeeded)
	}
	if entries[0].Failed != 0 {
		t.Errorf("failed = %d, want 0", entries[0].Failed)
	}
}

func TestOplog_DeployBulkWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting", "hello.md"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Succeeded != 2 {
		t.Errorf("succeeded = %d, want 2", entries[0].Succeeded)
	}
	if len(entries[0].Assets) != 2 {
		t.Errorf("assets count = %d, want 2", len(entries[0].Assets))
	}
}

func TestOplog_RemoveWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy first
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	// Remove
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "remove", "greeting"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("remove: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	// Should have 2 entries: deploy + remove
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[1].Operation != oplog.OpRemove {
		t.Errorf("operation = %q, want %q", entries[1].Operation, oplog.OpRemove)
	}
}

func TestOplog_SyncWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "sync"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Operation != oplog.OpSync {
		t.Errorf("operation = %q, want %q", entries[0].Operation, oplog.OpSync)
	}
}

func TestOplog_DryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--dry-run", "deploy", "greeting"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy --dry-run: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 0 {
		t.Errorf("expected 0 log entries for dry-run, got %d", len(entries))
	}
}

// runND executes a single nd command against configPath and returns stdout,
// failing the test if the command errors.
func runND(t *testing.T, configPath string, args ...string) string {
	t.Helper()
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append([]string{"--config", configPath}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("nd %v: %v\n%s", args, err, out.String())
	}
	return out.String()
}

// countOp counts log entries with the given operation type.
func countOp(entries []oplog.LogEntry, op oplog.OperationType) int {
	n := 0
	for _, e := range entries {
		if e.Operation == op {
			n++
		}
	}
	return n
}

// findOp returns the first log entry with the given operation type.
func findOp(entries []oplog.LogEntry, op oplog.OperationType) (oplog.LogEntry, bool) {
	for _, e := range entries {
		if e.Operation == op {
			return e, true
		}
	}
	return oplog.LogEntry{}, false
}

// gitTestCmd runs git hermetically (no dependence on user/global git identity).
func gitTestCmd(t *testing.T, args ...string) {
	t.Helper()
	base := []string{
		"-c", "user.email=test@example.com",
		"-c", "user.name=nd test",
		"-c", "commit.gpgsign=false",
		"-c", "init.defaultBranch=main",
	}
	cmd := exec.Command("git", append(base, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

// seededBareRepo creates a bare git repo (basename == name) seeded with one
// commit on main, and returns its absolute path. Cloning it produces a working
// tree with upstream tracking so `git pull --ff-only` succeeds.
func seededBareRepo(t *testing.T, name string) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), name)
	gitTestCmd(t, "init", "--bare", bare)

	seed := t.TempDir()
	gitTestCmd(t, "clone", bare, seed)
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitTestCmd(t, "-C", seed, "add", ".")
	gitTestCmd(t, "-C", seed, "commit", "-m", "initial")
	gitTestCmd(t, "-C", seed, "push", "origin", "HEAD:refs/heads/main")
	return bare
}

func TestOplog_ProfileDeployWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "profile", "create", "pd-test", "--assets", "skills/greeting")
	runND(t, configPath, "profile", "deploy", "pd-test")

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpDeploy)
	if !ok {
		t.Fatalf("expected an OpDeploy entry from profile deploy, got: %v", entries)
	}
	if e.Detail != "profile:pd-test" {
		t.Errorf("detail = %q, want %q", e.Detail, "profile:pd-test")
	}
}

func TestOplog_ProfileSwitchWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "profile", "create", "prof-a", "--assets", "skills/greeting")
	runND(t, configPath, "profile", "create", "prof-b", "--assets", "commands/hello.md")
	runND(t, configPath, "profile", "deploy", "prof-a")
	runND(t, configPath, "--yes", "profile", "switch", "prof-b")

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpProfileSwitch)
	if !ok {
		t.Fatalf("expected an OpProfileSwitch entry, got: %v", entries)
	}
	if e.Detail != "prof-a -> prof-b" {
		t.Errorf("detail = %q, want %q", e.Detail, "prof-a -> prof-b")
	}
}

func TestOplog_SnapshotRestoreWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "deploy", "greeting")
	runND(t, configPath, "snapshot", "save", "restore-snap")
	runND(t, configPath, "--yes", "snapshot", "restore", "restore-snap")

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpSnapshotRestore)
	if !ok {
		t.Fatalf("expected an OpSnapshotRestore entry, got: %v", entries)
	}
	if e.Detail != "restore-snap" {
		t.Errorf("detail = %q, want %q", e.Detail, "restore-snap")
	}
}

func TestOplog_SourceAddLocalWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	newSrc := filepath.Join(t.TempDir(), "extra-src")
	if err := os.MkdirAll(newSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	runND(t, configPath, "source", "add", newSrc)

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpSourceAdd)
	if !ok {
		t.Fatalf("expected an OpSourceAdd entry, got: %v", entries)
	}
	if e.Detail != "local extra-src" {
		t.Errorf("detail = %q, want %q", e.Detail, "local extra-src")
	}
}

func TestOplog_SourceAddGitWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	bare := seededBareRepo(t, "gitsrc")
	runND(t, configPath, "source", "add", "file://"+bare)

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpSourceAdd)
	if !ok {
		t.Fatalf("expected an OpSourceAdd entry, got: %v", entries)
	}
	if e.Detail != "git gitsrc" {
		t.Errorf("detail = %q, want %q", e.Detail, "git gitsrc")
	}
}

func TestOplog_SourceRemoveWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// my-source is pre-registered by setupDeployEnv with no deployments.
	runND(t, configPath, "--yes", "source", "remove", "my-source")

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpSourceRemove)
	if !ok {
		t.Fatalf("expected an OpSourceRemove entry, got: %v", entries)
	}
	if e.Detail != "my-source" {
		t.Errorf("detail = %q, want %q", e.Detail, "my-source")
	}
}

func TestOplog_SourceSyncWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	bare := seededBareRepo(t, "syncrepo")
	runND(t, configPath, "source", "add", "file://"+bare)
	runND(t, configPath, "sync", "--source", "syncrepo")

	entries := readLogEntries(t, logDir(configPath))
	e, ok := findOp(entries, oplog.OpSourceSync)
	if !ok {
		t.Fatalf("expected an OpSourceSync entry, got: %v", entries)
	}
	if e.Detail != "syncrepo" {
		t.Errorf("detail = %q, want %q", e.Detail, "syncrepo")
	}
}

func TestOplog_UninstallWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "deploy", "greeting")
	runND(t, configPath, "--yes", "uninstall")

	entries := readLogEntries(t, logDir(configPath))
	if countOp(entries, oplog.OpUninstall) != 1 {
		t.Fatalf("expected 1 OpUninstall entry, got: %v", entries)
	}
}

// --- Dry-run no-log: the dry-run guard must precede every LogOp call. ---

func TestOplog_RemoveDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "deploy", "greeting")
	runND(t, configPath, "--yes", "--dry-run", "remove", "greeting")

	entries := readLogEntries(t, logDir(configPath))
	if countOp(entries, oplog.OpRemove) != 0 {
		t.Errorf("dry-run remove must not log an OpRemove entry, got: %v", entries)
	}
}

func TestOplog_SnapshotRestoreDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "deploy", "greeting")
	runND(t, configPath, "snapshot", "save", "dr-snap")
	runND(t, configPath, "--dry-run", "snapshot", "restore", "dr-snap")

	entries := readLogEntries(t, logDir(configPath))
	if countOp(entries, oplog.OpSnapshotRestore) != 0 {
		t.Errorf("dry-run restore must not log an OpSnapshotRestore entry, got: %v", entries)
	}
}

func TestOplog_ProfileDeployDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "profile", "create", "dr-pd", "--assets", "skills/greeting")
	runND(t, configPath, "--dry-run", "profile", "deploy", "dr-pd")

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 0 {
		t.Errorf("dry-run profile deploy must not log anything, got: %v", entries)
	}
}

func TestOplog_ProfileSwitchDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "profile", "create", "ds-a", "--assets", "skills/greeting")
	runND(t, configPath, "profile", "create", "ds-b", "--assets", "commands/hello.md")
	runND(t, configPath, "profile", "deploy", "ds-a")
	runND(t, configPath, "--dry-run", "profile", "switch", "ds-b")

	entries := readLogEntries(t, logDir(configPath))
	if countOp(entries, oplog.OpProfileSwitch) != 0 {
		t.Errorf("dry-run switch must not log an OpProfileSwitch entry, got: %v", entries)
	}
}

func TestOplog_SyncDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "--dry-run", "sync")

	entries := readLogEntries(t, logDir(configPath))
	if len(entries) != 0 {
		t.Errorf("dry-run sync must not log anything, got: %v", entries)
	}
}

func TestOplog_UninstallDryRunDoesNotLog(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	runND(t, configPath, "deploy", "greeting")
	runND(t, configPath, "--dry-run", "uninstall")

	entries := readLogEntries(t, logDir(configPath))
	if countOp(entries, oplog.OpUninstall) != 0 {
		t.Errorf("dry-run uninstall must not log an OpUninstall entry, got: %v", entries)
	}
}

func TestOplog_SnapshotSaveWritesEntry(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something first to have state
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	// Save snapshot
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "snapshot", "save", "test-snap"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("snapshot save: %v", err)
	}

	entries := readLogEntries(t, logDir(configPath))
	// deploy entry + snapshot-save entry
	var found bool
	for _, e := range entries {
		if e.Operation == oplog.OpSnapshotSave {
			found = true
			if e.Detail != "test-snap" {
				t.Errorf("detail = %q, want %q", e.Detail, "test-snap")
			}
		}
	}
	if !found {
		t.Errorf("expected a snapshot-save log entry, got operations: %v", entries)
	}
}
