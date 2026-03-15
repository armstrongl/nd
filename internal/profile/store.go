package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/state"
)

// Compile-time check that Store implements deploy.SnapshotSaver.
var _ deploy.SnapshotSaver = (*Store)(nil)

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

// --- Snapshot methods ---

// userSnapshotDir returns the directory for user-created snapshots.
func (s *Store) userSnapshotDir() string {
	return filepath.Join(s.snapshotsDir, "user")
}

// autoSnapshotDir returns the directory for auto-created snapshots.
func (s *Store) autoSnapshotDir() string {
	return filepath.Join(s.snapshotsDir, "auto")
}

// snapshotPath returns the filesystem path for a snapshot.
func (s *Store) snapshotPath(name string, auto bool) string {
	if auto {
		return filepath.Join(s.autoSnapshotDir(), name+".yaml")
	}
	return filepath.Join(s.userSnapshotDir(), name+".yaml")
}

// SaveSnapshot validates and persists a snapshot.
// Returns an error if a snapshot with the same name already exists.
func (s *Store) SaveSnapshot(snap Snapshot) error {
	if err := ValidateName(snap.Name); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	if errs := snap.Validate(); len(errs) > 0 {
		return fmt.Errorf("save snapshot %q: %w", snap.Name, errs[0])
	}

	dir := s.userSnapshotDir()
	if snap.Auto {
		dir = s.autoSnapshotDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create snapshots directory: %w", err)
	}

	path := s.snapshotPath(snap.Name, snap.Auto)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("snapshot %q already exists", snap.Name)
	}

	data, err := yaml.Marshal(&snap)
	if err != nil {
		return fmt.Errorf("marshal snapshot %q: %w", snap.Name, err)
	}
	return nd.AtomicWrite(path, data)
}

// GetSnapshot reads a snapshot from disk.
func (s *Store) GetSnapshot(name string, auto bool) (*Snapshot, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	data, err := os.ReadFile(s.snapshotPath(name, auto))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot %q not found", name)
		}
		return nil, fmt.Errorf("read snapshot %q: %w", name, err)
	}

	var snap Snapshot
	if err := yaml.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot %q: %w", name, err)
	}
	return &snap, nil
}

// listSnapshotsInDir reads snapshot summaries from a single directory.
func (s *Store) listSnapshotsInDir(dir string) ([]SnapshotSummary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var summaries []SnapshotSummary
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		isAuto := dir == s.autoSnapshotDir()
		snap, err := s.GetSnapshot(name, isAuto)
		if err != nil {
			continue
		}
		summaries = append(summaries, SnapshotSummary{
			Name:            snap.Name,
			Auto:            snap.Auto,
			DeploymentCount: len(snap.Deployments),
			CreatedAt:       snap.CreatedAt,
		})
	}
	return summaries, nil
}

// ListSnapshots returns summaries of all snapshots (user + auto).
func (s *Store) ListSnapshots() ([]SnapshotSummary, error) {
	user, err := s.listSnapshotsInDir(s.userSnapshotDir())
	if err != nil {
		return nil, fmt.Errorf("list user snapshots: %w", err)
	}
	auto, err := s.listSnapshotsInDir(s.autoSnapshotDir())
	if err != nil {
		return nil, fmt.Errorf("list auto snapshots: %w", err)
	}

	result := append(user, auto...)
	if result == nil {
		result = []SnapshotSummary{}
	}
	return result, nil
}

// AutoSnapshot creates a timestamped auto-snapshot from current deployments.
// Auto-snapshot names use format "auto-YYYYMMDDTHHmmss-NNNNNNNNN" with nanosecond
// precision to avoid collisions when multiple auto-snapshots are created within
// the same second.
func (s *Store) AutoSnapshot(deployments []SnapshotEntry) (*Snapshot, error) {
	now := time.Now()
	name := "auto-" + now.Format("20060102T150405") + fmt.Sprintf("-%09d", now.Nanosecond())

	if deployments == nil {
		deployments = []SnapshotEntry{}
	}

	snap := Snapshot{
		Version:     nd.SchemaVersion,
		Name:        name,
		CreatedAt:   now.Truncate(time.Second),
		Auto:        true,
		Deployments: deployments,
	}

	dir := s.autoSnapshotDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create auto snapshot directory: %w", err)
	}

	path := s.snapshotPath(name, true)
	data, err := yaml.Marshal(&snap)
	if err != nil {
		return nil, fmt.Errorf("marshal auto snapshot: %w", err)
	}

	if err := nd.AtomicWrite(path, data); err != nil {
		return nil, fmt.Errorf("write auto snapshot: %w", err)
	}

	return &snap, nil
}

// PruneAutoSnapshots removes the oldest auto-snapshots beyond the keep limit.
// Auto-snapshot filenames sort chronologically because they use timestamps.
func (s *Store) PruneAutoSnapshots(keep int) error {
	dir := s.autoSnapshotDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read auto snapshots directory: %w", err)
	}

	var yamlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			yamlFiles = append(yamlFiles, entry.Name())
		}
	}

	if len(yamlFiles) <= keep {
		return nil
	}

	// ReadDir returns entries sorted by name. Timestamps sort chronologically.
	// Remove oldest (first entries) keeping the newest `keep` entries.
	toRemove := yamlFiles[:len(yamlFiles)-keep]
	for _, name := range toRemove {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove auto snapshot %q: %w", name, err)
		}
	}

	return nil
}

// DeleteSnapshot removes a snapshot from disk.
func (s *Store) DeleteSnapshot(name string, auto bool) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}

	path := s.snapshotPath(name, auto)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %q not found", name)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete snapshot %q: %w", name, err)
	}
	return nil
}

// AutoSave converts deployments to snapshot entries, creates an auto-snapshot,
// and prunes old auto-snapshots to keep the last 5.
// Implements deploy.SnapshotSaver interface.
func (s *Store) AutoSave(deployments []state.Deployment) error {
	entries := make([]SnapshotEntry, len(deployments))
	for i, d := range deployments {
		entries[i] = SnapshotEntry{
			SourceID:    d.SourceID,
			AssetType:   d.AssetType,
			AssetName:   d.AssetName,
			SourcePath:  d.SourcePath,
			LinkPath:    d.LinkPath,
			Scope:       d.Scope,
			ProjectPath: d.ProjectPath,
			Origin:      d.Origin,
			DeployedAt:  d.DeployedAt,
		}
	}

	if _, err := s.AutoSnapshot(entries); err != nil {
		return err
	}
	return s.PruneAutoSnapshots(5)
}
