package tui

import (
	"errors"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unknownMsg is an unrecognised message type used to test the Update fall-through path.
type unknownMsg struct{}

func newTestModel(follow bool) (Model, chan logentry.LogEntry) {
	ch := make(chan logentry.LogEntry, 64)
	state := NewState(follow, nil)
	state.Resize(120, 40)
	return New(ch, state), ch
}

func TestModel_Init_ReturnsCmd(t *testing.T) {
	m, _ := newTestModel(true)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestModel_View_BeforeResize(t *testing.T) {
	ch := make(chan logentry.LogEntry, 1)
	state := NewState(true, nil)
	m := New(ch, state)
	out := m.View()
	assert.Contains(t, out, "Initialising")
}

func TestModel_View_AfterResize(t *testing.T) {
	m, _ := newTestModel(true)
	out := m.View()
	assert.NotContains(t, out, "Initialising")
	assert.NotEmpty(t, out)
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2 := updated.(Model)
	assert.Equal(t, 80, m2.state.Width)
	assert.Equal(t, 24, m2.state.Height)
}

func TestModel_EntryMsg_AddsToState(t *testing.T) {
	m, _ := newTestModel(true)
	e := logentry.LogEntry{
		Timestamp: time.Now(),
		Service:   "api",
		Level:     logentry.LogLevelInfo,
		Message:   "test message",
	}
	updated, _ := m.Update(entryMsg{entry: e})
	m2 := updated.(Model)
	assert.Equal(t, 1, len(m2.state.Entries))
}

func TestModel_TickMsg(t *testing.T) {
	m, _ := newTestModel(true)
	updated, cmd := m.Update(tickMsg{})
	assert.NotNil(t, updated)
	assert.NotNil(t, cmd)
}

func TestModel_Key_Quit(t *testing.T) {
	m, _ := newTestModel(true)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m2 := updated.(Model)
	assert.True(t, m2.quitting)
	require.NotNil(t, cmd)
}

func TestModel_Key_Quit_CtrlC(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m2 := updated.(Model)
	assert.True(t, m2.quitting)
}

func TestModel_Key_Quit_InSearch_DoesNotQuit(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSearch)
	m.search.Focus()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m2 := updated.(Model)
	assert.False(t, m2.quitting)
}

func TestModel_Key_Help(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m2 := updated.(Model)
	assert.True(t, m2.state.ShowHelp)
	// another key closes help
	updated2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m3 := updated2.(Model)
	assert.False(t, m3.state.ShowHelp)
}

func TestModel_Key_Help_View_ShowsHelp(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.ShowHelp = true
	out := m.View()
	assert.Contains(t, out, "Keybindings")
}

func TestModel_Key_SearchFocus(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m2 := updated.(Model)
	assert.Equal(t, FocusSearch, m2.state.Focus)
}

func TestModel_Key_TabToSidebar(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m2 := updated.(Model)
	assert.Equal(t, FocusSidebar, m2.state.Focus)
}

func TestModel_Key_ToggleFollow(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m2 := updated.(Model)
	assert.False(t, m2.state.FollowMode)
}

func TestModel_Key_ToggleRegex(t *testing.T) {
	m, _ := newTestModel(true)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := updated.(Model)
	assert.True(t, m2.state.RegexMode)
}

func TestModel_Key_ClearFilter(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetQuery("hello")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m2 := updated.(Model)
	assert.Empty(t, m2.state.Query)
}

func TestModel_Key_ScrollUp(t *testing.T) {
	m, _ := newTestModel(false)
	for range 50 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2 := updated.(Model)
	assert.Equal(t, 1, m2.state.ScrollOffset)
}

func TestModel_Key_ScrollUp_K(t *testing.T) {
	m, _ := newTestModel(false)
	for range 50 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m2 := updated.(Model)
	assert.Equal(t, 1, m2.state.ScrollOffset)
}

func TestModel_Key_ScrollDown(t *testing.T) {
	m, _ := newTestModel(false)
	for range 50 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	m.state.ScrollOffset = 10
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(Model)
	assert.Equal(t, 9, m2.state.ScrollOffset)
}

func TestModel_Key_ScrollDown_J(t *testing.T) {
	m, _ := newTestModel(false)
	for range 50 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	m.state.ScrollOffset = 5
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m2 := updated.(Model)
	assert.Equal(t, 4, m2.state.ScrollOffset)
}

func TestModel_Key_PageUp(t *testing.T) {
	m, _ := newTestModel(false)
	for range 100 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m2 := updated.(Model)
	assert.Greater(t, m2.state.ScrollOffset, 0)
}

func TestModel_Key_PageDown(t *testing.T) {
	m, _ := newTestModel(false)
	for range 100 {
		m.state.AddEntry(makeEntry("api", "INFO", "line"))
	}
	m.state.ScrollOffset = 30
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m2 := updated.(Model)
	assert.Less(t, m2.state.ScrollOffset, 30)
}

func TestModel_Sidebar_Navigation(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "a"))
	m.state.AddEntry(makeEntry("worker", "INFO", "b"))
	m.state.SetFocus(FocusSidebar)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(Model)
	assert.Equal(t, 1, m2.state.SidebarCursor())
}

func TestModel_Sidebar_Up(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "a"))
	m.state.AddEntry(makeEntry("worker", "INFO", "b"))
	m.state.SetFocus(FocusSidebar)
	m.state.SidebarMoveDown()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2 := updated.(Model)
	assert.Equal(t, 0, m2.state.SidebarCursor())
}

func TestModel_Sidebar_K(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "a"))
	m.state.AddEntry(makeEntry("worker", "INFO", "b"))
	m.state.SetFocus(FocusSidebar)
	m.state.SidebarMoveDown()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m2 := updated.(Model)
	assert.Equal(t, 0, m2.state.SidebarCursor())
}

func TestModel_Sidebar_J(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "a"))
	m.state.AddEntry(makeEntry("worker", "INFO", "b"))
	m.state.SetFocus(FocusSidebar)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m2 := updated.(Model)
	assert.Equal(t, 1, m2.state.SidebarCursor())
}

func TestModel_Sidebar_Space_TogglesService(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "msg"))
	m.state.SetFocus(FocusSidebar)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m2 := updated.(Model)
	assert.False(t, m2.state.ServiceEnabled["api"])
}

func TestModel_Sidebar_Tab_ReturnsToMain(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSidebar)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m2 := updated.(Model)
	assert.Equal(t, FocusMain, m2.state.Focus)
}

func TestModel_Sidebar_Esc_ReturnsToMain(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSidebar)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(Model)
	assert.Equal(t, FocusMain, m2.state.Focus)
}

func TestModel_Search_Enter_ReturnToMain(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSearch)
	m.search.Focus()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	assert.Equal(t, FocusMain, m2.state.Focus)
}

func TestModel_Search_Esc_ReturnToMain(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSearch)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(Model)
	assert.Equal(t, FocusMain, m2.state.Focus)
}

func TestModel_View_Quitting_Empty(t *testing.T) {
	m, _ := newTestModel(true)
	m.quitting = true
	assert.Empty(t, m.View())
}

func TestModel_View_WithEntries(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "hello world"))
	out := m.View()
	assert.Contains(t, out, "hello world")
}

func TestModel_View_SearchActive(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSearch)
	out := m.View()
	assert.NotEmpty(t, out)
}

func TestModel_View_SmallTerminal(t *testing.T) {
	ch := make(chan logentry.LogEntry, 1)
	state := NewState(true, nil)
	state.Resize(10, 4) // very small
	m := New(ch, state)
	out := m.View()
	assert.NotEmpty(t, out) // should not panic
}

func TestModel_View_VerySmallHeight_ClampsBodyH(t *testing.T) {
	ch := make(chan logentry.LogEntry, 1)
	state := NewState(true, nil)
	state.Resize(80, 2) // height < 4 → bodyH = Height-3 < 1 → clamp to 1
	m := New(ch, state)
	out := m.View()
	assert.NotEmpty(t, out)
}

func TestExportCreate_Default(t *testing.T) {
	// Call the original (unmocked) exportCreate to cover its lambda body.
	t.Chdir(t.TempDir())
	f, err := exportCreate("logpilot-test.log")
	if err == nil {
		f.Close()
	}
}

func TestModel_Export_CreatesFile(t *testing.T) {
	orig := exportCreate
	defer func() { exportCreate = orig }()
	tf, err := os.CreateTemp("", "logpilot-happy-*")
	require.NoError(t, err)
	defer os.Remove(tf.Name())
	exportCreate = func(_ string) (*os.File, error) { return tf, nil }

	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "exported line"))
	m.exportLogs()
	assert.Empty(t, m.exportErr)
}

func TestWaitForEntry_ClosedChannel(t *testing.T) {
	ch := make(chan logentry.LogEntry)
	close(ch)
	cmd := waitForEntry(ch)
	msg := cmd()
	assert.Nil(t, msg)
}

func TestWaitForEntry_OpenChannel(t *testing.T) {
	ch := make(chan logentry.LogEntry, 1)
	e := makeEntry("svc", "INFO", "hello")
	ch <- e
	cmd := waitForEntry(ch)
	msg := cmd()
	em, ok := msg.(entryMsg)
	require.True(t, ok)
	assert.Equal(t, "hello", em.entry.Message)
}

func TestTickCmd_FiresTickMsg(t *testing.T) {
	cmd := tickCmd()
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		_, ok := msg.(tickMsg)
		assert.True(t, ok)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("tickCmd timed out")
	}
}

func TestModel_Update_NonKeyMsg_InSearchFocus(t *testing.T) {
	m, _ := newTestModel(true)
	m.state.SetFocus(FocusSearch)
	m.search.Focus()
	updated, _ := m.Update(unknownMsg{})
	m2 := updated.(Model)
	assert.Equal(t, FocusSearch, m2.state.Focus)
}

func TestModel_Update_NonKeyMsg_NotInSearchFocus(t *testing.T) {
	m, _ := newTestModel(true)
	updated, cmd := m.Update(unknownMsg{})
	assert.NotNil(t, updated)
	assert.Nil(t, cmd)
}

func TestModel_Key_Export(t *testing.T) {
	orig := exportCreate
	defer func() { exportCreate = orig }()
	tf, err := os.CreateTemp("", "logpilot-export-*")
	require.NoError(t, err)
	defer os.Remove(tf.Name())
	exportCreate = func(_ string) (*os.File, error) { return tf, nil }

	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "exported line"))
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m2 := updated.(Model)
	assert.Empty(t, m2.exportErr)
}

func TestModel_View_WithExportErr(t *testing.T) {
	m, _ := newTestModel(true)
	m.exportErr = "export failed: permission denied"
	out := m.View()
	assert.Contains(t, out, "export failed")
}

func TestModel_ExportLogs_CreateError(t *testing.T) {
	orig := exportCreate
	defer func() { exportCreate = orig }()
	exportCreate = func(_ string) (*os.File, error) {
		return nil, errors.New("permission denied")
	}
	m, _ := newTestModel(true)
	m.exportLogs()
	assert.Contains(t, m.exportErr, "export failed")
}

func TestModel_ExportLogs_WriteError(t *testing.T) {
	orig := exportCreate
	defer func() { exportCreate = orig }()

	// Open an existing file read-only so Write inside Export will fail.
	tf, err := os.CreateTemp("", "logpilot-ro-*")
	require.NoError(t, err)
	name := tf.Name()
	tf.Close()
	defer os.Remove(name)

	ro, err := os.Open(name)
	require.NoError(t, err)
	defer ro.Close()
	exportCreate = func(_ string) (*os.File, error) { return ro, nil }

	m, _ := newTestModel(true)
	m.state.AddEntry(makeEntry("api", "INFO", "msg"))
	m.exportLogs()
	assert.Contains(t, m.exportErr, "export failed")
}
