package nd_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestBuiltinContextFileNames(t *testing.T) {
	names := nd.BuiltinContextFileNames()
	if len(names) != 4 {
		t.Fatalf("expected 4 built-in context file names, got %d", len(names))
	}
}

func TestIsLocalOnlyContext(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"CLAUDE.md", false},
		{"AGENTS.md", false},
		{"CLAUDE.local.md", true},
		{"AGENTS.local.md", true},
		{"CUSTOM.local.md", true},
		{"short.md", false},
	}
	for _, tt := range tests {
		if got := nd.IsLocalOnlyContext(tt.name); got != tt.want {
			t.Errorf("IsLocalOnlyContext(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
