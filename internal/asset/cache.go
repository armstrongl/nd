package asset

import "time"

// CachedIndex is the on-disk representation of the asset discovery cache.
// Stored at ~/.cache/nd/index/<source_id>.yaml.
type CachedIndex struct {
	Version   int       `yaml:"version"`
	SourceID  string    `yaml:"source_id"`
	BuiltAt   time.Time `yaml:"built_at"`
	SourceMod time.Time `yaml:"source_mod"`
	Assets    []Asset   `yaml:"assets"`
}

// IsStale returns true if the cache is older than the source's last modification.
func (c *CachedIndex) IsStale(currentSourceMod time.Time) bool {
	return currentSourceMod.After(c.SourceMod)
}
