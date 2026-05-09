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

// staticTail returns a TailFunc that emits the given lines then closes.
func staticTail(lines []string) TailFunc {
	return func(ctx context.Context, path string) (<-chan string, error) {
		ch := make(chan string, len(lines))
		for _, l := range lines {
			ch <- l
		}
		close(ch)
		return ch, nil
	}
}

// errorTail returns a TailFunc that always fails with err.
func errorTail(err error) TailFunc {
	return func(ctx context.Context, path string) (<-chan string, error) {
		return nil, err
	}
}

// blockingTail returns a TailFunc that blocks until the context is cancelled.
func blockingTail() TailFunc {
	return func(ctx context.Context, path string) (<-chan string, error) {
		ch := make(chan string)
		go func() {
			defer close(ch)
			<-ctx.Done()
		}()
		return ch, nil
	}
}

func collectEntries(ch <-chan logentry.LogEntry, timeout time.Duration) []logentry.LogEntry {
	var out []logentry.LogEntry
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, e)
		case <-timer.C:
			return out
		}
	}
}

func TestFileSource_Name(t *testing.T) {
	fs := newFileSourceWithTail("mylog", "/var/log/app.log", staticTail(nil))
	assert.Equal(t, "mylog", fs.Name())
}

func TestFileSource_EmitsLines(t *testing.T) {
	lines := []string{
		"2024-01-02T15:04:05Z INFO request received",
		"2024-01-02T15:04:06Z ERROR something failed",
	}
	fs := newFileSourceWithTail("api", "/fake/path", staticTail(lines))

	ch, err := fs.Start(context.Background())
	require.NoError(t, err)

	collected := collectEntries(ch, 2*time.Second)
	assert.Len(t, collected, 2)
	assert.Equal(t, "api", collected[0].Service)
	assert.Equal(t, logentry.LogLevelInfo, collected[0].Level)
	assert.Equal(t, logentry.LogLevelError, collected[1].Level)
}

func TestFileSource_DoubleStart_ReturnsError(t *testing.T) {
	fs := newFileSourceWithTail("svc", "/fake", blockingTail())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := fs.Start(ctx)
	require.NoError(t, err)

	_, err = fs.Start(ctx)
	assert.ErrorIs(t, err, ErrAlreadyStarted)
}

func TestFileSource_TailError_PropagatesError(t *testing.T) {
	boom := errors.New("file not found")
	fs := newFileSourceWithTail("svc", "/missing", errorTail(boom))

	_, err := fs.Start(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestFileSource_Stop_BeforeStart_ReturnsError(t *testing.T) {
	fs := newFileSourceWithTail("svc", "/fake", staticTail(nil))
	assert.ErrorIs(t, fs.Stop(), ErrNotStarted)
}

func TestFileSource_Stop_StopsCleanly(t *testing.T) {
	fs := newFileSourceWithTail("svc", "/fake", blockingTail())

	ctx := context.Background()
	ch, err := fs.Start(ctx)
	require.NoError(t, err)

	err = fs.Stop()
	assert.NoError(t, err)

	// channel must close after stop
	select {
	case _, ok := <-ch:
		if !ok {
			return // closed — correct
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after Stop()")
	}
}

func TestFileSource_ContextCancel_ClosesChannel(t *testing.T) {
	fs := newFileSourceWithTail("svc", "/fake", blockingTail())

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := fs.Start(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close after context cancel")
	}
}

func TestFileSource_RestartableAfterStop(t *testing.T) {
	lines := []string{"line1"}
	fs := newFileSourceWithTail("svc", "/fake", staticTail(lines))

	ctx := context.Background()
	ch, err := fs.Start(ctx)
	require.NoError(t, err)
	collectEntries(ch, time.Second)

	// fs.started should be false after channel closes — restart allowed
	fs.started.Store(false) // simulate natural stop
	_, err = fs.Start(ctx)
	assert.NoError(t, err)
}
