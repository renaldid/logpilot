package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderStatusBar returns the status bar string at terminal width.
func RenderStatusBar(s *State) string {
	parts := []string{}

	parts = append(parts, fmt.Sprintf("%d services", len(s.Services)))
	parts = append(parts, fmt.Sprintf("%d logs", len(s.Filtered)))

	if s.FollowMode {
		parts = append(parts, styleStatusFollow.Render("follow: ON"))
	} else {
		parts = append(parts, "follow: OFF")
	}

	if s.RegexMode {
		parts = append(parts, styleStatusRegex.Render("regex: ON"))
	}

	if s.FilterError != nil {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("regex error"))
	}

	if s.Dropped > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).
			Render(fmt.Sprintf("dropped: %d", s.Dropped)))
	}

	left := strings.Join(parts, " · ")
	right := helpHint(s.Focus)

	gap := s.Width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	return styleStatusBar.Width(s.Width).Render(bar)
}

func helpHint(focus FocusArea) string {
	switch focus {
	case FocusSidebar:
		return "↑↓ navigate · space toggle · tab main"
	case FocusSearch:
		return "enter confirm · esc cancel"
	default:
		return "/ search · tab sidebar · ? help · q quit"
	}
}
