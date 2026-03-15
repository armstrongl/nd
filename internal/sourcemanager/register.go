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

// Remove unregisters a source by ID. Does not delete deployed assets
// or cloned directories — that is the caller's responsibility.
func (sm *SourceManager) Remove(sourceID string) error {
	idx := -1
	for i, s := range sm.cfg.Sources {
		if s.ID == sourceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("source %q not found", sourceID)
	}

	removed := sm.cfg.Sources[idx]
	sm.cfg.Sources = append(sm.cfg.Sources[:idx], sm.cfg.Sources[idx+1:]...)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		// Roll back
		sm.cfg.Sources = append(sm.cfg.Sources[:idx], append([]config.SourceEntry{removed}, sm.cfg.Sources[idx:]...)...)
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

// AddGit registers a Git repository as an asset source by cloning it.
// Clone target is derived from sm.sourcesDir (e.g., ~/.config/nd/sources/).
func (sm *SourceManager) AddGit(url string, alias string) (*source.Source, error) {
	expandedURL := ExpandGitURL(url)

	// Check for duplicate URL (persisted in config via SourceEntry.URL)
	for _, s := range sm.cfg.Sources {
		if s.Type == nd.SourceGit && s.URL == expandedURL {
			return nil, fmt.Errorf("git source %q is already registered as %q", url, s.ID)
		}
	}

	existingIDs := make(map[string]bool)
	for _, s := range sm.cfg.Sources {
		existingIDs[s.ID] = true
	}
	repoName := RepoNameFromURL(url)
	id := GenerateSourceID(filepath.Join(sm.sourcesDir, repoName), existingIDs)

	cloneTarget := filepath.Join(sm.sourcesDir, id)

	if err := os.MkdirAll(sm.sourcesDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sources dir: %w", err)
	}

	if err := gitClone(expandedURL, cloneTarget); err != nil {
		os.RemoveAll(cloneTarget)
		return nil, err
	}

	entry := config.SourceEntry{
		ID:    id,
		Type:  nd.SourceGit,
		Path:  cloneTarget,
		URL:   expandedURL,
		Alias: alias,
	}

	sm.cfg.Sources = append(sm.cfg.Sources, entry)

	if err := WriteConfig(sm.configPath, sm.cfg); err != nil {
		sm.cfg.Sources = sm.cfg.Sources[:len(sm.cfg.Sources)-1]
		os.RemoveAll(cloneTarget)
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &source.Source{
		ID:    id,
		Type:  nd.SourceGit,
		Path:  cloneTarget,
		URL:   expandedURL,
		Alias: alias,
		Order: len(sm.cfg.Sources) - 1,
	}, nil
}
