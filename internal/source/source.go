package source

import "github.com/armstrongl/nd/internal/nd"

// Source represents a registered asset source (local directory or Git repo).
type Source struct {
	ID        string        `yaml:"id"              json:"id"`
	Type      nd.SourceType `yaml:"type"            json:"type"`
	Path      string        `yaml:"path"            json:"path"`
	URL       string        `yaml:"url,omitempty"   json:"url,omitempty"`
	Alias     string        `yaml:"alias,omitempty" json:"alias,omitempty"`
	Order     int           `yaml:"-"               json:"order"`
	Manifest  *Manifest     `yaml:"-"               json:"-"`
	Available bool          `yaml:"-"               json:"available"`
}
