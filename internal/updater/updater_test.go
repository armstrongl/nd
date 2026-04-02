package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	t.Helper()
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"0.4.0", "0.3.0", true},
		{"v0.4.0", "v0.3.0", true},
		{"0.3.0", "0.3.0", false},
		{"0.2.0", "0.3.0", false},
		{"1.0.0", "0.9.9", true},
		{"0.3.1", "0.3.0", true},
		{"0.3.0", "0.3.1", false},
		// dev builds report 0.0.0 — a real release should never appear newer
		{"0.0.0", "dev", false},
		{"0.1.0", "dev", true},
		// pre-release suffix stripped
		{"0.4.0-beta", "0.3.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.candidate+"_vs_"+tt.current, func(t *testing.T) {
			got := IsNewer(tt.candidate, tt.current)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.candidate, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheckCached_missing(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	got, err := CheckCached(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for missing cache, got %q", got)
	}
}

func TestCheckCached_fresh(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeCache(t, dir, "0.5.0", time.Now().UTC())

	got, err := CheckCached(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.5.0" {
		t.Errorf("expected %q, got %q", "0.5.0", got)
	}
}

func TestCheckCached_stale(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeCache(t, dir, "0.5.0", time.Now().UTC().Add(-25*time.Hour))

	got, err := CheckCached(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for stale cache, got %q", got)
	}
}

func TestCheckCached_corrupt(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, cacheFileName)
	if err := os.WriteFile(path, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := CheckCached(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for corrupt cache, got %q", got)
	}
}

// writeCache is a test helper that writes a cache entry to dir.
func writeCache(t *testing.T, dir, version string, at time.Time) {
	t.Helper()
	entry := cacheEntry{LatestVersion: version, CheckedAt: at}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), data, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}
