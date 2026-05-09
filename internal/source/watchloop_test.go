package source

import (
	"bufio"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestWatchLoop_ContextCancel_Exits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan fsnotify.Event)
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader(""))
	lines := make(chan string, 10)

	done := make(chan struct{})
	go func() {
		watchLoop(ctx, events, errs, reader, lines)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchLoop did not exit after context cancel")
	}
}

func TestWatchLoop_EventsChannelClosed_Exits(t *testing.T) {
	events := make(chan fsnotify.Event)
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader(""))
	lines := make(chan string, 10)

	done := make(chan struct{})
	go func() {
		watchLoop(context.Background(), events, errs, reader, lines)
		close(done)
	}()

	close(events) // simulate watcher being closed
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchLoop did not exit when events channel closed")
	}
}

func TestWatchLoop_ErrorChannel_Exits(t *testing.T) {
	events := make(chan fsnotify.Event)
	errs := make(chan error, 1)
	reader := bufio.NewReader(strings.NewReader(""))
	lines := make(chan string, 10)

	done := make(chan struct{})
	go func() {
		watchLoop(context.Background(), events, errs, reader, lines)
		close(done)
	}()

	errs <- assert.AnError // inject a watcher error
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchLoop did not exit on watcher error")
	}
}

func TestWatchLoop_WriteEvent_EmitsLines(t *testing.T) {
	content := "line1\nline2\n"
	events := make(chan fsnotify.Event, 1)
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader(content))
	lines := make(chan string, 10)

	events <- fsnotify.Event{Op: fsnotify.Write, Name: "/tmp/app.log"}
	close(events) // exit after processing one event

	watchLoop(context.Background(), events, errs, reader, lines)

	close(lines)
	var got []string
	for l := range lines {
		got = append(got, l)
	}
	assert.Len(t, got, 2)
}

func TestWatchLoop_CreateEvent_EmitsLines(t *testing.T) {
	content := "hello\n"
	events := make(chan fsnotify.Event, 1)
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader(content))
	lines := make(chan string, 10)

	events <- fsnotify.Event{Op: fsnotify.Create, Name: "/tmp/app.log"}
	close(events)

	watchLoop(context.Background(), events, errs, reader, lines)

	close(lines)
	var got []string
	for l := range lines {
		got = append(got, l)
	}
	assert.NotEmpty(t, got)
}

func TestWatchLoop_NonWriteEvent_DoesNotEmit(t *testing.T) {
	events := make(chan fsnotify.Event, 1)
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader("should-not-appear\n"))
	lines := make(chan string, 10)

	// Chmod event — should not trigger emitNewLines
	events <- fsnotify.Event{Op: fsnotify.Chmod, Name: "/tmp/app.log"}
	close(events)

	watchLoop(context.Background(), events, errs, reader, lines)

	close(lines)
	assert.Empty(t, lines)
}

func TestWatchLoop_TimerPoll_EmitsLines(t *testing.T) {
	// no events — let the 100ms timer fire and pick up content
	content := "polled-line\n"
	events := make(chan fsnotify.Event) // no events written
	errs := make(chan error)
	reader := bufio.NewReader(strings.NewReader(content))
	lines := make(chan string, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	watchLoop(ctx, events, errs, reader, lines)

	close(lines)
	var got []string
	for l := range lines {
		got = append(got, l)
	}
	assert.NotEmpty(t, got)
}
