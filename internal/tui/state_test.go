package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEntry(svc, level, msg string) logentry.LogEntry {
	return logentry.LogEntry{
		Timestamp: time.Now(),
		Service:   svc,
		Level:     logentry.ParseLevel(level),
		Message:   msg,
		Raw:       msg,
	}
}

func TestNewState_Defaults(t *testing.T) {
	s := NewState(true, nil)
	assert.True(t, s.FollowMode)
	assert.NotNil(t, s.ServiceEnabled)
	assert.NotNil(t, s.LevelEnabled)
	assert.NotNil(t, s.ServiceColors)
	assert.Empty(t, s.Entries)
}

func TestState_AddEntry_RegistersService(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	assert.Contains(t, s.Services, "api")
	assert.Equal(t, 1, len(s.Filtered))
}

func TestState_AddEntry_FollowMode_ResetsOffset(t *testing.T) {
	s := NewState(true, nil)
	s.ScrollOffset = 10
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_AddEntry_NoFollow_KeepsOffset(t *testing.T) {
	s := NewState(false, nil)
	s.ScrollOffset = 5
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	assert.Equal(t, 5, s.ScrollOffset)
}

func TestState_AddEntries_Bulk(t *testing.T) {
	s := NewState(true, nil)
	entries := []logentry.LogEntry{
		makeEntry("api", "INFO", "a"),
		makeEntry("worker", "WARN", "b"),
		makeEntry("db", "ERROR", "c"),
	}
	s.AddEntries(entries)
	assert.Equal(t, 3, len(s.Entries))
	assert.Equal(t, 3, len(s.Filtered))
	assert.Len(t, s.Services, 3)
}

func TestState_AddEntries_Empty(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntries(nil)
	assert.Empty(t, s.Entries)
}

func TestState_Services_AreSorted(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("zservice", "INFO", "z"))
	s.AddEntry(makeEntry("aservice", "INFO", "a"))
	s.AddEntry(makeEntry("mservice", "INFO", "m"))
	assert.Equal(t, []string{"aservice", "mservice", "zservice"}, s.Services)
}

func TestState_Services_NoDuplicates(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "first"))
	s.AddEntry(makeEntry("api", "INFO", "second"))
	assert.Len(t, s.Services, 1)
}

func TestState_ToggleService_UnregisteredService(t *testing.T) {
	s := NewState(true, nil)
	// "ghost" was never added via AddEntry, so serviceIsEnabled hits the !ok path.
	s.ToggleService("ghost")
	assert.False(t, s.ServiceEnabled["ghost"])
	s.ToggleService("ghost")
	assert.True(t, s.ServiceEnabled["ghost"])
}

func TestState_ToggleService_HidesEntries(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "api-msg"))
	s.AddEntry(makeEntry("worker", "INFO", "worker-msg"))

	s.ToggleService("api")
	assert.False(t, s.ServiceEnabled["api"])
	for _, e := range s.Filtered {
		assert.NotEqual(t, "api", e.Service)
	}
}

func TestState_ToggleService_RestoresEntries(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.ToggleService("api") // disable
	s.ToggleService("api") // re-enable
	assert.True(t, s.ServiceEnabled["api"])
	assert.Len(t, s.Filtered, 1)
}

func TestState_ToggleLevel_FiltersLevel(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "info-msg"))
	s.AddEntry(makeEntry("api", "ERROR", "error-msg"))

	s.ToggleLevel(logentry.LogLevelInfo) // disables INFO
	for _, e := range s.Filtered {
		assert.NotEqual(t, logentry.LogLevelInfo, e.Level)
	}
}

func TestState_ToggleLevel_FromEmpty_EnablesAll_ThenDisablesOne(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "a"))
	s.AddEntry(makeEntry("api", "DEBUG", "b"))
	s.AddEntry(makeEntry("api", "WARN", "c"))

	// First toggle: initializes LevelEnabled with all enabled except INFO
	s.ToggleLevel(logentry.LogLevelInfo)
	assert.False(t, s.LevelEnabled[logentry.LogLevelInfo])
	assert.True(t, s.LevelEnabled[logentry.LogLevelDebug])
}

func TestState_ToggleLevel_ToggleTwice_ReEnables(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "ERROR", "err"))
	s.ToggleLevel(logentry.LogLevelError) // disable
	s.ToggleLevel(logentry.LogLevelError) // re-enable
	assert.True(t, s.LevelEnabled[logentry.LogLevelError])
	assert.Len(t, s.Filtered, 1)
}

func TestState_SetQuery_FuzzySearch(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "server started"))
	s.AddEntry(makeEntry("api", "INFO", "connection refused"))

	s.SetQuery("server")
	assert.Len(t, s.Filtered, 1)
	assert.Equal(t, "server started", s.Filtered[0].Message)
}

func TestState_SetQuery_Empty_ShowsAll(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "a"))
	s.AddEntry(makeEntry("api", "INFO", "b"))
	s.SetQuery("a")
	s.SetQuery("")
	assert.Len(t, s.Filtered, 2)
}

func TestState_ToggleRegex_SwitchesMode(t *testing.T) {
	s := NewState(true, nil)
	assert.False(t, s.RegexMode)
	s.ToggleRegex()
	assert.True(t, s.RegexMode)
	s.ToggleRegex()
	assert.False(t, s.RegexMode)
}

func TestState_RegexMode_ValidPattern(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "request received"))
	s.AddEntry(makeEntry("api", "ERROR", "error occurred"))
	s.ToggleRegex()
	s.SetQuery("^request")
	assert.Nil(t, s.FilterError)
	require.Len(t, s.Filtered, 1)
	assert.Equal(t, "request received", s.Filtered[0].Message)
}

func TestState_RegexMode_InvalidPattern_ShowsError(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.ToggleRegex()
	s.SetQuery("[invalid(regex")
	assert.NotNil(t, s.FilterError)
	// on error, falls back to all entries (no text filter)
	assert.Len(t, s.Filtered, 1)
}

func TestState_ToggleFollow_TogglesFlag(t *testing.T) {
	s := NewState(true, nil)
	assert.True(t, s.FollowMode)
	s.ToggleFollow()
	assert.False(t, s.FollowMode)
	s.ToggleFollow()
	assert.True(t, s.FollowMode)
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_ToggleFollow_On_ResetsOffset(t *testing.T) {
	s := NewState(false, nil)
	s.ScrollOffset = 20
	s.ToggleFollow() // turn on
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_ClearFilter_ResetsEverything(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	s.SetQuery("hello")
	s.ToggleRegex()
	s.ToggleLevel(logentry.LogLevelInfo)

	s.ClearFilter()
	assert.Empty(t, s.Query)
	assert.False(t, s.RegexMode)
	assert.Empty(t, s.LevelEnabled)
	assert.Nil(t, s.FilterError)
	assert.Len(t, s.Filtered, 1)
}

func TestState_ClearFilter_ReEnablesAllServices(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.ToggleService("api") // disable
	s.ClearFilter()
	assert.True(t, s.ServiceEnabled["api"])
}

func TestState_ScrollUp_DisablesFollow(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 40)
	for range 50 {
		s.AddEntry(makeEntry("api", "INFO", "line"))
	}
	s.ScrollUp(5)
	assert.False(t, s.FollowMode)
	assert.Equal(t, 5, s.ScrollOffset)
}

func TestState_ScrollUp_ClampedAtMax(t *testing.T) {
	s := NewState(false, nil)
	s.Resize(120, 10)
	for range 5 {
		s.AddEntry(makeEntry("api", "INFO", "line"))
	}
	s.ScrollUp(1000) // more than max
	// visibleLines = 10 - 3 = 7; maxOffset = 5 - 7 = -2 → clamped to 0
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_ScrollDown_ResetsFollowWhenAtBottom(t *testing.T) {
	s := NewState(false, nil)
	s.Resize(120, 40)
	for range 50 {
		s.AddEntry(makeEntry("api", "INFO", "line"))
	}
	s.ScrollUp(10)
	assert.False(t, s.FollowMode)
	// scroll down MORE than current offset to go past zero → triggers follow
	s.ScrollDown(20)
	assert.True(t, s.FollowMode)
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_ScrollDown_DoesNotGoNegative(t *testing.T) {
	s := NewState(false, nil)
	s.ScrollOffset = 3
	s.ScrollDown(100)
	assert.Equal(t, 0, s.ScrollOffset)
}

func TestState_SetFocus_UpdatesFocus(t *testing.T) {
	s := NewState(true, nil)
	assert.Equal(t, FocusMain, s.Focus)
	s.SetFocus(FocusSidebar)
	assert.Equal(t, FocusSidebar, s.Focus)
	s.SetFocus(FocusSearch)
	assert.Equal(t, FocusSearch, s.Focus)
}

func TestState_ToggleHelp(t *testing.T) {
	s := NewState(true, nil)
	assert.False(t, s.ShowHelp)
	s.ToggleHelp()
	assert.True(t, s.ShowHelp)
	s.ToggleHelp()
	assert.False(t, s.ShowHelp)
}

func TestState_Resize(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 40)
	assert.Equal(t, 120, s.Width)
	assert.Equal(t, 40, s.Height)
}

func TestState_Export_WritesFilteredEntries(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	s.AddEntry(makeEntry("worker", "ERROR", "boom"))
	s.ToggleService("worker") // hide worker

	var buf bytes.Buffer
	err := s.Export(&buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "hello")
	assert.NotContains(t, out, "boom")
}

func TestState_Export_WriterError(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "hello"))
	err := s.Export(&errorWriter{})
	assert.Error(t, err)
}

func TestState_VisibleEntries_Empty(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 40)
	assert.Nil(t, s.VisibleEntries())
}

func TestState_VisibleEntries_LessThanViewport(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 40)
	s.AddEntry(makeEntry("api", "INFO", "a"))
	s.AddEntry(makeEntry("api", "INFO", "b"))
	visible := s.VisibleEntries()
	assert.Len(t, visible, 2)
}

func TestState_VisibleEntries_MoreThanViewport(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 10) // visibleLines = 10-3 = 7
	for range 20 {
		s.AddEntry(makeEntry("api", "INFO", "line"))
	}
	visible := s.VisibleEntries()
	assert.Len(t, visible, 7)
}

func TestState_VisibleEntries_ScrolledUp(t *testing.T) {
	s := NewState(false, nil)
	s.Resize(120, 10) // visibleLines = 7
	msgs := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	for _, m := range msgs {
		s.AddEntry(makeEntry("api", "INFO", m))
	}
	s.ScrollUp(2)
	visible := s.VisibleEntries()
	// last 7 entries are h,i,j (bottom) → scrolled up 2 → shows b,c,d,e,f,g,h
	assert.Len(t, visible, 7)
	assert.Equal(t, "h", visible[len(visible)-1].Message)
}

func TestState_VisibleEntries_ZeroHeight(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 0)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	assert.Nil(t, s.VisibleEntries())
}

func TestState_VisibleEntries_ScrollOffsetPastEnd(t *testing.T) {
	s := NewState(false, nil)
	s.Resize(120, 10)
	s.AddEntry(makeEntry("api", "INFO", "only"))
	s.ScrollOffset = 100 // beyond end
	visible := s.VisibleEntries()
	assert.Nil(t, visible)
}

func TestState_SidebarCursorValid_WhenFocusSidebar(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.SetFocus(FocusSidebar)
	assert.True(t, s.SidebarCursorValid())
}

func TestState_SidebarCursorValid_WhenNoServices(t *testing.T) {
	s := NewState(true, nil)
	s.SetFocus(FocusSidebar)
	assert.False(t, s.SidebarCursorValid())
}

func TestState_SidebarCursorValid_WhenFocusMain(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	assert.False(t, s.SidebarCursorValid())
}

func TestState_WithColors(t *testing.T) {
	colors := map[string]string{"api": "#7C3AED"}
	s := NewState(true, colors)
	assert.Equal(t, "#7C3AED", s.ServiceColors["api"])
}

func TestState_AddEntries_FollowMode_KeepsOffset(t *testing.T) {
	s := NewState(false, nil)
	s.ScrollOffset = 10
	s.AddEntries([]logentry.LogEntry{
		makeEntry("api", "INFO", "msg"),
	})
	assert.Equal(t, 10, s.ScrollOffset)
}

// errorWriter always returns an error on Write.
type errorWriter struct{}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, assert.AnError
}

func TestLevelColor_AllLevels(t *testing.T) {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "UNKNOWN", "BOGUS"}
	for _, l := range levels {
		c := LevelColor(l)
		assert.NotEmpty(t, string(c), "level %s should have a color", l)
	}
}

func TestServiceColor_Deterministic(t *testing.T) {
	palette := []string{"#aaa", "#bbb", "#ccc"}
	c1 := ServiceColor("api", palette)
	c2 := ServiceColor("api", palette)
	assert.Equal(t, c1, c2)
}

func TestServiceColor_EmptyPalette(t *testing.T) {
	c := ServiceColor("api", nil)
	assert.Equal(t, lipglossDefaultColor, string(c))
}

func TestServiceColor_NegativeHash(t *testing.T) {
	// Name with characters that produce a negative hash sum
	// '\x80' produces a large rune value that when summed causes overflow
	palette := []string{"#111"}
	c := ServiceColor(strings.Repeat("z", 100), palette)
	assert.NotEmpty(t, string(c))
}

const lipglossDefaultColor = "#AAAAAA"
