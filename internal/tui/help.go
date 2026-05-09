package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpRows = [][2]string{
	{"/", "focus search bar"},
	{"Tab", "switch sidebar / main"},
	{"Space", "toggle service on/off (sidebar)"},
	{"↑ / ↓", "scroll or navigate"},
	{"f", "toggle follow mode"},
	{"r", "toggle regex mode"},
	{"c", "clear all filters"},
	{"e", "export filtered logs to file"},
	{"?", "toggle this help"},
	{"q / Ctrl+C", "quit"},
}

// RenderHelp renders the help overlay centred in the terminal.
func RenderHelp(width, height int) string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Render("Keybindings") + "\n\n")

	for _, row := range helpRows {
		key := styleHelpKey.Width(14).Render(row[0])
		desc := styleHelpDesc.Render(row[1])
		sb.WriteString(key + "  " + desc + "\n")
	}

	box := styleHelp.Render(sb.String())

	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)

	padLeft := (width - bw) / 2
	padTop := (height - bh) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	topPad := strings.Repeat("\n", padTop)
	leftPad := strings.Repeat(" ", padLeft)

	lines := strings.Split(box, "\n")
	result := topPad
	for _, l := range lines {
		result += leftPad + l + "\n"
	}
	return result
}
