package profile

import (
	"fmt"
	"regexp"
)

// namePattern matches valid profile and snapshot names: alphanumeric, hyphens, underscores.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ValidateName checks that a profile or snapshot name is safe for use as a filename.
func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid name %q: must match [a-zA-Z0-9][a-zA-Z0-9_-]*", name)
	}
	return nil
}
