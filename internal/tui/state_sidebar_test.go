package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState_SidebarCursor_Initial(t *testing.T) {
	s := NewState(true, nil)
	assert.Equal(t, 0, s.SidebarCursor())
}

func TestState_SidebarCursorService_Empty(t *testing.T) {
	s := NewState(true, nil)
	assert.Equal(t, "", s.SidebarCursorService())
}

func TestState_SidebarCursorService_WithServices(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.AddEntry(makeEntry("worker", "INFO", "msg"))
	// Services are sorted: ["api", "worker"]
	assert.Equal(t, "api", s.SidebarCursorService())
}

func TestState_SidebarMoveDown(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.AddEntry(makeEntry("worker", "INFO", "msg"))
	s.SidebarMoveDown()
	assert.Equal(t, 1, s.SidebarCursor())
	assert.Equal(t, "worker", s.SidebarCursorService())
}

func TestState_SidebarMoveDown_ClampedAtEnd(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.SidebarMoveDown()
	s.SidebarMoveDown()
	assert.Equal(t, 0, s.SidebarCursor()) // clamped at 0 (only 1 service)
}

func TestState_SidebarMoveUp_ClampedAtZero(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "msg"))
	s.SidebarMoveUp()
	assert.Equal(t, 0, s.SidebarCursor()) // can't go below 0
}

func TestState_SidebarMoveUpAndDown(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "a"))
	s.AddEntry(makeEntry("worker", "INFO", "b"))
	s.AddEntry(makeEntry("db", "INFO", "c"))
	// sorted: ["api", "db", "worker"]
	s.SidebarMoveDown()
	s.SidebarMoveDown()
	assert.Equal(t, 2, s.SidebarCursor())
	s.SidebarMoveUp()
	assert.Equal(t, 1, s.SidebarCursor())
}

func TestState_ToggleCursorService_TogglesCurrentService(t *testing.T) {
	s := NewState(true, nil)
	s.AddEntry(makeEntry("api", "INFO", "a"))
	s.AddEntry(makeEntry("worker", "INFO", "b"))
	// cursor at "api"
	s.ToggleCursorService()
	assert.False(t, s.ServiceEnabled["api"])
	// re-toggle
	s.ToggleCursorService()
	assert.True(t, s.ServiceEnabled["api"])
}

func TestState_ToggleCursorService_Empty_DoesNotPanic(t *testing.T) {
	s := NewState(true, nil)
	assert.NotPanics(t, func() { s.ToggleCursorService() })
}
