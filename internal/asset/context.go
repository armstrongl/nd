package asset

// ContextInfo holds context-specific details for context assets.
// FileName is a plain string (not an enum) to support custom context file
// types registered via config.
type ContextInfo struct {
	FolderName string
	FileName   string
}

// ContextMeta represents the _meta.yaml file inside a context folder.
type ContextMeta struct {
	Description    string   `yaml:"description"                json:"description"`
	Tags           []string `yaml:"tags,omitempty"              json:"tags,omitempty"`
	TargetLanguage string   `yaml:"target_language,omitempty"   json:"target_language,omitempty"`
	TargetProject  string   `yaml:"target_project,omitempty"    json:"target_project,omitempty"`
	TargetAgent    string   `yaml:"target_agent,omitempty"      json:"target_agent,omitempty"`
}

// Validate checks ContextMeta fields for correctness.
func (m *ContextMeta) Validate() error {
	return nil
}
