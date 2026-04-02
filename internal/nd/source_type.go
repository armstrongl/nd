package nd

// SourceType distinguishes local directories, Git repos, and built-in sources.
type SourceType string

const (
	SourceLocal   SourceType = "local"
	SourceGit     SourceType = "git"
	SourceBuiltin SourceType = "builtin"
)

// BuiltinSourceID is the reserved source ID for the built-in source.
// User sources cannot use this ID.
const BuiltinSourceID = "builtin"
