package nd

// SourceType distinguishes local directories from Git repos.
type SourceType string

const (
	SourceLocal SourceType = "local"
	SourceGit   SourceType = "git"
)
