package sourcemanager

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestGitCloneNonZeroExit exercises git.go:62-64: a non-zero `git clone` exit
// is wrapped as "git clone <url>: ...". Cloning a nonexistent local repository
// fails fast without any network access.
func TestGitCloneNonZeroExit(t *testing.T) {
	target := filepath.Join(t.TempDir(), "clone")
	err := gitClone("/nonexistent/repo/does-not-exist", target)
	if err == nil {
		t.Fatal("expected error cloning a nonexistent repository")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("expected wrapped git clone error, got: %v", err)
	}
}

// TestGitPullNonZeroExit exercises git.go:72-74: a non-zero `git pull` exit is
// wrapped as "git pull in <dir>: ...". Running pull in a directory that is not
// a git repository fails without any network access.
func TestGitPullNonZeroExit(t *testing.T) {
	err := gitPull(t.TempDir())
	if err == nil {
		t.Fatal("expected error running git pull outside a repository")
	}
	if !strings.Contains(err.Error(), "git pull in") {
		t.Errorf("expected wrapped git pull error, got: %v", err)
	}
}
