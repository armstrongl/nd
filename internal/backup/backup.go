package backup

import (
	"time"

	"github.com/armstrongl/nd/internal/nd"
)

// Backup represents a backed-up file.
// Naming convention: <filename>.<ISO-8601-timestamp>.bak
// Stored in: ~/.config/nd/backups/
// Retention: last 5 per target location (grouped by OriginalPath).
type Backup struct {
	OriginalPath string              `json:"original_path"`
	BackupPath   string              `json:"backup_path"`
	CreatedAt    time.Time           `json:"created_at"`
	OriginalKind nd.OriginalFileKind `json:"original_kind"`
}
