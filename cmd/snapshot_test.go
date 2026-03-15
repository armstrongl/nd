package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/larah/nd/internal/output"
)

func TestSnapshotSaveCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something first so there's state to snapshot
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Save snapshot
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "snapshot", "save", "test-snap"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "test-snap") {
		t.Errorf("expected snapshot name in output, got: %s", got)
	}
}

func TestSnapshotSaveCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "snapshot", "save", "json-snap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestSnapshotSaveCmd_Duplicate(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Save once
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "save", "dup-snap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Save again — should fail
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "snapshot", "save", "dup-snap"})
	if err := rootCmd2.Execute(); err == nil {
		t.Fatal("expected error for duplicate snapshot")
	}
}

func TestSnapshotListCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Save a snapshot
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "save", "list-snap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// List snapshots
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "snapshot", "list"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "list-snap") {
		t.Errorf("expected snapshot name in output, got: %s", got)
	}
}

func TestSnapshotListCmd_Empty(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No snapshots") {
		t.Errorf("expected 'No snapshots' in output, got: %s", got)
	}
}

func TestSnapshotListCmd_JSON(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--json", "snapshot", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp output.JSONResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

func TestSnapshotDeleteCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Save a snapshot
	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "save", "del-snap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Delete it
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "snapshot", "delete", "del-snap"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Deleted") {
		t.Errorf("expected 'Deleted' in output, got: %s", got)
	}
}

func TestSnapshotDeleteCmd_NotFound(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	app := &App{}
	rootCmd := NewRootCmd(app)
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "--yes", "snapshot", "delete", "nonexistent"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for nonexistent snapshot")
	}
}

func TestSnapshotRestoreCmd(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something first so snapshot has content
	app0 := &App{}
	rootCmd0 := NewRootCmd(app0)
	var out bytes.Buffer
	rootCmd0.SetOut(&out)
	rootCmd0.SetErr(&out)
	rootCmd0.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd0.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Save a snapshot
	app := &App{}
	rootCmd := NewRootCmd(app)
	out.Reset()
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "save", "restore-test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Restore with --yes (skips confirmation)
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--yes", "snapshot", "restore", "restore-test"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "restored") || !strings.Contains(got, "restore-test") {
		t.Errorf("expected restore confirmation in output, got: %s", got)
	}
}

func TestSnapshotRestoreCmd_DryRun(t *testing.T) {
	configPath, _ := setupDeployEnv(t)

	// Deploy something first so snapshot has content
	app0 := &App{}
	rootCmd0 := NewRootCmd(app0)
	var out bytes.Buffer
	rootCmd0.SetOut(&out)
	rootCmd0.SetErr(&out)
	rootCmd0.SetArgs([]string{"--config", configPath, "deploy", "greeting"})
	if err := rootCmd0.Execute(); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Save a snapshot
	app := &App{}
	rootCmd := NewRootCmd(app)
	out.Reset()
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--config", configPath, "snapshot", "save", "restore-snap"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Dry-run restore
	app2 := &App{}
	rootCmd2 := NewRootCmd(app2)
	out.Reset()
	rootCmd2.SetOut(&out)
	rootCmd2.SetErr(&out)
	rootCmd2.SetArgs([]string{"--config", configPath, "--dry-run", "snapshot", "restore", "restore-snap"})
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("dry-run restore failed: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got: %s", got)
	}
}
