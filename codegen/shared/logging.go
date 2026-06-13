package shared

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// LevelTrace is the slog level used for Trace logging. slog has no native Trace
// level, so we use Debug-4 — the community convention for a level below Debug.
const LevelTrace = slog.LevelDebug - 4

// NewSlogLogger wraps a caller-supplied *slog.Logger into the pipeline Logger,
// adapting slog's structured API to the printf/println-style methods the
// generation pipeline depends on.
func NewSlogLogger(l *slog.Logger) Logger {
	return &slogLogger{logger: l}
}

// NewTextLogger returns a Logger backed by slog's text handler writing plain
// text (no ANSI colours) to out at the given minimum level.
func NewTextLogger(out io.Writer, level slog.Level) Logger {
	handler := slog.NewTextHandler(out, &slog.HandlerOptions{Level: level})
	return NewSlogLogger(slog.New(handler))
}

// DefaultLogger returns the fallback Logger used when a pipeline component is
// constructed without an explicit one: slog text output to stderr at Info.
func DefaultLogger() Logger {
	return NewTextLogger(os.Stderr, slog.LevelInfo)
}

type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Tracef(format string, args ...any) { l.logf(LevelTrace, format, args...) }
func (l *slogLogger) Debugf(format string, args ...any) { l.logf(slog.LevelDebug, format, args...) }
func (l *slogLogger) Infof(format string, args ...any)  { l.logf(slog.LevelInfo, format, args...) }
func (l *slogLogger) Warnf(format string, args ...any)  { l.logf(slog.LevelWarn, format, args...) }
func (l *slogLogger) Errorf(format string, args ...any) { l.logf(slog.LevelError, format, args...) }

func (l *slogLogger) Debugln(args ...any) { l.logln(slog.LevelDebug, args...) }
func (l *slogLogger) Infoln(args ...any)  { l.logln(slog.LevelInfo, args...) }

func (l *slogLogger) Error(args ...any) {
	l.logger.Log(context.Background(), slog.LevelError, fmt.Sprint(args...))
}

func (l *slogLogger) logf(level slog.Level, format string, args ...any) {
	l.logger.Log(context.Background(), level, fmt.Sprintf(format, args...))
}

func (l *slogLogger) logln(level slog.Level, args ...any) {
	l.logger.Log(context.Background(), level, strings.TrimSuffix(fmt.Sprintln(args...), "\n"))
}
