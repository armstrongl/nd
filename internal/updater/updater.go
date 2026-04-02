package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	cacheFileName = "update-check.json"
	cacheMaxAge   = 24 * time.Hour
	githubAPIURL  = "https://api.github.com/repos/armstrongl/nd/releases/latest"
	checkTimeout  = 5 * time.Second
)

type cacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

// IsBrewInstall reports whether the running binary was installed via Homebrew.
// It resolves the real executable path (following symlinks) and checks for
// Homebrew path patterns on macOS and Linux.
func IsBrewInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return false
	}
	lower := strings.ToLower(exe)
	return strings.Contains(lower, "/cellar/") || strings.Contains(lower, "/homebrew/")
}

// CheckCached returns the latest version string from the on-disk cache if the
// cache exists and is younger than 24 hours. Returns an empty string (not an
// error) when the cache is missing, stale, or unreadable.
func CheckCached(cacheDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, cacheFileName))
	if err != nil {
		return "", nil
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", nil
	}
	if time.Since(entry.CheckedAt) > cacheMaxAge {
		return "", nil
	}
	return entry.LatestVersion, nil
}

// RefreshAsync fetches the latest release version from GitHub in a background
// goroutine and persists the result to the cache file. Errors are discarded
// silently so this never blocks or fails a command.
func RefreshAsync(cacheDir string) {
	go func() {
		latest, err := fetchLatestVersion()
		if err != nil || latest == "" {
			return
		}
		entry := cacheEntry{
			LatestVersion: latest,
			CheckedAt:     time.Now().UTC(),
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return
		}
		_ = os.WriteFile(filepath.Join(cacheDir, cacheFileName), data, 0o600)
	}()
}

// IsNewer reports whether candidate is a strictly newer semantic version than
// current. Both strings may optionally carry a "v" prefix.
func IsNewer(candidate, current string) bool {
	cv := parseSemver(candidate)
	cc := parseSemver(current)
	for i := range cv {
		if cv[i] > cc[i] {
			return true
		}
		if cv[i] < cc[i] {
			return false
		}
	}
	return false
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: checkTimeout}
	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return strings.TrimPrefix(payload.TagName, "v"), nil
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		p = strings.SplitN(p, "-", 2)[0] // strip pre-release suffix
		out[i], _ = strconv.Atoi(p)
	}
	return out
}
