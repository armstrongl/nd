package oplog

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	logFileName    = "operations.log"
	rotatedSuffix  = ".1"
	defaultMaxSize = 1 << 20 // 1 MB
)

// Writer appends operation log entries as JSONL to a log file.
// Rotates the log when it exceeds the configured max size.
type Writer struct {
	path    string // full path to operations.log
	maxSize int64
}

// Option configures Writer behavior.
type Option func(*Writer)

// WithMaxSize sets the maximum log file size before rotation.
func WithMaxSize(n int64) Option {
	return func(w *Writer) { w.maxSize = n }
}

// NewWriter creates a Writer that logs to logDir/operations.log.
func NewWriter(logDir string, opts ...Option) *Writer {
	w := &Writer{
		path:    filepath.Join(logDir, logFileName),
		maxSize: defaultMaxSize,
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Log appends a single entry to the operation log.
// Creates the log directory and file if they don't exist.
// Rotates the file if it exceeds the max size before writing.
func (w *Writer) Log(entry LogEntry) error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}

	if err := w.rotateIfNeeded(); err != nil {
		return err
	}

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = f.Write(data)
	return err
}

// rotateIfNeeded checks the current log file size and rotates if over max.
func (w *Writer) rotateIfNeeded() error {
	info, err := os.Stat(w.path)
	if err != nil {
		return nil // file doesn't exist yet — nothing to rotate
	}

	if info.Size() < w.maxSize {
		return nil
	}

	rotated := w.path + rotatedSuffix
	return os.Rename(w.path, rotated)
}
