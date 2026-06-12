package unifi //nolint: testpackage

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ansiEscape is the CSI introducer; its absence proves no colour codes are emitted.
const ansiEscape = "\x1b["

func TestNewDefaultLoggerNoANSI(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := newSlogLoggerToWriter(t, &buf, InfoLevel)

	l.Info("hello world")
	l.Errorf("boom: %d", 42)

	out := buf.String()
	require.NotEmpty(t, out)
	assert.NotContains(t, out, ansiEscape, "default logger must not emit ANSI escape codes to a non-TTY writer")
	assert.Contains(t, out, "hello world")
	assert.Contains(t, out, "boom: 42")
}

func TestNewDefaultLoggerDisabledIsNoop(t *testing.T) {
	t.Parallel()
	l := NewDefaultLogger(DisabledLevel)
	_, ok := l.(*noopLogger)
	assert.True(t, ok, "DisabledLevel must return the noop logger")
}

func TestSlogLevelMapping(t *testing.T) {
	t.Parallel()
	cases := map[LoggingLevel]slog.Level{
		TraceLevel: slog.LevelDebug - 4, // -8
		DebugLevel: slog.LevelDebug,
		InfoLevel:  slog.LevelInfo,
		WarnLevel:  slog.LevelWarn,
		ErrorLevel: slog.LevelError,
		// unknown values fall back to Info
		LoggingLevel(99): slog.LevelInfo,
	}
	for level, want := range cases {
		assert.Equal(t, want, slogLevel(level), "level %d", level)
	}
	assert.Equal(t, levelTrace, slog.Level(-8), "Trace must map to Debug-4 (-8)")
}

// TestNewDefaultLoggerHonorsMinLevel verifies the configured minimum level is
// applied: a WARN-level logger drops Info/Debug/Trace but keeps Warn/Error.
func TestNewDefaultLoggerHonorsMinLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := newSlogLoggerToWriter(t, &buf, WarnLevel)

	l.Trace("trace-line")
	l.Debug("debug-line")
	l.Info("info-line")
	l.Warn("warn-line")
	l.Error("error-line")

	out := buf.String()
	assert.NotContains(t, out, "trace-line")
	assert.NotContains(t, out, "debug-line")
	assert.NotContains(t, out, "info-line")
	assert.Contains(t, out, "warn-line")
	assert.Contains(t, out, "error-line")
}

func TestNewSlogLoggerWrapsSlog(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// Trace emits below Debug, so set the handler floor to levelTrace to capture it.
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: levelTrace})
	l := NewSlogLogger(slog.New(handler))

	l.Tracef("t=%d", 1)
	l.Infof("i=%s", "x")

	out := buf.String()
	assert.NotContains(t, out, ansiEscape)
	assert.Contains(t, out, "t=1")
	assert.Contains(t, out, "i=x")
	// Trace is emitted at the custom DEBUG-4 level.
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if strings.Contains(line, "t=1") {
			assert.Contains(t, line, "level=DEBUG-4")
		}
	}
}

// newSlogLoggerToWriter builds a slog-backed Logger writing to w at the given
// level, mirroring NewDefaultLogger but with an injectable writer for assertions.
func newSlogLoggerToWriter(t *testing.T, w *bytes.Buffer, level LoggingLevel) Logger {
	t.Helper()
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slogLevel(level)})
	return &slogLogger{logger: slog.New(handler)}
}
