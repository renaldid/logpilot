package pipeline

import (
	"sync"
	"sync/atomic"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// RingBuffer is a fixed-capacity, thread-safe circular log buffer.
// When full, the oldest entry is silently dropped to make room for the newest.
type RingBuffer struct {
	mu       sync.RWMutex
	data     []logentry.LogEntry
	head     int   // index of the oldest entry
	size     int   // number of valid entries currently stored
	capacity int
	dropped  atomic.Int64 // cumulative count of dropped entries
}

// NewRingBuffer returns a RingBuffer with the given capacity.
// Panics if capacity < 1.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		panic("ringbuffer: capacity must be >= 1")
	}
	return &RingBuffer{
		data:     make([]logentry.LogEntry, capacity),
		capacity: capacity,
	}
}

// Push adds an entry to the buffer. If the buffer is full, the oldest entry
// is overwritten and the dropped counter is incremented.
func (r *RingBuffer) Push(entry logentry.LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size == r.capacity {
		// overwrite oldest — advance head
		r.data[r.head] = entry
		r.head = (r.head + 1) % r.capacity
		r.dropped.Add(1)
		return
	}

	tail := (r.head + r.size) % r.capacity
	r.data[tail] = entry
	r.size++
}

// Snapshot returns a copy of all entries in insertion order (oldest first).
func (r *RingBuffer) Snapshot() []logentry.LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return nil
	}
	out := make([]logentry.LogEntry, r.size)
	for i := range r.size {
		out[i] = r.data[(r.head+i)%r.capacity]
	}
	return out
}

// Len returns the number of entries currently stored.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// Cap returns the maximum capacity.
func (r *RingBuffer) Cap() int { return r.capacity }

// Dropped returns the cumulative number of entries that were dropped due to overflow.
func (r *RingBuffer) Dropped() int64 { return r.dropped.Load() }

// Clear removes all entries from the buffer.
func (r *RingBuffer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.head = 0
	r.size = 0
}
