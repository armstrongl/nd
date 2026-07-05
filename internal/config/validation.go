package config

import (
	"fmt"
	"strings"

	"github.com/armstrongl/nd/internal/nd"
)

// knownDeployAgentNames lists the built-in agent names accepted for
// default_deploy_agents. Kept local to avoid an import cycle with the agent
// package (which imports config); it must stay in sync with the agents defined
// in agent.New (internal/agent/registry.go).
var knownDeployAgentNames = []string{"claude-code", "copilot"}

// isKnownDeployAgent reports whether name is a recognized built-in agent.
func isKnownDeployAgent(name string) bool {
	for _, n := range knownDeployAgentNames {
		if n == name {
			return true
		}
	}
	return false
}

// ValidationError represents a single config validation failure.
type ValidationError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s:%d: field %s: %s", e.File, e.Line, e.Field, e.Message)
	}
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

// Validate checks all fields for correctness.
// Returns a slice of ValidationError with file and line info (NFR-005).
func (c *Config) Validate() []ValidationError {
	var errs []ValidationError

	if c.Version < 1 {
		errs = append(errs, ValidationError{
			Field: "version", Message: "must be >= 1",
		})
	}

	if c.Version > nd.SchemaVersion {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: fmt.Sprintf("config version %d is newer than supported version %d (downgrade?)", c.Version, nd.SchemaVersion),
		})
	}

	switch c.DefaultScope {
	case nd.ScopeGlobal, nd.ScopeProject:
		// valid
	default:
		errs = append(errs, ValidationError{
			Field:   "default_scope",
			Message: fmt.Sprintf("invalid scope %q, must be %q or %q", c.DefaultScope, nd.ScopeGlobal, nd.ScopeProject),
		})
	}

	if c.DefaultAgent == "" {
		errs = append(errs, ValidationError{
			Field: "default_agent", Message: "must not be empty",
		})
	}

	for _, name := range c.DefaultDeployAgents {
		if !isKnownDeployAgent(name) {
			errs = append(errs, ValidationError{
				Field:   "default_deploy_agents",
				Message: fmt.Sprintf("unknown agent %q, must be one of %s", name, strings.Join(knownDeployAgentNames, ", ")),
			})
		}
	}

	switch c.SymlinkStrategy {
	case nd.SymlinkAbsolute, nd.SymlinkRelative:
		// valid
	default:
		errs = append(errs, ValidationError{
			Field:   "symlink_strategy",
			Message: fmt.Sprintf("invalid strategy %q, must be %q or %q", c.SymlinkStrategy, nd.SymlinkAbsolute, nd.SymlinkRelative),
		})
	}

	seenIDs := make(map[string]bool)
	for i, s := range c.Sources {
		field := fmt.Sprintf("sources[%d]", i)
		if s.ID == "" {
			errs = append(errs, ValidationError{
				Field: field + ".id", Message: "must not be empty",
			})
		} else if seenIDs[s.ID] {
			errs = append(errs, ValidationError{
				Field: field + ".id", Message: fmt.Sprintf("duplicate source ID %q", s.ID),
			})
		} else {
			seenIDs[s.ID] = true
		}

		if s.Path == "" {
			errs = append(errs, ValidationError{
				Field: field + ".path", Message: "must not be empty",
			})
		}

		switch s.Type {
		case nd.SourceLocal, nd.SourceGit, nd.SourceBuiltin:
			// valid
		default:
			errs = append(errs, ValidationError{
				Field:   field + ".type",
				Message: fmt.Sprintf("invalid source type %q", s.Type),
			})
		}
	}

	return errs
}
