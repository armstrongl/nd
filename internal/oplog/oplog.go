package oplog

import (
	"time"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/nd"
)

// LogEntry records a single nd operation for the operation log.
type LogEntry struct {
	Timestamp time.Time        `json:"timestamp"`
	Operation OperationType    `json:"operation"`
	Assets    []asset.Identity `json:"assets,omitempty"`
	Scope     nd.Scope         `json:"scope,omitempty"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Detail    string           `json:"detail,omitempty"`
}

// OperationType categorizes log entries.
type OperationType string

const (
	OpDeploy          OperationType = "deploy"
	OpRemove          OperationType = "remove"
	OpSync            OperationType = "sync"
	OpProfileSwitch   OperationType = "profile-switch"
	OpSnapshotSave    OperationType = "snapshot-save"
	OpSnapshotRestore OperationType = "snapshot-restore"
	OpSourceAdd       OperationType = "source-add"
	OpSourceRemove    OperationType = "source-remove"
	OpSourceSync      OperationType = "source-sync"
	OpUninstall       OperationType = "uninstall"
)
