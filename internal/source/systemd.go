//go:build linux

package source

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// JournalctlFunc starts journalctl for the given unit and returns a channel of raw log lines.
// The channel is closed when the context is cancelled or journalctl exits.
type JournalctlFunc func(ctx context.Context, unit string) (<-chan string, error)

// SystemdSource tails a systemd unit journal and emits parsed LogEntry values.
type SystemdSource struct {
	name        string
	unit        string
	journalFunc JournalctlFunc

	started atomic.Bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewSystemdSource returns a SystemdSource that reads from journalctl.
func NewSystemdSource(name, unit string) *SystemdSource {
	return &SystemdSource{name: name, unit: unit, journalFunc: journalctlTail}
}

func newSystemdSourceWithFunc(name, unit string, fn JournalctlFunc) *SystemdSource {
	return &SystemdSource{name: name, unit: unit, journalFunc: fn}
}

func (s *SystemdSource) Name() string { return s.name }

func (s *SystemdSource) Start(ctx context.Context) (<-chan logentry.LogEntry, error) {
	if !s.started.CompareAndSwap(false, true) {
		return nil, ErrAlreadyStarted
	}

	lines, err := s.journalFunc(ctx, s.unit)
	if err != nil {
		s.started.Store(false)
		return nil, fmt.Errorf("journalctl %q: %w", s.unit, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	out := make(chan logentry.LogEntry, 64)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-lines:
				if !ok {
					return
				}
				out <- logentry.Parse(s.name, line)
			}
		}
	}()

	return out, nil
}

func (s *SystemdSource) Stop() error {
	if !s.started.Load() {
		return ErrNotStarted
	}
	s.cancel()
	s.wg.Wait()
	s.started.Store(false)
	return nil
}

// journalctlTail is the production journalctl implementation.
func journalctlTail(ctx context.Context, unit string) (<-chan string, error) {
	cmd := exec.CommandContext(ctx, "journalctl", "-f", "-u", unit, "--output=cat", "--no-pager")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	lines := make(chan string, 128)
	go func() {
		defer close(lines)
		defer cmd.Wait() //nolint:errcheck
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case lines <- scanner.Text():
			}
		}
	}()

	return lines, nil
}
