package oplog_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/oplog"
)

func TestLogEntryJSONRoundTrip(t *testing.T) {
	entry := oplog.LogEntry{
		Timestamp: time.Now().Truncate(time.Second),
		Operation: oplog.OpDeploy,
		Assets:    []asset.Identity{{SourceID: "s", Type: nd.AssetSkill, Name: "x"}},
		Scope:     nd.ScopeGlobal,
		Succeeded: 1,
		Failed:    0,
	}
	data, err := json.Marshal(&entry)
	if err != nil {
		t.Fatal(err)
	}
	var got oplog.LogEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Operation != oplog.OpDeploy {
		t.Errorf("operation: got %q", got.Operation)
	}
}
