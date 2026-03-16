package backup_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/backup"
	"github.com/armstrongl/nd/internal/nd"
)

func TestBackupJSONRoundTrip(t *testing.T) {
	b := backup.Backup{
		OriginalPath: "/Users/dev/.claude/CLAUDE.md",
		BackupPath:   "/Users/dev/.config/nd/backups/CLAUDE.md.2026-03-14T10-30-00.bak",
		CreatedAt:    time.Now().Truncate(time.Second),
		OriginalKind: nd.FileKindPlainFile,
	}
	data, err := json.Marshal(&b)
	if err != nil {
		t.Fatal(err)
	}
	var got backup.Backup
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.OriginalKind != nd.FileKindPlainFile {
		t.Errorf("kind: got %q", got.OriginalKind)
	}
}
