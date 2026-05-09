//go:build linux

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

func staticJournalctl(lines []string) JournalctlFunc {
	return func(_ context.Context, _ string) (<-chan string, error) {
		ch := make(chan string, len(lines))
		for _, l := range lines {
			ch <- l
		}
		close(ch)
		return ch, nil
	}
}

func errorJournalctl(err error) JournalctlFunc {
	return func(_ context.Context, _ string) (<-chan string, error) {
		return nil, err
	}
}

func blockingJournalctl() JournalctlFunc {
	return func(ctx context.Context, _ string) (<-chan string, error) {
		ch := make(chan string)
		go func() {
			defer close(ch)
			<-ctx.Done()
		}()
		return ch, nil
	}
}

func TestSystemdSource_Name(t *testing.T) {
	s := NewSystemdSource("nginx", "nginx.service")
	assert.Equal(t, "nginx", s.Name())
}

func TestSystemdSource_EmitsLines(t *testing.T) {
	lines := []string{
		"2024-01-02T15:04:05Z INFO started nginx",
		"2024-01-02T15:04:06Z ERROR connection refused",
	}
	s := newSystemdSourceWithFunc("nginx", "nginx.service", staticJournalctl(lines))

	ch, err := s.Start(context.Background())
	require.NoError(t, err)

	collected := collectEntries(ch, 2*time.Second)
	assert.Len(t, collected, 2)
	assert.Equal(t, "nginx", collected[0].Service)
	assert.Equal(t, logentry.LogLevelInfo, collected[0].Level)
	assert.Equal(t, logentry.LogLevelError, collected[1].Level)
}

func TestSystemdSource_DoubleStart_ReturnsError(t *testing.T) {
	s := newSystemdSourceWithFunc("svc", "svc.service", blockingJournalctl())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := s.Start(ctx)
	require.NoError(t, err)

	_, err = s.Start(ctx)
	assert.ErrorIs(t, err, ErrAlreadyStarted)
}

func TestSystemdSource_JournalctlError_PropagatesError(t *testing.T) {
	boom := errors.New("journalctl not found")
	s := newSystemdSourceWithFunc("svc", "svc.service", errorJournalctl(boom))

	_, err := s.Start(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestSystemdSource_Stop_BeforeStart_ReturnsError(t *testing.T) {
	s := newSystemdSourceWithFunc("svc", "svc.service", staticJournalctl(nil))
	assert.ErrorIs(t, s.Stop(), ErrNotStarted)
}

func TestSystemdSource_Stop_StopsCleanly(t *testing.T) {
	s := newSystemdSourceWithFunc("svc", "svc.service", blockingJournalctl())

	ch, err := s.Start(context.Background())
	require.NoError(t, err)

	assert.NoError(t, s.Stop())

	select {
	case _, ok := <-ch:
		if !ok {
			return
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after Stop()")
	}
}

func TestSystemdSource_ContextCancel_ClosesChannel(t *testing.T) {
	s := newSystemdSourceWithFunc("svc", "svc.service", blockingJournalctl())

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.Start(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after context cancel")
	}
}

func TestSystemdSource_RestartableAfterStop(t *testing.T) {
	lines := []string{"line1"}
	s := newSystemdSourceWithFunc("svc", "svc.service", staticJournalctl(lines))

	ctx := context.Background()
	ch, err := s.Start(ctx)
	require.NoError(t, err)
	collectEntries(ch, time.Second)

	s.started.Store(false)
	_, err = s.Start(ctx)
	assert.NoError(t, err)
}
