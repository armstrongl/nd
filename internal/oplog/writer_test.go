package oplog_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
)

func TestWriterCreatesDirectoryAndFile(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")

	w := oplog.NewWriter(logDir)
	entry := oplog.LogEntry{
		Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		Operation: oplog.OpDeploy,
		Assets:    []asset.Identity{{SourceID: "src", Type: nd.AssetSkill, Name: "greeting"}},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
	}

	if err := w.Log(entry); err != nil {
		t.Fatalf("Log() error: %v", err)
	}

	logPath := filepath.Join(logDir, "operations.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestWriterAppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir)

	entries := []oplog.LogEntry{
		{
			Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			Operation: oplog.OpDeploy,
			Assets:    []asset.Identity{{SourceID: "s1", Type: nd.AssetSkill, Name: "a"}},
			Scope:     nd.ScopeGlobal,
			Succeeded: 1,
		},
		{
			Timestamp: time.Date(2026, 3, 21, 10, 5, 0, 0, time.UTC),
			Operation: oplog.OpRemove,
			Assets:    []asset.Identity{{SourceID: "s1", Type: nd.AssetSkill, Name: "a"}},
			Scope:     nd.ScopeGlobal,
			Succeeded: 1,
		},
	}

	for _, e := range entries {
		if err := w.Log(e); err != nil {
			t.Fatalf("Log() error: %v", err)
		}
	}

	logPath := filepath.Join(dir, "operations.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Each line must be valid JSON that round-trips to LogEntry
	for i, line := range lines {
		var got oplog.LogEntry
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
			continue
		}
		if got.Operation != entries[i].Operation {
			t.Errorf("line %d: operation = %q, want %q", i, got.Operation, entries[i].Operation)
		}
		if got.Succeeded != entries[i].Succeeded {
			t.Errorf("line %d: succeeded = %d, want %d", i, got.Succeeded, entries[i].Succeeded)
		}
	}
}

func TestWriterRotatesAtMaxSize(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir, oplog.WithMaxSize(500)) // 500 bytes for testing

	logPath := filepath.Join(dir, "operations.log")
	rotatedPath := filepath.Join(dir, "operations.log.1")

	// Write entries until we exceed the max size
	entry := oplog.LogEntry{
		Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		Operation: oplog.OpDeploy,
		Assets:    []asset.Identity{{SourceID: "my-source", Type: nd.AssetSkill, Name: "some-skill-name"}},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
		Detail:    "padding to make the entry larger",
	}

	// Write enough entries to exceed 500 bytes (each entry is ~200 bytes)
	for i := 0; i < 4; i++ {
		if err := w.Log(entry); err != nil {
			t.Fatalf("Log() iteration %d error: %v", i, err)
		}
	}

	// After rotation, the old file should exist as .log.1
	if _, err := os.Stat(rotatedPath); err != nil {
		t.Fatalf("rotated file not created: %v", err)
	}

	// The current log should have fewer entries (post-rotation writes)
	current, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read current log: %v", err)
	}

	rotated, err := os.ReadFile(rotatedPath)
	if err != nil {
		t.Fatalf("read rotated log: %v", err)
	}

	// Rotated file should be non-empty
	if len(rotated) == 0 {
		t.Error("rotated file is empty")
	}

	// Current file should be smaller than max size
	if len(current) > 500 {
		t.Errorf("current log size %d exceeds max 500", len(current))
	}
}

func TestWriterRotationOverwritesPreviousBackup(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir, oplog.WithMaxSize(300))

	entry := oplog.LogEntry{
		Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		Operation: oplog.OpDeploy,
		Assets:    []asset.Identity{{SourceID: "src", Type: nd.AssetSkill, Name: "skill-a"}},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
		Detail:    "padding to increase entry size",
	}

	// Write enough to trigger rotation twice
	for i := 0; i < 10; i++ {
		if err := w.Log(entry); err != nil {
			t.Fatalf("Log() iteration %d error: %v", i, err)
		}
	}

	rotatedPath := filepath.Join(dir, "operations.log.1")
	if _, err := os.Stat(rotatedPath); err != nil {
		t.Fatalf("rotated file should exist: %v", err)
	}

	// Only .log and .log.1 should exist (no .log.2, etc.)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log.2") {
			t.Errorf("unexpected second rotation file: %s", e.Name())
		}
	}
}

func TestWriterAllOperationTypes(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir)

	ops := []oplog.OperationType{
		oplog.OpDeploy, oplog.OpRemove, oplog.OpSync,
		oplog.OpProfileSwitch, oplog.OpSnapshotSave, oplog.OpSnapshotRestore,
		oplog.OpSourceAdd, oplog.OpSourceRemove, oplog.OpSourceSync,
		oplog.OpUninstall,
	}

	for _, op := range ops {
		if err := w.Log(oplog.LogEntry{
			Timestamp: time.Now(),
			Operation: op,
			Succeeded: 1,
		}); err != nil {
			t.Errorf("Log(%s) error: %v", op, err)
		}
	}

	logPath := filepath.Join(dir, "operations.log")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var count int
	for scanner.Scan() {
		count++
		var entry oplog.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Errorf("line %d: invalid JSON: %v", count, err)
		}
	}
	if count != len(ops) {
		t.Errorf("expected %d lines, got %d", len(ops), count)
	}
}

func TestWriterPartialFailureEntry(t *testing.T) {
	dir := t.TempDir()
	w := oplog.NewWriter(dir)

	entry := oplog.LogEntry{
		Timestamp: time.Now(),
		Operation: oplog.OpDeploy,
		Assets: []asset.Identity{
			{SourceID: "s", Type: nd.AssetSkill, Name: "ok"},
			{SourceID: "s", Type: nd.AssetSkill, Name: "fail"},
		},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
		Failed:    1,
		Detail:    "skills/fail: source not found",
	}

	if err := w.Log(entry); err != nil {
		t.Fatalf("Log() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "operations.log"))
	if err != nil {
		t.Fatal(err)
	}

	var got oplog.LogEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Failed != 1 {
		t.Errorf("failed = %d, want 1", got.Failed)
	}
	if got.Detail == "" {
		t.Error("detail should contain failure info")
	}
}
