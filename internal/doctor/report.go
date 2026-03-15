package doctor

import (
	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/state"
)

// Report is the aggregate output of nd doctor.
// Each section corresponds to a check category.
type Report struct {
	Config      ConfigCheck        `json:"config"`
	Sources     []SourceCheck      `json:"sources"`
	Deployments []state.HealthCheck `json:"deployments"`
	Agents      []AgentCheck       `json:"agents"`
	Git         GitCheck           `json:"git"`
	Summary     Summary            `json:"summary"`
}

// ConfigCheck reports config file validation results.
type ConfigCheck struct {
	GlobalValid  bool                     `json:"global_valid"`
	ProjectValid bool                     `json:"project_valid"`
	Errors       []config.ValidationError `json:"errors,omitempty"`
}

// SourceCheck reports the accessibility and health of one source.
type SourceCheck struct {
	SourceID   string `json:"source_id"`
	Available  bool   `json:"available"`
	AssetCount int    `json:"asset_count"`
	Detail     string `json:"detail,omitempty"`
}

// AgentCheck reports whether an agent's directories exist and are writable.
type AgentCheck struct {
	AgentName  string `json:"agent_name"`
	Detected   bool   `json:"detected"`
	GlobalDir  string `json:"global_dir"`
	GlobalOK   bool   `json:"global_ok"`
	ProjectDir string `json:"project_dir,omitempty"`
	ProjectOK  bool   `json:"project_ok,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// GitCheck reports whether Git is available (needed for Git-sourced repos).
type GitCheck struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// Summary provides pass/warn/fail counts across all checks.
type Summary struct {
	Pass int `json:"pass"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
}
