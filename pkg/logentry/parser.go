package logentry

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// common log line patterns: "2006-01-02T15:04:05Z INFO message"
var textPattern = regexp.MustCompile(
	`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:[\.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+(\w+)\s+(.+)$`,
)

type jsonLog struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
}

// Parse attempts to extract structured fields from a raw log line.
// Falls back to a raw entry with LogLevelUnknown when no pattern matches.
func Parse(service, raw string) LogEntry {
	raw = strings.TrimRight(raw, "\r\n")
	entry := LogEntry{
		Timestamp: time.Now(),
		Service:   service,
		Level:     LogLevelUnknown,
		Message:   raw,
		Raw:       raw,
	}

	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		var j jsonLog
		if err := json.Unmarshal([]byte(raw), &j); err == nil {
			entry.Level = ParseLevel(j.Level)
			entry.Message = firstNonEmpty(j.Msg, j.Message, raw)
			if t, err := time.Parse(time.RFC3339Nano, j.Time); err == nil {
				entry.Timestamp = t
			}
			return entry
		}
	}

	if m := textPattern.FindStringSubmatch(raw); len(m) == 4 {
		if t, err := parseTimestamp(m[1]); err == nil {
			entry.Timestamp = t
		}
		entry.Level = ParseLevel(m[2])
		entry.Message = m[3]
	}

	return entry
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var tsFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
}

func parseTimestamp(s string) (time.Time, error) {
	for _, f := range tsFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &time.ParseError{Value: s}
}
