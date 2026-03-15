package version

import (
	"strings"
	"testing"
)

func TestString_Defaults(t *testing.T) {
	s := String()
	if !strings.Contains(s, "dev") {
		t.Errorf("expected default version to contain 'dev', got %q", s)
	}
	if !strings.Contains(s, "none") {
		t.Errorf("expected default commit to contain 'none', got %q", s)
	}
	if !strings.Contains(s, "unknown") {
		t.Errorf("expected default date to contain 'unknown', got %q", s)
	}
}

func TestString_Custom(t *testing.T) {
	origV, origC, origD := Version, Commit, Date
	defer func() { Version, Commit, Date = origV, origC, origD }()

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-03-15"

	s := String()
	want := "nd version v1.2.3 (commit: abc1234, built: 2026-03-15)"
	if s != want {
		t.Errorf("got %q, want %q", s, want)
	}
}
