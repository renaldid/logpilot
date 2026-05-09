package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/renaldid/logpilot/internal/source"
	"github.com/renaldid/logpilot/pkg/logentry"
)

// Aggregator fans in log entries from multiple sources into a single channel
// and simultaneously pushes every entry into a RingBuffer.
type Aggregator struct {
	sources []source.LogSource
	buf     *RingBuffer

	out    chan logentry.LogEntry
	cancel context.CancelFunc
	wg     sync.WaitGroup

	started atomic.Bool
}

// NewAggregator creates an Aggregator backed by the given sources and buffer.
// The output channel has a buffer of outputBuf entries; set to 0 for unbuffered.
func NewAggregator(sources []source.LogSource, buf *RingBuffer, outputBuf int) *Aggregator {
	return &Aggregator{
		sources: sources,
		buf:     buf,
		out:     make(chan logentry.LogEntry, outputBuf),
	}
}

// Start begins reading from all sources concurrently.
// Returns ErrAlreadyStarted if the aggregator is already running.
// The returned channel receives every entry emitted by any source and is
// closed once all sources have stopped.
func (a *Aggregator) Start(ctx context.Context) (<-chan logentry.LogEntry, error) {
	if !a.started.CompareAndSwap(false, true) {
		return nil, source.ErrAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	for _, s := range a.sources {
		ch, err := s.Start(ctx)
		if err != nil {
			// cancel all already-started sources and clean up
			cancel()
			a.drainRemaining()
			a.started.Store(false)
			return nil, fmt.Errorf("start source %q: %w", s.Name(), err)
		}
		a.wg.Add(1)
		go a.fanIn(ch)
	}

	// close output when all fan-in goroutines finish
	go func() {
		a.wg.Wait()
		close(a.out)
	}()

	return a.out, nil
}

// Stop signals all sources to stop and waits for the aggregator to drain.
func (a *Aggregator) Stop() error {
	if !a.started.Load() {
		return source.ErrNotStarted
	}
	a.cancel()
	for _, s := range a.sources {
		_ = s.Stop()
	}
	a.wg.Wait()
	a.started.Store(false)
	return nil
}

func (a *Aggregator) fanIn(ch <-chan logentry.LogEntry) {
	defer a.wg.Done()
	for entry := range ch {
		a.buf.Push(entry)
		a.out <- entry
	}
}

// drainRemaining blocks until all in-flight fan-in goroutines finish.
// Used during error recovery in Start.
func (a *Aggregator) drainRemaining() {
	a.wg.Wait()
}
