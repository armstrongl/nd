package nd

// Context file name constants for the built-in context file names.
const (
	ContextCLAUDE      = "CLAUDE.md"
	ContextAGENTS      = "AGENTS.md"
	ContextCLAUDELocal = "CLAUDE.local.md"
	ContextAGENTSLocal = "AGENTS.local.md"
)

// BuiltinContextFileNames returns all built-in context file name constants.
func BuiltinContextFileNames() []string {
	return []string{
		ContextCLAUDE, ContextAGENTS, ContextCLAUDELocal, ContextAGENTSLocal,
	}
}

// IsLocalOnlyContext returns true if a context filename deploys only at project scope.
// Works for both built-in and custom context file types (checks the .local.md suffix).
func IsLocalOnlyContext(filename string) bool {
	return len(filename) > 9 && filename[len(filename)-9:] == ".local.md"
}
