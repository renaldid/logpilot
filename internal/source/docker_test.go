package source

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDockerClient implements DockerClient for testing.
type mockDockerClient struct {
	containers []container.Summary
	logs       map[string]string // containerID → log content
	listErr    error
	logsErr    error
	closed     bool
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.containers, nil
}

func (m *mockDockerClient) ContainerLogs(_ context.Context, id string, _ container.LogsOptions) (io.ReadCloser, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	content := m.logs[id]
	return io.NopCloser(strings.NewReader(content)), nil
}

func (m *mockDockerClient) Close() error {
	m.closed = true
	return nil
}

func makeContainer(id, name string) container.Summary {
	return container.Summary{
		ID:    id,
		Names: []string{"/" + name},
	}
}

func TestDockerSource_Name(t *testing.T) {
	src := newDockerSourceWithClient("docker", "", &mockDockerClient{})
	assert.Equal(t, "docker", src.Name())
}

func TestDockerSource_Start_ListError(t *testing.T) {
	boom := errors.New("daemon not running")
	client := &mockDockerClient{listErr: boom}
	src := newDockerSourceWithClient("docker", "", client)

	_, err := src.Start(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestDockerSource_Start_NoContainers(t *testing.T) {
	client := &mockDockerClient{}
	src := newDockerSourceWithClient("docker", "", client)

	ch, err := src.Start(context.Background())
	require.NoError(t, err)

	// channel must close immediately with no containers
	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("channel did not close")
	}
}

func TestDockerSource_Start_EmitsLogs(t *testing.T) {
	// Docker multiplexed format: 8-byte header + content
	// We'll include the 8-byte header prefix in the mock log content
	logContent := "        2024-01-02T15:04:05Z INFO request received\n"
	client := &mockDockerClient{
		containers: []container.Summary{makeContainer("abc123", "api")},
		logs:       map[string]string{"abc123": logContent},
	}
	src := newDockerSourceWithClient("docker", "", client)

	ch, err := src.Start(context.Background())
	require.NoError(t, err)

	var entries []string
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
loop:
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				break loop
			}
			entries = append(entries, e.Service)
		case <-timer.C:
			break loop
		}
	}
	assert.NotEmpty(t, entries)
}

func TestDockerSource_DoubleStart_ReturnsError(t *testing.T) {
	client := &mockDockerClient{}
	src := newDockerSourceWithClient("docker", "", client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := src.Start(ctx)
	require.NoError(t, err)

	_, err = src.Start(ctx)
	assert.ErrorIs(t, err, ErrAlreadyStarted)
	_ = src.Stop()
}

func TestDockerSource_Stop_BeforeStart_ReturnsError(t *testing.T) {
	client := &mockDockerClient{}
	src := newDockerSourceWithClient("docker", "", client)
	assert.ErrorIs(t, src.Stop(), ErrNotStarted)
}

func TestDockerSource_Stop_ClosesClient(t *testing.T) {
	client := &mockDockerClient{}
	src := newDockerSourceWithClient("docker", "", client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := src.Start(ctx)
	require.NoError(t, err)

	err = src.Stop()
	require.NoError(t, err)
	assert.True(t, client.closed)
}

func TestContainerName_WithSlashPrefix(t *testing.T) {
	c := container.Summary{Names: []string{"/my-service"}}
	assert.Equal(t, "my-service", containerName(c))
}

func TestContainerName_WithoutSlash(t *testing.T) {
	c := container.Summary{Names: []string{"my-service"}}
	assert.Equal(t, "my-service", containerName(c))
}

func TestContainerName_NoNames_UsesShortID(t *testing.T) {
	c := container.Summary{ID: "abcdef123456789"}
	assert.Equal(t, "abcdef123456", containerName(c))
}
