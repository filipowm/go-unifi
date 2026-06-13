package shared //nolint:testpackage // tests the slog adapter's level/format behavior directly

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ansiEscape is the CSI introducer; its absence proves no colour codes are emitted.
const ansiEscape = "\x1b["

// newBufferLogger builds a slog-backed Logger writing to buf at levelTrace so
// every level (including Trace, which is below Debug) is captured.
func newBufferLogger(buf *bytes.Buffer) Logger {
	return NewSlogLogger(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: LevelTrace})))
}

func TestSlogLoggerLevelsAndFormatting(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := newBufferLogger(&buf)

	l.Tracef("t=%d", 1)
	l.Debugf("d=%s", "x")
	l.Infof("i=%d", 2)
	l.Warnf("w=%d", 3)
	l.Errorf("e=%d", 4)
	l.Debugln("dbg", "line")
	l.Infoln("Structure JSONs ready!")
	l.Error(errors.New("boom"))

	out := buf.String()
	assert.NotContains(t, out, ansiEscape, "text handler to a buffer must not emit ANSI colour codes")

	// Each line carries the formatted message at the expected level.
	wants := map[string]string{
		"t=1":                    "level=DEBUG-4", // Trace maps to Debug-4
		"d=x":                    "level=DEBUG",
		"i=2":                    "level=INFO",
		"w=3":                    "level=WARN",
		"e=4":                    "level=ERROR",
		"boom":                   "level=ERROR", // Error(args...) uses fmt.Sprint
		"Structure JSONs ready!": "level=INFO",
	}
	for msg, level := range wants {
		line := lineContaining(t, out, msg)
		assert.Containsf(t, line, level, "message %q must be logged at %s", msg, level)
	}

	// Println-style methods join args with spaces and drop the trailing newline.
	dbgLine := lineContaining(t, out, "dbg")
	assert.Contains(t, dbgLine, `msg="dbg line"`, "Debugln must space-join args")
	assert.NotContains(t, dbgLine, `\n`, "Debugln must not embed a trailing newline in the message")
}

func TestSlogLoggerHonorsMinLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := NewTextLogger(&buf, slog.LevelWarn)

	l.Tracef("trace-line")
	l.Debugf("debug-line")
	l.Infof("info-line")
	l.Warnf("warn-line")
	l.Errorf("error-line")

	out := buf.String()
	assert.NotContains(t, out, "trace-line")
	assert.NotContains(t, out, "debug-line")
	assert.NotContains(t, out, "info-line")
	assert.Contains(t, out, "warn-line")
	assert.Contains(t, out, "error-line")
}

func TestLevelTraceIsDebugMinus4(t *testing.T) {
	t.Parallel()
	assert.Equal(t, LevelTrace, slog.Level(-8), "Trace must map to Debug-4 (-8)")
}

// lineContaining returns the single output line containing substr, failing if
// none is found.
func lineContaining(t *testing.T, out, substr string) string {
	t.Helper()
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	require.Failf(t, "no log line found", "expected a line containing %q in:\n%s", substr, out)
	return ""
}
