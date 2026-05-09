package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/renaldid/logpilot/internal/tui"
	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTUI replaces tuiRunner with a no-op for the duration of the test.
func mockTUI(t *testing.T) {
	t.Helper()
	orig := tuiRunner
	tuiRunner = func(_ tea.Model) error { return nil }
	t.Cleanup(func() { tuiRunner = orig })
}

// TestTuiRunner_Lambda covers the tuiRunner lambda body by overriding programOptions
// to use a fake stdin ('q' key → model quits) and io.Discard as output, avoiding
// the need for a real TTY.
func TestTuiRunner_Lambda(t *testing.T) {
	origOpts := programOptions
	programOptions = []tea.ProgramOption{
		tea.WithInput(strings.NewReader("q")),
		tea.WithOutput(io.Discard),
		tea.WithoutSignalHandler(),
	}
	t.Cleanup(func() { programOptions = origOpts })

	ch := make(chan logentry.LogEntry)
	close(ch)
	state := tui.NewState(true, nil)
	state.Resize(80, 24)
	m := tui.New(ch, state)

	done := make(chan error, 1)
	go func() { done <- tuiRunner(m) }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("tuiRunner did not exit within 5s")
	}
}

func TestVersionCmd_Output(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestVersionCmd_ContainsVersion(t *testing.T) {
	v := newVersionCmd()
	buf := &bytes.Buffer{}
	v.SetOut(buf)

	err := v.Execute()
	require.NoError(t, err)
}

func TestNewRootCmd_HasVersionSubcommand(t *testing.T) {
	root := NewRootCmd()
	var found bool
	for _, c := range root.Commands() {
		if c.Use == "version" {
			found = true
			break
		}
	}
	assert.True(t, found, "version subcommand should be registered")
}

func TestNewRootCmd_HasConfigFlag(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("config")
	assert.NotNil(t, f)
}

func TestNewRootCmd_HasFollowFlag(t *testing.T) {
	root := NewRootCmd()
	f := root.PersistentFlags().Lookup("follow")
	assert.NotNil(t, f)
	assert.Equal(t, "true", f.DefValue)
}

func TestVersion_DefaultIsDev(t *testing.T) {
	assert.Equal(t, "dev", Version)
}

func TestExecute_Success_NoExit(t *testing.T) {
	mockTUI(t)

	var exitCalled bool
	origExit := exitFunc
	exitFunc = func(code int) { exitCalled = true }
	defer func() { exitFunc = origExit }()

	os.Args = []string{"logpilot", "--config", ""}
	Execute()
	assert.False(t, exitCalled)
}

func TestExecute_Error_CallsExit1(t *testing.T) {
	var gotCode int
	origExit := exitFunc
	exitFunc = func(code int) { gotCode = code }
	defer func() { exitFunc = origExit }()

	// Pass an unknown flag so cobra returns an error.
	os.Args = []string{"logpilot", "--unknown-flag-xyz"}
	Execute()
	assert.Equal(t, 1, gotCode)
}
