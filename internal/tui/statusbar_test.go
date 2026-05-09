package tui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderStatusBar_Basic(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	out := RenderStatusBar(s)
	assert.Contains(t, out, "1 services")
	assert.Contains(t, out, "1 logs")
}

func TestRenderStatusBar_FollowOn(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	out := RenderStatusBar(s)
	assert.Contains(t, out, "follow: ON")
}

func TestRenderStatusBar_FollowOff(t *testing.T) {
	s := NewState(false, nil)
	s.Resize(120, 30)
	out := RenderStatusBar(s)
	assert.Contains(t, out, "follow: OFF")
}

func TestRenderStatusBar_RegexOn(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.ToggleRegex()
	out := RenderStatusBar(s)
	assert.Contains(t, out, "regex: ON")
}

func TestRenderStatusBar_FilterError(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.ToggleRegex()
	s.SetQuery("[invalid")
	out := RenderStatusBar(s)
	assert.Contains(t, out, "regex error")
}

func TestRenderStatusBar_Dropped(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.Dropped = 42
	out := RenderStatusBar(s)
	assert.Contains(t, out, "dropped: 42")
}

func TestRenderStatusBar_HelpHints(t *testing.T) {
	tests := []struct {
		focus FocusArea
		want  string
	}{
		{FocusMain, "? help"},
		{FocusSidebar, "navigate"},
		{FocusSearch, "enter confirm"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("focus=%d", tt.focus), func(t *testing.T) {
			s := NewState(true, nil)
			s.Resize(120, 30)
			s.SetFocus(tt.focus)
			out := RenderStatusBar(s)
			assert.Contains(t, out, tt.want)
		})
	}
}

func TestRenderStatusBar_NarrowTerminal(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(20, 10) // very narrow
	out := RenderStatusBar(s)
	assert.NotEmpty(t, out) // should not panic
}
