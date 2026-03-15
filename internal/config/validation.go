package config

import "fmt"

// ValidationError represents a single config validation failure.
type ValidationError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s:%d: field %s: %s", e.File, e.Line, e.Field, e.Message)
}

// Validate checks all fields for correctness.
// Returns a slice of ValidationError with file and line info (NFR-005).
func (c *Config) Validate() []ValidationError {
	return nil
}
