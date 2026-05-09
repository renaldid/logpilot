package logentry

import (
	"fmt"
	"strings"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	LogLevelUnknown LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a LogLevel. Case-insensitive.
// Returns LogLevelUnknown for unrecognized values.
func ParseLevel(s string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG", "DBG", "TRACE":
		return LogLevelDebug
	case "INFO", "INFORMATION":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR", "ERR", "FATAL", "CRITICAL":
		return LogLevelError
	default:
		return LogLevelUnknown
	}
}

// LogEntry is the canonical log record passed through the pipeline.
type LogEntry struct {
	Timestamp time.Time
	Service   string
	Level     LogLevel
	Message   string
	Raw       string // original unmodified line
}

func (e LogEntry) String() string {
	return fmt.Sprintf("%s %s %s %s",
		e.Timestamp.Format(time.RFC3339),
		e.Service,
		e.Level,
		e.Message,
	)
}
