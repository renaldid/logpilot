package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// entryMsg carries a new log entry from the source goroutine.
type entryMsg struct{ entry logentry.LogEntry }

// tickMsg is used for periodic UI refresh.
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Model is the top-level BubbleTea model for logpilot.
// It is a thin wrapper around State and delegates all business logic there.
type Model struct {
	state     *State
	entries   <-chan logentry.LogEntry
	search    textinput.Model
	quitting  bool
	exportErr string
}

// New creates a Model from an entry channel and initial state.
func New(entries <-chan logentry.LogEntry, state *State) Model {
	ti := textinput.New()
	ti.Placeholder = "filter logs…"
	ti.CharLimit = 256
	ti.Prompt = "/ "
	ti.PromptStyle = styleSearchPrompt

	return Model{
		state:   state,
		entries: entries,
		search:  ti,
	}
}

// waitForEntry returns a Cmd that reads the next log entry from the channel.
func waitForEntry(ch <-chan logentry.LogEntry) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return nil
		}
		return entryMsg{entry: e}
	}
}

// Init starts the entry reader and the tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEntry(m.entries),
		tickCmd(),
	)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.state.Resize(msg.Width, msg.Height)
		return m, nil

	case entryMsg:
		m.state.AddEntry(msg.entry)
		return m, waitForEntry(m.entries)

	case tickMsg:
		return m, tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// forward to search textinput when focused
	if m.state.Focus == FocusSearch {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.state.SetQuery(m.search.Value())
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (work in any focus area)
	switch msg.String() {
	case "ctrl+c", "q":
		if m.state.Focus != FocusSearch {
			m.quitting = true
			return m, tea.Quit
		}
	case "?":
		if m.state.Focus != FocusSearch {
			m.state.ToggleHelp()
			return m, nil
		}
	}

	// Help overlay: any key closes it
	if m.state.ShowHelp {
		m.state.ToggleHelp()
		return m, nil
	}

	switch m.state.Focus {
	case FocusSearch:
		return m.handleSearchKey(msg)
	case FocusSidebar:
		return m.handleSidebarKey(msg)
	default:
		return m.handleMainKey(msg)
	}
}

func (m Model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.state.SetFocus(FocusSearch)
		m.search.Focus()
		return m, textinput.Blink
	case "tab":
		m.state.SetFocus(FocusSidebar)
		return m, nil
	case "f":
		m.state.ToggleFollow()
	case "r":
		m.state.ToggleRegex()
	case "c":
		m.state.ClearFilter()
		m.search.SetValue("")
	case "e":
		m.exportLogs()
	case "up", "k":
		m.state.ScrollUp(1)
	case "down", "j":
		m.state.ScrollDown(1)
	case "pgup":
		m.state.ScrollUp(m.state.visibleLines() / 2)
	case "pgdown":
		m.state.ScrollDown(m.state.visibleLines() / 2)
	}
	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "esc":
		m.state.SetFocus(FocusMain)
	case "up", "k":
		m.state.SidebarMoveUp()
	case "down", "j":
		m.state.SidebarMoveDown()
	case " ":
		m.state.ToggleCursorService()
	}
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.search.Blur()
		m.state.SetFocus(FocusMain)
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.state.SetQuery(m.search.Value())
	return m, cmd
}

// View renders the full TUI layout.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.state.Width == 0 {
		return "Initialising…"
	}

	if m.state.ShowHelp {
		return RenderHelp(m.state.Width, m.state.Height)
	}

	bodyH := m.state.Height - 3 // search + status + divider
	if bodyH < 1 {
		bodyH = 1
	}

	logW := m.state.Width - sidebarWidth - 1
	if logW < 20 {
		logW = 20
	}

	sidebar := RenderSidebar(m.state, bodyH)
	logview := RenderLogView(m.state, logW, bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, logview)

	searchActive := m.state.Focus == FocusSearch
	searchView := m.search.View()
	if searchActive {
		searchView = styleSearchActive.Width(m.state.Width - 4).Render(searchView)
	} else {
		searchView = styleSearchInactive.Width(m.state.Width - 4).Render(searchView)
	}

	status := RenderStatusBar(m.state)

	var export string
	if m.exportErr != "" {
		export = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(m.exportErr)
	}

	return strings.Join([]string{body, searchView, status + export}, "\n")
}

// exportCreate is the file-creation function; overridable in tests.
var exportCreate = func(name string) (*os.File, error) { return os.Create(name) }

// exportLogs writes filtered logs to a timestamped file.
func (m *Model) exportLogs() {
	fname := fmt.Sprintf("logpilot-export-%s.log", time.Now().Format("20060102-150405"))
	f, err := exportCreate(fname)
	if err != nil {
		m.exportErr = "export failed: " + err.Error()
		return
	}
	defer f.Close()
	if err := m.state.Export(f); err != nil {
		m.exportErr = "export failed: " + err.Error()
		return
	}
	m.exportErr = ""
}
