package sourcemanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larah/nd/internal/config"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/source"
)

// GenerateSourceID creates a source ID from a path's base name,
// deduplicating with a numeric suffix if needed.
func GenerateSourceID(path string, existingIDs map[string]bool) string {
	base := filepath.Base(path)
	if existingIDs == nil || !existingIDs[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !existingIDs[candidate] {
			return candidate
		}
	}
}

// AddLocal registers a local directory as an asset source.
func (sm *SourceManager) AddLocal(path string, alias string) (*source.Source, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", absPath)
	}

	// Check for duplicate
	for _, s := range sm.cfg.Sources {
		if s.Path == absPath {
			return nil, fmt.Errorf("source at %q is already registered as %q", absPath, s.ID)
		}
	}

	existingIDs := make(map[string]bool)
	for _, s := range sm.cfg.Sources {
		existingIDs[s.ID] = true
	}
	id := GenerateSourceID(absPath, existingIDs)

	entry := config.SourceEntry{
		ID:    id,
		Type:  nd.SourceLocal,
		Path:  absPath,
		Alias: alias,
	}

	sm.cfg.Sources = append(sm.cfg.Sources, entry)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		// Roll back in-memory change
		sm.cfg.Sources = sm.cfg.Sources[:len(sm.cfg.Sources)-1]
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &source.Source{
		ID:    id,
		Type:  nd.SourceLocal,
		Path:  absPath,
		Alias: alias,
		Order: len(sm.cfg.Sources) - 1,
	}, nil
}
