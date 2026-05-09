package tui

import (
	"fmt"
	"io"
	"sort"

	"github.com/renaldid/logpilot/internal/pipeline"
	"github.com/renaldid/logpilot/pkg/logentry"
)

// FocusArea identifies which UI region has keyboard focus.
type FocusArea int

const (
	FocusMain    FocusArea = iota // log viewport has focus
	FocusSidebar                  // sidebar service list has focus
	FocusSearch                   // search bar has focus
)

// State holds all application state that drives the TUI render.
// It is intentionally decoupled from BubbleTea so it can be unit-tested.
type State struct {
	// Entries holds every log entry received so far.
	Entries []logentry.LogEntry

	// Filtered is the result of applying current filters to Entries.
	Filtered []logentry.LogEntry

	// Services is a sorted slice of all known service names.
	Services []string

	// ServiceEnabled controls whether each service's entries are shown.
	// nil entry or true = enabled; false = hidden.
	ServiceEnabled map[string]bool

	// LevelEnabled controls which log levels are shown.
	// nil map = all levels shown.
	LevelEnabled map[logentry.LogLevel]bool

	// Query is the current search string.
	Query string

	// RegexMode switches text search between fuzzy and regex.
	RegexMode bool

	// FollowMode auto-scrolls the viewport to the latest entry.
	FollowMode bool

	// ScrollOffset is the number of lines scrolled from the bottom (0 = bottom).
	ScrollOffset int

	// Focus is the currently focused UI area.
	Focus FocusArea

	// ShowHelp toggles the help overlay.
	ShowHelp bool

	// Width and Height are the terminal dimensions.
	Width, Height int

	// sidebarCursor is the focused service index in the sidebar.
	sidebarCursor int

	// FilterError holds the last regex compilation error, if any.
	FilterError error

	// Dropped is the number of entries dropped by the ring buffer.
	Dropped int64

	// ServiceColors maps service name → hex color string.
	ServiceColors map[string]string
}

// NewState returns a State with sensible defaults.
func NewState(followMode bool, colors map[string]string) *State {
	if colors == nil {
		colors = map[string]string{}
	}
	return &State{
		ServiceEnabled: map[string]bool{},
		LevelEnabled:   map[logentry.LogLevel]bool{},
		FollowMode:     followMode,
		ServiceColors:  colors,
	}
}

// AddEntry appends a new log entry, registers its service if new, and re-filters.
func (s *State) AddEntry(e logentry.LogEntry) {
	s.Entries = append(s.Entries, e)
	s.registerService(e.Service)
	s.applyFilters()
	if s.FollowMode {
		s.ScrollOffset = 0
	}
}

// AddEntries bulk-adds entries and filters once at the end.
func (s *State) AddEntries(entries []logentry.LogEntry) {
	for _, e := range entries {
		s.Entries = append(s.Entries, e)
		s.registerService(e.Service)
	}
	s.applyFilters()
	if s.FollowMode {
		s.ScrollOffset = 0
	}
}

// ToggleService flips the enabled state of service name.
func (s *State) ToggleService(name string) {
	// default is enabled; toggle flips between true and false
	s.ServiceEnabled[name] = !s.serviceIsEnabled(name)
	s.applyFilters()
}

// ToggleLevel flips the enabled state of a log level.
// An empty LevelEnabled map means "all enabled".
func (s *State) ToggleLevel(level logentry.LogLevel) {
	if len(s.LevelEnabled) == 0 {
		// initialise: enable all then flip this one off
		for _, l := range allLevels() {
			s.LevelEnabled[l] = true
		}
		s.LevelEnabled[level] = false
	} else {
		s.LevelEnabled[level] = !s.LevelEnabled[level]
	}
	s.applyFilters()
}

// SetQuery updates the search query and re-filters.
func (s *State) SetQuery(q string) {
	s.Query = q
	s.applyFilters()
}

// ToggleRegex switches between fuzzy and regex search.
func (s *State) ToggleRegex() {
	s.RegexMode = !s.RegexMode
	s.applyFilters()
}

// ToggleFollow toggles auto-scroll mode.
func (s *State) ToggleFollow() {
	s.FollowMode = !s.FollowMode
	if s.FollowMode {
		s.ScrollOffset = 0
	}
}

// ClearFilter resets query and all level/service filters.
func (s *State) ClearFilter() {
	s.Query = ""
	s.RegexMode = false
	s.LevelEnabled = map[logentry.LogLevel]bool{}
	for name := range s.ServiceEnabled {
		s.ServiceEnabled[name] = true
	}
	s.FilterError = nil
	s.applyFilters()
}

// ScrollUp scrolls the viewport toward older entries.
func (s *State) ScrollUp(n int) {
	s.FollowMode = false
	visible := s.visibleLines()
	maxOffset := len(s.Filtered) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	s.ScrollOffset += n
	if s.ScrollOffset > maxOffset {
		s.ScrollOffset = maxOffset
	}
}

// ScrollDown scrolls the viewport toward newer entries.
func (s *State) ScrollDown(n int) {
	s.ScrollOffset -= n
	if s.ScrollOffset < 0 {
		s.ScrollOffset = 0
		s.FollowMode = true
	}
}

// SetFocus changes which UI area has keyboard focus.
func (s *State) SetFocus(area FocusArea) {
	s.Focus = area
}

// ToggleHelp shows or hides the help overlay.
func (s *State) ToggleHelp() {
	s.ShowHelp = !s.ShowHelp
}

// Resize updates the terminal dimensions.
func (s *State) Resize(w, h int) {
	s.Width = w
	s.Height = h
}

// Export writes all filtered entries to w in plain text.
func (s *State) Export(w io.Writer) error {
	for _, e := range s.Filtered {
		if _, err := fmt.Fprintln(w, e.String()); err != nil {
			return err
		}
	}
	return nil
}

// VisibleEntries returns the slice of filtered entries that fit in the viewport.
func (s *State) VisibleEntries() []logentry.LogEntry {
	n := len(s.Filtered)
	if n == 0 {
		return nil
	}
	visible := s.visibleLines()
	if visible <= 0 {
		return nil
	}

	// Offset is measured from the bottom.
	end := n - s.ScrollOffset
	if end <= 0 {
		return nil
	}
	start := end - visible
	if start < 0 {
		start = 0
	}
	return s.Filtered[start:end]
}

// SidebarCursorValid returns true when the sidebar is focused and has services.
func (s *State) SidebarCursorValid() bool {
	return s.Focus == FocusSidebar && len(s.Services) > 0
}

// SidebarCursor returns the index of the currently highlighted service.
func (s *State) SidebarCursor() int { return s.sidebarCursor }

// SidebarCursorService returns the service name at the current cursor, or "".
func (s *State) SidebarCursorService() string {
	if len(s.Services) == 0 {
		return ""
	}
	return s.Services[s.sidebarCursor]
}

// SidebarMoveUp moves the sidebar cursor up.
func (s *State) SidebarMoveUp() {
	if s.sidebarCursor > 0 {
		s.sidebarCursor--
	}
}

// SidebarMoveDown moves the sidebar cursor down.
func (s *State) SidebarMoveDown() {
	if s.sidebarCursor < len(s.Services)-1 {
		s.sidebarCursor++
	}
}

// ToggleCursorService toggles the currently highlighted service.
func (s *State) ToggleCursorService() {
	if svc := s.SidebarCursorService(); svc != "" {
		s.ToggleService(svc)
	}
}

// — private helpers —

func (s *State) registerService(name string) {
	if _, known := s.ServiceEnabled[name]; !known {
		s.ServiceEnabled[name] = true
		s.Services = append(s.Services, name)
		sort.Strings(s.Services)
	}
}

func (s *State) serviceIsEnabled(name string) bool {
	v, ok := s.ServiceEnabled[name]
	if !ok {
		return true // default: enabled
	}
	return v
}

func (s *State) applyFilters() {
	svcFilter := make(map[string]bool)
	for name, enabled := range s.ServiceEnabled {
		svcFilter[name] = enabled
	}

	var lvlFilter map[logentry.LogLevel]bool
	if len(s.LevelEnabled) > 0 {
		lvlFilter = s.LevelEnabled
	}

	opts := pipeline.FilterOptions{
		EnabledServices: svcFilter,
		EnabledLevels:   lvlFilter,
		Query:           s.Query,
		RegexMode:       s.RegexMode,
	}

	result, err := pipeline.Apply(s.Entries, opts)
	s.FilterError = err
	if err != nil {
		// on filter error, show all entries without text filter
		opts.Query = ""
		result, _ = pipeline.Apply(s.Entries, opts)
	}
	s.Filtered = result
}

func (s *State) visibleLines() int {
	// layout: sidebar+viewport rows = Height - searchbar(1) - statusbar(1) - border(1)
	return s.Height - 3
}

func allLevels() []logentry.LogLevel {
	return []logentry.LogLevel{
		logentry.LogLevelDebug,
		logentry.LogLevelInfo,
		logentry.LogLevelWarn,
		logentry.LogLevelError,
		logentry.LogLevelUnknown,
	}
}
