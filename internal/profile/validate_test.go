package profile_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/profile"
)

func TestValidateNameAcceptsValid(t *testing.T) {
	valid := []string{"go-backend", "my_profile", "test123", "A", "a-b_c-1"}
	for _, name := range valid {
		if err := profile.ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateNameRejectsInvalid(t *testing.T) {
	invalid := []string{
		"",              // empty
		"has spaces",    // spaces
		"path/traversal", // slashes
		"../escape",     // dot-dot
		".hidden",       // leading dot
		"special!char",  // special char
		"a@b",           // at sign
	}
	for _, name := range invalid {
		if err := profile.ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", name)
		}
	}
}
