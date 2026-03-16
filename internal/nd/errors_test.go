package nd_test

import (
	"errors"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestPathTraversalError(t *testing.T) {
	err := &nd.PathTraversalError{
		Path:     "../../../etc/passwd",
		Root:     "/Users/dev/source",
		SourceID: "my-source",
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
	var pte *nd.PathTraversalError
	if !errors.As(err, &pte) {
		t.Error("should be assertable as PathTraversalError")
	}
}

func TestLockError(t *testing.T) {
	err := &nd.LockError{Path: "/path/to/lock", Timeout: "5s", Stale: false}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}

	stale := &nd.LockError{Path: "/path/to/lock", Timeout: "5s", Stale: true}
	if stale.Error() == err.Error() {
		t.Error("stale and non-stale messages should differ")
	}
}

func TestConflictError(t *testing.T) {
	err := &nd.ConflictError{
		TargetPath:   "/Users/dev/.claude/CLAUDE.md",
		ExistingKind: nd.FileKindPlainFile,
		AssetName:    "go-project-rules",
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}
