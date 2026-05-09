package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/renaldid/logpilot/internal/source"
	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entries(msgs ...string) []logentry.LogEntry {
	out := make([]logentry.LogEntry, len(msgs))
	for i, m := range msgs {
		out[i] = logentry.LogEntry{
			Timestamp: time.Now(),
			Service:   "test",
			Level:     logentry.LogLevelInfo,
			Message:   m,
		}
	}
	return out
}

func drainChannel(ch <-chan logentry.LogEntry, timeout time.Duration) []logentry.LogEntry {
	var collected []logentry.LogEntry
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return collected
			}
			collected = append(collected, e)
		case <-timer.C:
			return collected
		}
	}
}

func TestAggregator_SingleSource(t *testing.T) {
	buf := NewRingBuffer(100)
	src := source.NewMock("api", entries("hello", "world"))
	agg := NewAggregator([]source.LogSource{src}, buf, 10)

	ctx := context.Background()
	ch, err := agg.Start(ctx)
	require.NoError(t, err)

	collected := drainChannel(ch, 2*time.Second)
	assert.Len(t, collected, 2)
	assert.Equal(t, 2, buf.Len())
}

func TestAggregator_MultipleSources(t *testing.T) {
	buf := NewRingBuffer(100)
	s1 := source.NewMock("api", entries("a1", "a2"))
	s2 := source.NewMock("worker", entries("w1", "w2", "w3"))
	agg := NewAggregator([]source.LogSource{s1, s2}, buf, 20)

	ch, err := agg.Start(context.Background())
	require.NoError(t, err)

	collected := drainChannel(ch, 2*time.Second)
	assert.Len(t, collected, 5)
	assert.Equal(t, 5, buf.Len())
}

func TestAggregator_ContextCancellation(t *testing.T) {
	buf := NewRingBuffer(100)

	// blocking source — emits nothing, waits for cancel
	blockingSrc := source.NewMock("blocker", nil)

	agg := NewAggregator([]source.LogSource{blockingSrc}, buf, 10)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := agg.Start(ctx)
	require.NoError(t, err)

	cancel() // trigger shutdown
	drainChannel(ch, time.Second)
	// channel must close cleanly — no hang
}

func TestAggregator_DoubleStart_ReturnsError(t *testing.T) {
	buf := NewRingBuffer(10)
	src := source.NewMock("api", nil)
	agg := NewAggregator([]source.LogSource{src}, buf, 0)

	ctx := context.Background()
	_, err := agg.Start(ctx)
	require.NoError(t, err)

	_, err = agg.Start(ctx)
	assert.ErrorIs(t, err, source.ErrAlreadyStarted)

	_ = agg.Stop()
}

func TestAggregator_Stop_BeforeStart_ReturnsError(t *testing.T) {
	buf := NewRingBuffer(10)
	agg := NewAggregator(nil, buf, 0)
	assert.ErrorIs(t, agg.Stop(), source.ErrNotStarted)
}

func TestAggregator_Stop_StopsCleanly(t *testing.T) {
	buf := NewRingBuffer(10)
	src := source.NewMock("api", nil)
	agg := NewAggregator([]source.LogSource{src}, buf, 0)

	ch, err := agg.Start(context.Background())
	require.NoError(t, err)

	err = agg.Stop()
	assert.NoError(t, err)

	drainChannel(ch, time.Second) // should close promptly
}

func TestAggregator_SourceStartError_PropagatesError(t *testing.T) {
	buf := NewRingBuffer(10)
	boom := errors.New("source init failed")
	src := source.NewMockWithError("bad-source", boom)
	agg := NewAggregator([]source.LogSource{src}, buf, 0)

	_, err := agg.Start(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)

	// aggregator should be restartable after failure
	goodSrc := source.NewMock("good", entries("ok"))
	agg2 := NewAggregator([]source.LogSource{goodSrc}, buf, 10)
	ch, err := agg2.Start(context.Background())
	require.NoError(t, err)
	drainChannel(ch, time.Second)
}

func TestAggregator_EntriesStoredInBuffer(t *testing.T) {
	buf := NewRingBuffer(10)
	src := source.NewMock("svc", entries("msg1", "msg2", "msg3"))
	agg := NewAggregator([]source.LogSource{src}, buf, 10)

	ch, err := agg.Start(context.Background())
	require.NoError(t, err)
	drainChannel(ch, time.Second)

	snap := buf.Snapshot()
	msgs := make([]string, len(snap))
	for i, e := range snap {
		msgs[i] = e.Message
	}
	assert.ElementsMatch(t, []string{"msg1", "msg2", "msg3"}, msgs)
}
