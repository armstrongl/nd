package tui

import (
	"os"
	"path/filepath"
)

// helpSeenPath returns the location of the first-run help flag file. It lives
// beside nd's other state (e.g. state/deployments.yaml), derived from the
// config directory so it is scope-independent.
func helpSeenPath(svc Services) string {
	return filepath.Join(filepath.Dir(svc.GetConfigPath()), "state", "help_seen")
}

// helpSeen reports whether the first-run help tip has already been dismissed.
func helpSeen(svc Services) bool {
	_, err := os.Stat(helpSeenPath(svc))
	return err == nil
}

// markHelpSeen persists the first-run help flag so the tip never reappears.
func markHelpSeen(svc Services) error {
	path := helpSeenPath(svc)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("1\n"), 0o644)
}
