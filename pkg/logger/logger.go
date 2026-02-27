package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Logger is the logging interface used by the library.
type Logger interface {
	Info(msg string, obj any)
	Warn(msg string, obj any)
	Debug(msg string, obj any)
	Error(msg string, obj any)
}

// NopLogger discards all log messages.
type NopLogger struct{}

func (NopLogger) Info(string, any)  {}
func (NopLogger) Warn(string, any)  {}
func (NopLogger) Debug(string, any) {}
func (NopLogger) Error(string, any) {}

type writerLogger struct {
	w io.Writer
}

func (l writerLogger) write(level, msg string, obj any) {
	if l.w == nil {
		return
	}

	ts := time.Now().Format(time.RFC3339)
	if obj == nil {
		_, _ = fmt.Fprintf(l.w, "%s %-5s %s\n", ts, level, msg)
		return
	}

	b, err := json.Marshal(obj)
	if err != nil {
		_, _ = fmt.Fprintf(l.w, "%s %-5s %s obj=%q\n", ts, level, msg, fmt.Sprintf("%+v", obj))
		return
	}
	_, _ = fmt.Fprintf(l.w, "%s %-5s %s obj=%s\n", ts, level, msg, string(b))
}

// NewWriterLogger builds a logger that writes to an io.Writer.
func NewWriterLogger(w io.Writer) Logger {
	return writerLogger{w: w}
}

func (l writerLogger) Info(msg string, obj any)  { l.write("INFO", msg, obj) }
func (l writerLogger) Warn(msg string, obj any)  { l.write("WARN", msg, obj) }
func (l writerLogger) Debug(msg string, obj any) { l.write("DEBUG", msg, obj) }
func (l writerLogger) Error(msg string, obj any) { l.write("ERROR", msg, obj) }

// Debug writes a debug log when enabled and logger is non-nil.
func Debug(enabled bool, logger Logger, msg string, obj any) {
	if !enabled || logger == nil {
		return
	}
	logger.Debug(msg, obj)
}

// Debugf is a compatibility helper for format-style debug logging.
func Debugf(enabled bool, logger Logger, format string, args ...any) {
	Debug(enabled, logger, fmt.Sprintf(format, args...), nil)
}

// Info writes an info log when logger is non-nil.
func Info(logger Logger, msg string, obj any) {
	if logger == nil {
		return
	}
	logger.Info(msg, obj)
}

// Warn writes a warning log when logger is non-nil.
func Warn(logger Logger, msg string, obj any) {
	if logger == nil {
		return
	}
	logger.Warn(msg, obj)
}

// Error writes an error log when logger is non-nil.
func Error(logger Logger, msg string, obj any) {
	if logger == nil {
		return
	}
	logger.Error(msg, obj)
}
