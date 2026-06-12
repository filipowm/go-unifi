package unifi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// levelTrace is the slog level for Trace logging. slog has no native Trace
// level, so we use Debug-4 — the community convention for a level below Debug.
const levelTrace = slog.LevelDebug - 4

type Logger interface {
	Trace(format string)
	Debug(format string)
	Info(format string)
	Error(format string)
	Warn(format string)
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
	Warnf(format string, args ...any)
}

type LoggingLevel int

const (
	DisabledLevel LoggingLevel = iota
	TraceLevel
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
)

// slogLevel maps a LoggingLevel to its slog.Level equivalent. DisabledLevel is
// handled by the caller (returns a noop logger) and falls through to Info here.
func slogLevel(level LoggingLevel) slog.Level {
	switch level {
	case TraceLevel:
		return levelTrace
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	case DisabledLevel:
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// NewDefaultLogger returns a Logger backed by log/slog's text handler writing
// plain text (no ANSI colours) to os.Stderr. DisabledLevel yields a no-op logger.
func NewDefaultLogger(level LoggingLevel) Logger {
	if level == DisabledLevel {
		return &noopLogger{}
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel(level)})
	return &slogLogger{logger: slog.New(handler)}
}

// NewSlogLogger wraps a caller-supplied *slog.Logger into the Logger interface,
// reusing the same level mapping and *f formatting as the default logger.
func NewSlogLogger(l *slog.Logger) Logger {
	return &slogLogger{logger: l}
}

type noopLogger struct{}

func (l *noopLogger) Trace(msg string)                  {}
func (l *noopLogger) Debug(msg string)                  {}
func (l *noopLogger) Info(msg string)                   {}
func (l *noopLogger) Error(msg string)                  {}
func (l *noopLogger) Warn(msg string)                   {}
func (l *noopLogger) Tracef(format string, args ...any) {}
func (l *noopLogger) Debugf(format string, args ...any) {}
func (l *noopLogger) Infof(format string, args ...any)  {}
func (l *noopLogger) Errorf(format string, args ...any) {}
func (l *noopLogger) Warnf(format string, args ...any)  {}

type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Trace(msg string) { l.log(levelTrace, msg) }
func (l *slogLogger) Debug(msg string) { l.log(slog.LevelDebug, msg) }
func (l *slogLogger) Info(msg string)  { l.log(slog.LevelInfo, msg) }
func (l *slogLogger) Error(msg string) { l.log(slog.LevelError, msg) }
func (l *slogLogger) Warn(msg string)  { l.log(slog.LevelWarn, msg) }

func (l *slogLogger) Tracef(format string, args ...any) { l.logf(levelTrace, format, args...) }
func (l *slogLogger) Debugf(format string, args ...any) { l.logf(slog.LevelDebug, format, args...) }
func (l *slogLogger) Infof(format string, args ...any)  { l.logf(slog.LevelInfo, format, args...) }
func (l *slogLogger) Errorf(format string, args ...any) { l.logf(slog.LevelError, format, args...) }
func (l *slogLogger) Warnf(format string, args ...any)  { l.logf(slog.LevelWarn, format, args...) }

func (l *slogLogger) log(level slog.Level, msg string) {
	l.logger.Log(context.Background(), level, msg)
}

func (l *slogLogger) logf(level slog.Level, format string, args ...any) {
	l.logger.Log(context.Background(), level, fmt.Sprintf(format, args...))
}
