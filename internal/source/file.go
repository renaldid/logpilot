package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// TailFunc is a factory that creates a channel of raw log lines for a given path.
// The channel is closed when the context is cancelled or an unrecoverable error occurs.
// Injecting this function makes FileSource fully testable without a real filesystem.
type TailFunc func(ctx context.Context, path string) (<-chan string, error)

// FileSource tails a log file and emits parsed LogEntry values.
type FileSource struct {
	name     string
	path     string
	tailFunc TailFunc

	started atomic.Bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewFileSource returns a FileSource using the production fsnotify-backed tail.
func NewFileSource(name, path string) *FileSource {
	return &FileSource{name: name, path: path, tailFunc: fsTail}
}

// newFileSourceWithTail returns a FileSource with an injected tail function (for testing).
func newFileSourceWithTail(name, path string, tail TailFunc) *FileSource {
	return &FileSource{name: name, path: path, tailFunc: tail}
}

func (f *FileSource) Name() string { return f.name }

func (f *FileSource) Start(ctx context.Context) (<-chan logentry.LogEntry, error) {
	if !f.started.CompareAndSwap(false, true) {
		return nil, ErrAlreadyStarted
	}

	lines, err := f.tailFunc(ctx, f.path)
	if err != nil {
		f.started.Store(false)
		return nil, fmt.Errorf("tail %q: %w", f.path, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	f.cancel = cancel
	out := make(chan logentry.LogEntry, 64)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-lines:
				if !ok {
					return
				}
				out <- logentry.Parse(f.name, line)
			}
		}
	}()

	return out, nil
}

func (f *FileSource) Stop() error {
	if !f.started.Load() {
		return ErrNotStarted
	}
	f.cancel()
	f.wg.Wait()
	f.started.Store(false)
	return nil
}

// fsTail is the production tail implementation using fsnotify.
func fsTail(ctx context.Context, path string) (<-chan string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// seek to end so we only emit new lines
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		file.Close()
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		file.Close()
		return nil, err
	}

	if err := watcher.Add(path); err != nil {
		file.Close()
		watcher.Close()
		return nil, err
	}

	lines := make(chan string, 128)
	reader := bufio.NewReader(file)

	go func() {
		defer close(lines)
		defer watcher.Close()
		defer file.Close()
		watchLoop(ctx, watcher.Events, watcher.Errors, reader, lines)
	}()

	return lines, nil
}

// watchLoop is the inner select loop for fsTail. Extracted for testability.
func watchLoop(
	ctx context.Context,
	events <-chan fsnotify.Event,
	errs <-chan error,
	reader *bufio.Reader,
	lines chan<- string,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				emitNewLines(ctx, reader, lines)
			}
		case <-errs:
			return
		case <-time.After(100 * time.Millisecond):
			// poll for any buffered content when events are delayed
			emitNewLines(ctx, reader, lines)
		}
	}
}

func emitNewLines(ctx context.Context, r *bufio.Reader, out chan<- string) {
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			select {
			case <-ctx.Done():
				return
			case out <- line:
			}
		}
		if err != nil {
			return
		}
	}
}
