package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderHelp_ContainsKeybindings(t *testing.T) {
	out := RenderHelp(120, 40)
	assert.Contains(t, out, "Keybindings")
	assert.Contains(t, out, "quit")
	assert.Contains(t, out, "search")
	assert.Contains(t, out, "follow")
}

func TestRenderHelp_NarrowTerminal(t *testing.T) {
	out := RenderHelp(10, 5) // very small terminal
	assert.NotEmpty(t, out)  // should not panic
}

func TestRenderHelp_AllRowsPresent(t *testing.T) {
	out := RenderHelp(120, 40)
	for _, row := range helpRows {
		assert.Contains(t, out, row[0], "missing key: %s", row[0])
	}
}
