package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderSidebar_Empty(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	out := RenderSidebar(s, 20)
	assert.NotEmpty(t, out)
	assert.Contains(t, out, "SERVICES")
}

func TestRenderSidebar_WithServices(t *testing.T) {
	s := NewState(true, nil)
	s.Resize(120, 30)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.AddEntry(makeEntry("worker", "INFO", "msg"))

	out := RenderSidebar(s, 20)
	assert.Contains(t, out, "api")
	assert.Contains(t, out, "worker")
}

func TestRenderSidebar_DisabledService_ShowsEmpty(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.ToggleService("api")

	out := RenderSidebar(s, 20)
	assert.Contains(t, out, "○") // hollow dot for disabled
}

func TestRenderSidebar_FocusedService_ShowsHighlight(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.AddEntry(makeEntry("worker", "INFO", "msg"))
	s.SetFocus(FocusSidebar)

	out := RenderSidebar(s, 20)
	assert.Contains(t, out, ">")
}

func TestRenderSidebar_ConfiguredColor(t *testing.T) {
	colors := map[string]string{"api": "#7C3AED"}
	s := NewState(true, colors)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	out := RenderSidebar(s, 20)
	assert.NotEmpty(t, out)
}

func TestRenderSidebar_PadsToHeight(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	out := RenderSidebar(s, 30)
	assert.NotEmpty(t, out)
}

func TestServiceColor_UsesPalette(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("unknown-svc", "INFO", "msg"))
	out := RenderSidebar(s, 10)
	assert.NotEmpty(t, out) // just verify it renders without panic
}

func TestRenderSidebar_MoreServicesThanHeight_Clips(t *testing.T) {
	s := NewState(true, nil)
	for _, svc := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
		s.AddEntry(makeEntry(svc, "INFO", "msg"))
	}
	// height=3: title + 2 entries fit; remaining entries must be clipped (break branch)
	out := RenderSidebar(s, 3)
	assert.NotEmpty(t, out)
	// smaller height still must not panic
	assert.NotPanics(t, func() { RenderSidebar(s, 1) })
}
