package tui

import (
	"testing"

	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderLogView_Empty_ShowsWaiting(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	out := RenderLogView(s, 100, 20)
	assert.Contains(t, out, "Waiting for log entries")
}

func TestRenderLogView_FilteredEmpty_ShowsNoMatch(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	s.SetQuery("xyznotexist")
	out := RenderLogView(s, 100, 20)
	assert.Contains(t, out, "No entries match")
}

func TestRenderLogView_RegexError_FallsBackToAllEntries(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	s.ToggleRegex()
	s.SetQuery("[invalid(regex")
	// On regex error, filter falls back to showing all entries (no text filter)
	assert.NotNil(t, s.FilterError)
	out := RenderLogView(s, 100, 20)
	// Entry is still visible; error shown in status bar, not log view
	assert.Contains(t, out, "hello")
}

func TestRenderStatusBar_RegexError_ShowsInStatusBar(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	s.ToggleRegex()
	s.SetQuery("[invalid(regex")
	out := RenderStatusBar(s)
	assert.Contains(t, out, "regex error")
}

func TestRenderLogView_WithEntries_ShowsMessages(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "server started"))
	s.AddEntry(makeEntry("worker", "ERROR", "connection failed"))

	out := RenderLogView(s, 100, 20)
	assert.Contains(t, out, "server started")
	assert.Contains(t, out, "connection failed")
}

func TestRenderLogView_AllLevels_Rendered(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, l := range levels {
		s.AddEntry(makeEntry("svc", l, "msg from "+l))
	}
	out := RenderLogView(s, 100, 20)
	for _, l := range levels {
		assert.Contains(t, out, l)
	}
}

func TestTruncate_ShortString(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestTruncate_ExactLength(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 5))
}

func TestTruncate_LongString(t *testing.T) {
	result := truncate("hello world", 8)
	assert.LessOrEqual(t, len([]rune(result)), 8)
	assert.Contains(t, result, "…")
}

func TestTruncate_MaxThreeOrLess(t *testing.T) {
	result := truncate("hello", 2)
	assert.Equal(t, "he", result)
}

func TestTruncate_Unicode(t *testing.T) {
	// Unicode runes are truncated by rune count, not byte count
	result := truncate("日本語テスト", 4)
	assert.LessOrEqual(t, len([]rune(result)), 4)
}

func TestRenderEntry_UnknownLevel(t *testing.T) {
	e := makeEntry("api", "UNKNOWN", "some unknown log")
	out := renderEntry(e, 100)
	// UNKNOWN is 7 chars, fits in Width(7) — verify it renders without panic
	assert.Contains(t, out, "some unknown log")
}

func TestRenderLogView_RegexError_AllServicesDisabled_ShowsError(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	// Disabling all services causes the regex-error fallback to also return empty.
	s.ToggleService("api")
	s.ToggleRegex()
	s.SetQuery("[invalid(regex")
	require.NotNil(t, s.FilterError)
	out := RenderLogView(s, 100, 20)
	assert.Contains(t, out, "Regex error")
}

func TestRenderEntry_NarrowWidth(t *testing.T) {
	e := logentry.LogEntry{Service: "api", Message: "very long message that should be truncated"}
	// should not panic on narrow width
	out := renderEntry(e, 30)
	assert.NotEmpty(t, out)
}
