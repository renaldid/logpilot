package logentry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelUnknown, "UNKNOWN"},
		{LogLevel(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"DEBUG", LogLevelDebug},
		{"debug", LogLevelDebug},
		{"DBG", LogLevelDebug},
		{"TRACE", LogLevelDebug},
		{"INFO", LogLevelInfo},
		{"info", LogLevelInfo},
		{"INFORMATION", LogLevelInfo},
		{"WARN", LogLevelWarn},
		{"warn", LogLevelWarn},
		{"WARNING", LogLevelWarn},
		{"ERROR", LogLevelError},
		{"error", LogLevelError},
		{"ERR", LogLevelError},
		{"FATAL", LogLevelError},
		{"CRITICAL", LogLevelError},
		{"", LogLevelUnknown},
		{"unknown-level", LogLevelUnknown},
		{"  INFO  ", LogLevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLevel(tt.input))
		})
	}
}

func TestLogEntry_String(t *testing.T) {
	ts := time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)
	e := LogEntry{
		Timestamp: ts,
		Service:   "api",
		Level:     LogLevelInfo,
		Message:   "server started",
		Raw:       "2024-01-02T15:04:05Z INFO server started",
	}
	got := e.String()
	assert.Contains(t, got, "api")
	assert.Contains(t, got, "INFO")
	assert.Contains(t, got, "server started")
}
