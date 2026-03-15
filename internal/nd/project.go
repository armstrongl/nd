package nd

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectRoot walks up from startDir looking for a directory containing
// .git/ or .claude/. Returns the first match or an error if the filesystem
// root is reached without finding either marker.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve start dir: %w", err)
	}

	for {
		for _, marker := range []string{".git", ".claude"} {
			info, err := os.Stat(filepath.Join(dir, marker))
			if err == nil && info.IsDir() {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no project root found (looked for .git/ or .claude/ from %s)", startDir)
		}
		dir = parent
	}
}
