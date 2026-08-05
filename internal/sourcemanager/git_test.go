package sourcemanager_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/armstrongl/nd/internal/sourcemanager"
)

func TestExpandGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"owner/repo", "https://github.com/owner/repo.git"},
		{"my-org/my-skills", "https://github.com/my-org/my-skills.git"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"https://gitlab.com/org/repo.git", "https://gitlab.com/org/repo.git"},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git"},
		{"git@gitlab.com:org/repo.git", "git@gitlab.com:org/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "ssh://git@github.com/owner/repo.git"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sourcemanager.ExpandGitURL(tt.input)
			if got != tt.want {
				t.Errorf("ExpandGitURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/owner/my-skills.git", "my-skills"},
		{"https://github.com/owner/repo", "repo"},
		{"git@github.com:owner/repo.git", "repo"},
		{"owner/repo", "repo"},
		{"owner/my-cool-skills", "my-cool-skills"},
		{"ssh://git@github.com/owner/repo.git", "repo"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sourcemanager.RepoNameFromURL(tt.input)
			if got != tt.want {
				t.Errorf("RepoNameFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestAddGitCloneDoesNotPassStdin is a regression test for the --stdin error.
// When git clone's stdout was captured via CombinedOutput(), git's internal
// index-pack subprocess could fail with "fatal: --stdin requires a git
// repository" because the pipe interfered with fetch-pack/index-pack
// communication. This test clones a repo with multiple commits (forcing
// pack transfer) and verifies no such error occurs.
func TestAddGitCloneDoesNotPassStdin(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	sm, err := sourcemanager.New(configPath, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create a repo with enough content to generate pack data during clone.
	repoDir := filepath.Join(t.TempDir(), "upstream")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	run("init", repoDir)
	testFile := filepath.Join(repoDir, "README.md")
	os.WriteFile(testFile, []byte("# test repo\n"), 0o644)
	run("-C", repoDir, "add", ".")
	run("-C", repoDir, "commit", "-m", "first")
	os.WriteFile(testFile, []byte("# test repo\n\nupdated content\n"), 0o644)
	run("-C", repoDir, "add", ".")
	run("-C", repoDir, "commit", "-m", "second")

	src, err := sm.AddGit(repoDir, "")
	if err != nil {
		t.Fatalf("AddGit failed (possible --stdin regression): %v", err)
	}

	// Verify clone is a valid repo with the expected commits.
	cmd := exec.Command("git", "-C", src.Path, "rev-list", "--count", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-list: %v", err)
	}
	if got := string(out[:len(out)-1]); got != "2" {
		t.Errorf("expected 2 commits in clone, got %s", got)
	}
}
