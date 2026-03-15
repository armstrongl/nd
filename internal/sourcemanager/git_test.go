package sourcemanager_test

import (
	"testing"

	"github.com/larah/nd/internal/sourcemanager"
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
