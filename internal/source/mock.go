package source

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// MockSource is a controllable LogSource for use in tests.
// It emits a fixed list of entries then closes, or blocks until stopped.
type MockSource struct {
	name    string
	entries []logentry.LogEntry
	err     error // returned by Start if non-nil

	started atomic.Bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewMock returns a MockSource that emits entries then closes.
func NewMock(name string, entries []logentry.LogEntry) *MockSource {
	return &MockSource{name: name, entries: entries}
}

// NewMockWithError returns a MockSource whose Start returns err.
func NewMockWithError(name string, err error) *MockSource {
	return &MockSource{name: name, err: err}
}

func (m *MockSource) Name() string { return m.name }

func (m *MockSource) Start(ctx context.Context) (<-chan logentry.LogEntry, error) {
	if !m.started.CompareAndSwap(false, true) {
		return nil, ErrAlreadyStarted
	}
	if m.err != nil {
		m.started.Store(false)
		return nil, m.err
	}

	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	ch := make(chan logentry.LogEntry)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(ch)
		for _, e := range m.entries {
			select {
			case <-ctx.Done():
				return
			case ch <- e:
			}
		}
	}()
	return ch, nil
}

func (m *MockSource) Stop() error {
	if !m.started.Load() {
		return ErrNotStarted
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.started.Store(false)
	return nil
}
