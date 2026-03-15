package sourcemanager

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExpandGitURL expands GitHub shorthand (owner/repo) to a full HTTPS URL.
// Full URLs (HTTPS, SSH, ssh://) are returned as-is.
func ExpandGitURL(input string) string {
	if strings.Contains(input, "://") || strings.HasPrefix(input, "git@") {
		return input
	}
	// GitHub shorthand: owner/repo
	parts := strings.SplitN(input, "/", 2)
	if len(parts) == 2 && !strings.Contains(parts[0], ".") {
		return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1])
	}
	return input
}

// RepoNameFromURL extracts the repository name from a Git URL or shorthand.
func RepoNameFromURL(url string) string {
	// Handle shorthand: owner/repo
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, "/", 2)
		if len(parts) == 2 {
			return strings.TrimSuffix(parts[1], ".git")
		}
	}

	// Handle git@host:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		if idx := strings.LastIndex(url, "/"); idx >= 0 {
			return strings.TrimSuffix(url[idx+1:], ".git")
		}
		if idx := strings.LastIndex(url, ":"); idx >= 0 {
			return strings.TrimSuffix(url[idx+1:], ".git")
		}
	}

	// Handle https://host/owner/repo.git
	base := filepath.Base(url)
	return strings.TrimSuffix(base, ".git")
}

// gitClone clones a repository to the target directory.
func gitClone(url, targetDir string) error {
	cmd := exec.Command("git", "clone", url, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %s: %w", url, string(output), err)
	}
	return nil
}

// gitPull runs git pull --ff-only in the given directory.
func gitPull(repoDir string) error {
	cmd := exec.Command("git", "-C", repoDir, "pull", "--ff-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull in %s: %s: %w", repoDir, string(output), err)
	}
	return nil
}
