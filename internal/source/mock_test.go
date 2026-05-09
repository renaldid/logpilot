package source

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockEntries(msgs ...string) []logentry.LogEntry {
	out := make([]logentry.LogEntry, len(msgs))
	for i, m := range msgs {
		out[i] = logentry.LogEntry{
			Timestamp: time.Now(),
			Service:   "mock",
			Level:     logentry.LogLevelInfo,
			Message:   m,
		}
	}
	return out
}

func TestMockSource_Name(t *testing.T) {
	m := NewMock("my-service", nil)
	assert.Equal(t, "my-service", m.Name())
}

func TestMockSource_EmitsAllEntries(t *testing.T) {
	e := mockEntries("a", "b", "c")
	m := NewMock("svc", e)

	ch, err := m.Start(context.Background())
	require.NoError(t, err)

	var got []logentry.LogEntry
	for entry := range ch {
		got = append(got, entry)
	}
	assert.Len(t, got, 3)
}

func TestMockSource_DoubleStart_ReturnsError(t *testing.T) {
	m := NewMock("svc", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := m.Start(ctx)
	require.NoError(t, err)

	_, err = m.Start(ctx)
	assert.ErrorIs(t, err, ErrAlreadyStarted)
	_ = m.Stop()
}

func TestMockSource_Stop_BeforeStart_ReturnsError(t *testing.T) {
	m := NewMock("svc", nil)
	assert.ErrorIs(t, m.Stop(), ErrNotStarted)
}

func TestMockSource_Stop_StopsCleanly(t *testing.T) {
	m := NewMock("svc", nil)
	ctx := context.Background()
	_, err := m.Start(ctx)
	require.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestMockSource_ContextCancel_ClosesChannel(t *testing.T) {
	// source with no entries will block until ctx cancelled
	m := NewMock("svc", nil)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := m.Start(ctx)
	require.NoError(t, err)

	cancel()
	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("channel did not close after context cancel")
	}
}

func TestMockWithError_Start_ReturnsError(t *testing.T) {
	boom := errors.New("init failed")
	m := NewMockWithError("bad", boom)

	_, err := m.Start(context.Background())
	assert.ErrorIs(t, err, boom)
}

func TestMockSource_ContextCancel_MidSend_ExitsCleanly(t *testing.T) {
	// unbuffered channel + immediate cancel → ctx.Done() branch in goroutine
	e := mockEntries("a", "b", "c")
	m := NewMock("svc", e)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := m.Start(ctx)
	require.NoError(t, err)

	// cancel before consuming any entries
	cancel()

	// drain remaining so goroutine doesn't block
	for range ch {
	}
}

func TestMockWithError_IsRestartableAfterError(t *testing.T) {
	boom := errors.New("init failed")
	m := NewMockWithError("bad", boom)

	_, _ = m.Start(context.Background())
	// after an error, started should be false, so we can retry
	assert.False(t, m.started.Load())
}
