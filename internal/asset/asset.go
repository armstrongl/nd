package asset

// Asset represents a discovered asset from a registered source.
type Asset struct {
	Identity
	SourcePath  string       `yaml:"-" json:"source_path"`
	IsDir       bool         `yaml:"-" json:"is_dir"`
	GroupDir    string       `yaml:"-" json:"group_dir,omitempty"`
	ContextFile *ContextInfo `yaml:"-" json:"context_file,omitempty"`
	Meta        *ContextMeta `yaml:"-" json:"meta,omitempty"`
}
