package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/nd"
)

// ProfileSummary is a lightweight view of a profile for listing.
type ProfileSummary struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	AssetCount  int       `json:"asset_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SnapshotSummary is a lightweight view of a snapshot for listing.
type SnapshotSummary struct {
	Name            string    `json:"name"`
	Auto            bool      `json:"auto"`
	DeploymentCount int       `json:"deployment_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// Store manages profile and snapshot persistence on disk.
type Store struct {
	profilesDir  string
	snapshotsDir string
}

// NewStore creates a Store targeting the given directories.
func NewStore(profilesDir, snapshotsDir string) *Store {
	return &Store{
		profilesDir:  profilesDir,
		snapshotsDir: snapshotsDir,
	}
}

// profilePath returns the filesystem path for a profile by name.
func (s *Store) profilePath(name string) string {
	return filepath.Join(s.profilesDir, name+".yaml")
}

// CreateProfile validates and persists a new profile.
// Returns an error if a profile with the same name already exists.
func (s *Store) CreateProfile(p Profile) error {
	if err := ValidateName(p.Name); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	if errs := p.Validate(); len(errs) > 0 {
		return fmt.Errorf("create profile %q: %w", p.Name, errs[0])
	}

	path := s.profilePath(p.Name)

	if err := os.MkdirAll(s.profilesDir, 0o755); err != nil {
		return fmt.Errorf("create profiles directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("profile %q already exists", p.Name)
	}

	data, err := yaml.Marshal(&p)
	if err != nil {
		return fmt.Errorf("marshal profile %q: %w", p.Name, err)
	}
	return nd.AtomicWrite(path, data)
}

// GetProfile reads a profile from disk by name.
func (s *Store) GetProfile(name string) (*Profile, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	data, err := os.ReadFile(s.profilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		return nil, fmt.Errorf("read profile %q: %w", name, err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %q: %w", name, err)
	}
	return &p, nil
}

// ListProfiles returns summaries of all profiles on disk.
// Returns empty slice (not nil) if no profiles exist or directory is missing.
func (s *Store) ListProfiles() ([]ProfileSummary, error) {
	entries, err := os.ReadDir(s.profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileSummary{}, nil
		}
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	var summaries []ProfileSummary
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		p, err := s.GetProfile(name)
		if err != nil {
			continue // skip corrupt files
		}
		summaries = append(summaries, ProfileSummary{
			Name:        p.Name,
			Description: p.Description,
			AssetCount:  len(p.Assets),
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	if summaries == nil {
		summaries = []ProfileSummary{}
	}
	return summaries, nil
}

// DeleteProfile removes a profile from disk.
// Returns an error if the profile does not exist.
func (s *Store) DeleteProfile(name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}

	path := s.profilePath(name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", name)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete profile %q: %w", name, err)
	}
	return nil
}

// UpdateProfile overwrites an existing profile on disk.
// Returns an error if the profile does not exist.
func (s *Store) UpdateProfile(p Profile) error {
	if err := ValidateName(p.Name); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	if errs := p.Validate(); len(errs) > 0 {
		return fmt.Errorf("update profile %q: %w", p.Name, errs[0])
	}

	path := s.profilePath(p.Name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", p.Name)
	}

	data, err := yaml.Marshal(&p)
	if err != nil {
		return fmt.Errorf("marshal profile %q: %w", p.Name, err)
	}
	return nd.AtomicWrite(path, data)
}
