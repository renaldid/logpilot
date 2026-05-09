package tui

import "github.com/charmbracelet/lipgloss"

const sidebarWidth = 22

var (
	// Sidebar
	styleServiceActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	styleServiceInactive = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#555555"))

	styleSidebarTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				PaddingBottom(1)

	styleSidebarBorder = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false).
				BorderForeground(lipgloss.Color("#333333"))

	// Log viewport
	styleViewport = lipgloss.NewStyle().
			PaddingLeft(1)

	// Level colors
	levelColors = map[string]lipgloss.Color{
		"DEBUG":   "#555555",
		"INFO":    "#22C55E",
		"WARN":    "#F59E0B",
		"ERROR":   "#EF4444",
		"UNKNOWN": "#777777",
	}

	// Search bar
	styleSearchPrompt = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED"))

	styleSearchActive = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("#7C3AED"))

	styleSearchInactive = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("#333333"))

	// Status bar
	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E2E")).
			Foreground(lipgloss.Color("#888888")).
			PaddingLeft(1).
			PaddingRight(1)

	styleStatusFollow = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#22C55E"))

	styleStatusRegex = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	// Help overlay
	styleHelp = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			Background(lipgloss.Color("#0D0D1A"))

	styleHelpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	styleHelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))
)

// LevelColor returns the lipgloss color for a log level string.
func LevelColor(level string) lipgloss.Color {
	if c, ok := levelColors[level]; ok {
		return c
	}
	return levelColors["UNKNOWN"]
}

// ServiceColor picks a color from a palette based on the service name's hash.
func ServiceColor(name string, palette []string) lipgloss.Color {
	if len(palette) == 0 {
		return lipgloss.Color("#AAAAAA")
	}
	var h uint64
	for _, c := range name {
		h = h*31 + uint64(c)
	}
	return lipgloss.Color(palette[h%uint64(len(palette))])
}

// defaultPalette is used when no color config is provided.
var defaultPalette = []string{
	"#7C3AED", "#059669", "#D97706", "#2563EB",
	"#DC2626", "#7C3AED", "#0891B2", "#65A30D",
}
