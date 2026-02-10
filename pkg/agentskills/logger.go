package agentskills

import (
	"fmt"
	"io"
)

// Logger is the logging interface used by the library.
type Logger interface {
	Debugf(format string, args ...any)
}

// NopLogger discards all log messages.
type NopLogger struct{}

// Debugf discards log messages.
func (NopLogger) Debugf(string, ...any) {}

type writerLogger struct {
	w io.Writer
}

// Debugf writes a formatted log line to the configured writer.
func (l writerLogger) Debugf(format string, args ...any) {
	if l.w == nil {
		return
	}
	_, _ = fmt.Fprintf(l.w, format+"\n", args...)
}

// NewWriterLogger builds a logger that writes to an io.Writer.
func NewWriterLogger(w io.Writer) Logger {
	return writerLogger{w: w}
}

func debugf(enabled bool, logger Logger, format string, args ...any) {
	if !enabled || logger == nil {
		return
	}
	logger.Debugf(format, args...)
}

