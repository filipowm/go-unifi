package internal //nolint:testpackage // tests access unexported symbols

import (
	"context"
	"log/slog"
	"sync"

	"github.com/filipowm/go-unifi/v2/codegen/shared"
)

// capturedRecord is one log entry recorded by captureHandler, reduced to the
// level and message the tests assert on.
type capturedRecord struct {
	Level   slog.Level
	Message string
}

// captureHandler is a slog.Handler that records every record into a shared,
// mutex-guarded slice, letting tests inject an isolated logger and assert its
// output in parallel without touching shared state.
type captureHandler struct {
	mu      *sync.Mutex
	records *[]capturedRecord
}

func (h captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = append(*h.records, capturedRecord{Level: r.Level, Message: r.Message})
	return nil
}

func (h captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h captureHandler) WithGroup(string) slog.Handler      { return h }

// newCaptureLogger returns a shared.Logger that records every entry and an
// accessor returning a snapshot of the captured records. It captures at every
// level (Enabled is always true), so callers need not set a minimum level.
func newCaptureLogger() (shared.Logger, func() []capturedRecord) {
	mu := &sync.Mutex{}
	records := &[]capturedRecord{}
	logger := shared.NewSlogLogger(slog.New(captureHandler{mu: mu, records: records}))
	snapshot := func() []capturedRecord {
		mu.Lock()
		defer mu.Unlock()
		out := make([]capturedRecord, len(*records))
		copy(out, *records)
		return out
	}
	return logger, snapshot
}

// warnMessages returns the messages of all WARN-level captured records.
func warnMessages(records []capturedRecord) []string {
	var warns []string
	for _, r := range records {
		if r.Level == slog.LevelWarn {
			warns = append(warns, r.Message)
		}
	}
	return warns
}
