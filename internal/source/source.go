package source

import (
	"context"
	"errors"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// ErrAlreadyStarted is returned when Start is called on a running source.
var ErrAlreadyStarted = errors.New("source already started")

// ErrNotStarted is returned when Stop is called on a source that was never started.
var ErrNotStarted = errors.New("source not started")

// LogSource is the interface every log source adapter must implement.
// A source owns a goroutine that emits LogEntry values until the context
// is cancelled or an unrecoverable error occurs.
type LogSource interface {
	// Name returns a human-readable identifier used in log entries and the sidebar.
	Name() string

	// Start begins streaming log entries into the returned channel.
	// The channel is closed when the source stops (context cancel or error).
	// Calling Start on an already-running source returns ErrAlreadyStarted.
	Start(ctx context.Context) (<-chan logentry.LogEntry, error)

	// Stop signals the source to shut down and waits for its goroutine to exit.
	// Returns ErrNotStarted if the source was never started.
	Stop() error
}
