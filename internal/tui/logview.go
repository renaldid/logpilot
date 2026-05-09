package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// RenderLogView renders the log viewport as a string.
// width is the available width for the log area (terminal width minus sidebar).
// height is the number of visible rows.
func RenderLogView(s *State, width, height int) string {
	entries := s.VisibleEntries()
	if len(entries) == 0 {
		return styleViewport.
			Width(width).
			Height(height).
			Render(emptyMessage(s))
	}

	out := ""
	for _, e := range entries {
		out += renderEntry(e, width) + "\n"
	}

	return styleViewport.
		Width(width).
		Height(height).
		Render(out)
}

func renderEntry(e logentry.LogEntry, width int) string {
	ts := e.Timestamp.Format("15:04:05")
	levelStr := e.Level.String()
	levelColor := LevelColor(levelStr)

	tsStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Render(ts)

	svcStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Width(12).
		Render(truncate(e.Service, 12))

	lvlStyled := lipgloss.NewStyle().
		Foreground(levelColor).
		Bold(true).
		Width(7).
		Render(levelStr)

	// remaining width for message
	meta := len(ts) + 1 + 12 + 1 + 6 + 1
	msgWidth := width - meta - 2 // account for padding
	if msgWidth < 10 {
		msgWidth = 10
	}
	msg := truncate(e.Message, msgWidth)

	return fmt.Sprintf("%s %s %s %s", tsStyled, svcStyled, lvlStyled, msg)
}

func emptyMessage(s *State) string {
	if len(s.Entries) == 0 {
		return "Waiting for log entries…"
	}
	if s.FilterError != nil {
		return "Regex error: " + s.FilterError.Error()
	}
	return "No entries match current filters."
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
