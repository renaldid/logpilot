package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// RenderSidebar returns the sidebar column as a styled string.
// height is the number of rows available for the sidebar content.
func RenderSidebar(s *State, height int) string {
	title := styleSidebarTitle.Render("SERVICES")
	lines := []string{title}

	for i, svc := range s.Services {
		enabled := s.ServiceEnabled[svc]
		indicator := "●"
		if !enabled {
			indicator = "○"
		}

		color := serviceColor(svc, s.ServiceColors)
		label := lipgloss.NewStyle().Foreground(color).Render(indicator) + " " + svc

		// highlight focused item in sidebar
		if s.Focus == FocusSidebar {
			if i == s.sidebarCursor {
				label = styleServiceActive.Render("> " + indicator + " " + svc)
			}
		} else if !enabled {
			label = styleServiceInactive.Render(indicator + " " + svc)
		}

		lines = append(lines, label)
	}

	// pad to fill height
	for len(lines) < height {
		lines = append(lines, "")
	}

	content := ""
	for i, l := range lines {
		if i >= height {
			break
		}
		content += l + "\n"
	}

	return styleSidebarBorder.
		Width(sidebarWidth).
		Height(height).
		Render(fmt.Sprintf("%s", content))
}

// serviceColor returns the configured or palette color for a service.
func serviceColor(name string, configured map[string]string) lipgloss.Color {
	if c, ok := configured[name]; ok {
		return lipgloss.Color(c)
	}
	return ServiceColor(name, defaultPalette)
}
