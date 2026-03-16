package version

import "fmt"

// Set via ldflags at build time:
//
//	go build -ldflags "-X github.com/armstrongl/nd/internal/version.Version=v0.1.0 ..."
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a formatted version string for display.
func String() string {
	return fmt.Sprintf("nd version %s (commit: %s, built: %s)", Version, Commit, Date)
}
