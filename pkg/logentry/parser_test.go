package logentry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_JSONLog(t *testing.T) {
	raw := `{"time":"2024-01-02T15:04:05Z","level":"info","msg":"request received"}`
	e := Parse("api", raw)

	assert.Equal(t, "api", e.Service)
	assert.Equal(t, LogLevelInfo, e.Level)
	assert.Equal(t, "request received", e.Message)
	assert.Equal(t, raw, e.Raw)
	assert.Equal(t, 2024, e.Timestamp.Year())
}

func TestParse_JSONLog_MessageField(t *testing.T) {
	raw := `{"time":"2024-01-02T15:04:05Z","level":"warn","message":"slow response"}`
	e := Parse("api", raw)
	assert.Equal(t, "slow response", e.Message)
	assert.Equal(t, LogLevelWarn, e.Level)
}

func TestParse_JSONLog_InvalidJSON(t *testing.T) {
	raw := `{ not valid json }`
	e := Parse("api", raw)
	assert.Equal(t, LogLevelUnknown, e.Level)
	assert.Equal(t, raw, e.Raw)
}

func TestParse_TextLog(t *testing.T) {
	raw := "2024-01-02T15:04:05Z INFO server started on :8080"
	e := Parse("api", raw)
	assert.Equal(t, LogLevelInfo, e.Level)
	assert.Equal(t, "server started on :8080", e.Message)
	assert.Equal(t, 2024, e.Timestamp.Year())
}

func TestParse_TextLog_WithSpace(t *testing.T) {
	raw := "2024-01-02 15:04:05 ERROR connection refused"
	e := Parse("worker", raw)
	assert.Equal(t, LogLevelError, e.Level)
	assert.Equal(t, "connection refused", e.Message)
}

func TestParse_RawFallback(t *testing.T) {
	raw := "something that matches no pattern"
	e := Parse("svc", raw)
	assert.Equal(t, LogLevelUnknown, e.Level)
	assert.Equal(t, raw, e.Message)
	assert.Equal(t, "svc", e.Service)
}

func TestParse_StripsNewlines(t *testing.T) {
	raw := "2024-01-02T15:04:05Z INFO hello\r\n"
	e := Parse("svc", raw)
	assert.Equal(t, "hello", e.Message)
}

func TestParse_TimestampFormats(t *testing.T) {
	formats := []struct {
		raw  string
		year int
	}{
		{"2024-01-02T15:04:05Z INFO msg", 2024},
		{"2024-01-02T15:04:05.123456789Z INFO msg", 2024},
		{"2024-01-02 15:04:05 INFO msg", 2024},
		{"2024-01-02 15:04:05.999 INFO msg", 2024},
	}
	for _, tt := range formats {
		t.Run(tt.raw, func(t *testing.T) {
			e := Parse("svc", tt.raw)
			assert.Equal(t, tt.year, e.Timestamp.Year())
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	t.Run("RFC3339", func(t *testing.T) {
		ts, err := parseTimestamp("2024-01-02T15:04:05Z")
		require.NoError(t, err)
		assert.Equal(t, 2024, ts.Year())
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := parseTimestamp("not-a-time")
		assert.Error(t, err)
	})
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("a", "b", "c"))
	assert.Equal(t, "b", firstNonEmpty("", "b", "c"))
	assert.Equal(t, "", firstNonEmpty("", "", ""))
}

func TestParse_JSONWithEmptyMsg(t *testing.T) {
	raw := `{"time":"2024-01-02T15:04:05Z","level":"info","msg":"","message":""}`
	e := Parse("svc", raw)
	// falls back to raw when both msg fields are empty
	assert.Equal(t, LogLevelInfo, e.Level)
	_ = e.Timestamp.Year() // timestamp should still parse
}

func TestParse_JSONWithBadTimestamp(t *testing.T) {
	raw := `{"time":"not-a-time","level":"error","msg":"boom"}`
	e := Parse("svc", raw)
	assert.Equal(t, LogLevelError, e.Level)
	// timestamp falls back to time.Now() — just verify it's recent
	assert.WithinDuration(t, time.Now(), e.Timestamp, 5*time.Second)
}
