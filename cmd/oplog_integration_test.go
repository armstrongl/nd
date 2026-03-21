package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
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
