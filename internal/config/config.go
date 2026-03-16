package config

import "github.com/armstrongl/nd/internal/nd"

// Config represents the merged, resolved configuration.
// This is what the rest of the application uses after loading + merging.
// Merge order: built-in defaults -> global config -> project config -> CLI flags.
type Config struct {
	Version         int                `yaml:"version"          json:"version"`
	DefaultScope    nd.Scope           `yaml:"default_scope"    json:"default_scope"`
	DefaultAgent    string             `yaml:"default_agent"    json:"default_agent"`
	SymlinkStrategy nd.SymlinkStrategy `yaml:"symlink_strategy" json:"symlink_strategy"`
	Sources         []SourceEntry      `yaml:"sources"          json:"sources"`
	Agents          []AgentOverride    `yaml:"agents,omitempty" json:"agents,omitempty"`
	ContextTypes    []string           `yaml:"context_types,omitempty" json:"context_types,omitempty"`
}

// SourceEntry represents a source registration in the config file.
// Sources are listed in registration order (first registered = highest priority per FR-016a).
type SourceEntry struct {
	ID    string        `yaml:"id"              json:"id"`
	Type  nd.SourceType `yaml:"type"            json:"type"`
	Path  string        `yaml:"path"            json:"path"`
	URL   string        `yaml:"url,omitempty"   json:"url,omitempty"`
	Alias string        `yaml:"alias,omitempty" json:"alias,omitempty"`
}

// AgentOverride lets users customize agent config directory paths (FR-033).
type AgentOverride struct {
	Name       string `yaml:"name"        json:"name"`
	GlobalDir  string `yaml:"global_dir"  json:"global_dir"`
	ProjectDir string `yaml:"project_dir" json:"project_dir"`
}

// ProjectConfig represents .nd/config.yaml (project-level overrides).
// Fields are pointers so we can distinguish "not set" from "set to zero value"
// during the merge with global config.
type ProjectConfig struct {
	Version         int                 `yaml:"version"`
	DefaultScope    *nd.Scope           `yaml:"default_scope,omitempty"`
	DefaultAgent    *string             `yaml:"default_agent,omitempty"`
	SymlinkStrategy *nd.SymlinkStrategy `yaml:"symlink_strategy,omitempty"`
	Sources         []SourceEntry       `yaml:"sources,omitempty"`
	Agents          []AgentOverride     `yaml:"agents,omitempty"`
}
