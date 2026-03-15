package nd

import "fmt"

// PathTraversalError is returned when a path escapes its allowed root (NFR-012).
type PathTraversalError struct {
	Path     string // The offending path
	Root     string // The root it should be confined to
	SourceID string // Which source contained the path
}

func (e *PathTraversalError) Error() string {
	return fmt.Sprintf("path %q escapes source root %q in source %s", e.Path, e.Root, e.SourceID)
}

// LockError is returned when the state file lock cannot be acquired (NFR-011).
type LockError struct {
	Path    string // The file being locked
	Timeout string // How long we waited
	Stale   bool   // True if an existing lock was detected as stale
}

func (e *LockError) Error() string {
	if e.Stale {
		return fmt.Sprintf("stale lock on %s (held >60s), breaking and retrying", e.Path)
	}
	return fmt.Sprintf("could not acquire lock on %s within %s: another nd process may be running", e.Path, e.Timeout)
}

// ConflictError is returned when a deploy target already has a file/symlink (FR-016b).
type ConflictError struct {
	TargetPath   string           // Where we want to deploy
	ExistingKind OriginalFileKind // What's already there
	AssetName    string           // What we're trying to deploy
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict at %s: existing %s blocks deployment of %s", e.TargetPath, e.ExistingKind, e.AssetName)
}
