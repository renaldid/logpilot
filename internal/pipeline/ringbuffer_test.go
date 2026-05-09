package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeEntry(msg string) logentry.LogEntry {
	return logentry.LogEntry{
		Timestamp: time.Now(),
		Service:   "test",
		Level:     logentry.LogLevelInfo,
		Message:   msg,
		Raw:       msg,
	}
}

func TestNewRingBuffer_PanicsOnZeroCapacity(t *testing.T) {
	assert.Panics(t, func() { NewRingBuffer(0) })
}

func TestNewRingBuffer_PanicsOnNegativeCapacity(t *testing.T) {
	assert.Panics(t, func() { NewRingBuffer(-1) })
}

func TestRingBuffer_EmptyBuffer(t *testing.T) {
	rb := NewRingBuffer(10)
	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, 10, rb.Cap())
	assert.Nil(t, rb.Snapshot())
	assert.Equal(t, int64(0), rb.Dropped())
}

func TestRingBuffer_PushAndSnapshot(t *testing.T) {
	rb := NewRingBuffer(5)
	for i := range 3 {
		rb.Push(makeEntry(string(rune('A' + i))))
	}
	snap := rb.Snapshot()
	require.Len(t, snap, 3)
	assert.Equal(t, "A", snap[0].Message)
	assert.Equal(t, "B", snap[1].Message)
	assert.Equal(t, "C", snap[2].Message)
}

func TestRingBuffer_FillToCapacity(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(makeEntry("X"))
	rb.Push(makeEntry("Y"))
	rb.Push(makeEntry("Z"))

	assert.Equal(t, 3, rb.Len())
	assert.Equal(t, int64(0), rb.Dropped())
	snap := rb.Snapshot()
	assert.Equal(t, []string{"X", "Y", "Z"}, messages(snap))
}

func TestRingBuffer_Overflow_DropsOldest(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(makeEntry("A"))
	rb.Push(makeEntry("B"))
	rb.Push(makeEntry("C"))
	rb.Push(makeEntry("D")) // pushes out A
	rb.Push(makeEntry("E")) // pushes out B

	assert.Equal(t, 3, rb.Len())
	assert.Equal(t, int64(2), rb.Dropped())
	snap := rb.Snapshot()
	assert.Equal(t, []string{"C", "D", "E"}, messages(snap))
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Push(makeEntry("X"))
	rb.Push(makeEntry("Y"))
	rb.Clear()
	assert.Equal(t, 0, rb.Len())
	assert.Nil(t, rb.Snapshot())
}

func TestRingBuffer_ClearThenPush(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(makeEntry("A"))
	rb.Push(makeEntry("B"))
	rb.Clear()
	rb.Push(makeEntry("C"))
	snap := rb.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "C", snap[0].Message)
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := NewRingBuffer(100)
	const workers = 10
	const perWorker = 50

	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range perWorker {
				rb.Push(makeEntry(string(rune('A' + id))))
			}
		}(w)
	}
	wg.Wait()

	// all writes have happened — buffer should have exactly 100 entries
	assert.Equal(t, 100, rb.Len())
	total := int64(workers*perWorker) - int64(rb.Len())
	assert.Equal(t, total, rb.Dropped())
}

func TestRingBuffer_SnapshotIsACopy(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Push(makeEntry("original"))
	snap := rb.Snapshot()
	snap[0].Message = "mutated"

	snap2 := rb.Snapshot()
	assert.Equal(t, "original", snap2[0].Message)
}

func messages(entries []logentry.LogEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Message
	}
	return out
}
