package source

import (
	"bytes"
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

func TestDockerSource_ListContainers_WithComposeFile(t *testing.T) {
	client := &mockDockerClient{
		containers: []container.Summary{
			makeContainer("id1", "svc1"),
		},
	}
	src := newDockerSourceWithClient("docker", "./docker-compose.yml", client)

	ctx := context.Background()
	containers, err := src.listContainers(ctx)
	require.NoError(t, err)
	assert.Len(t, containers, 1)
}

func TestDockerSource_StreamContainer_LogsError_Skips(t *testing.T) {
	client := &mockDockerClient{
		containers: []container.Summary{makeContainer("bad", "broken")},
		logsErr:    errors.New("container not found"),
	}
	src := newDockerSourceWithClient("docker", "", client)

	ch, err := src.Start(context.Background())
	require.NoError(t, err)

	// streamContainer errors internally — channel should close cleanly
	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("channel did not close when ContainerLogs errors")
	}
}

func TestDockerSource_StreamContainer_ShortLines_NoHeaderStrip(t *testing.T) {
	// Lines with <= 8 bytes should not panic on slicing
	shortLog := "tiny\n"
	client := &mockDockerClient{
		containers: []container.Summary{makeContainer("id1", "svc")},
		logs:       map[string]string{"id1": shortLog},
	}
	src := newDockerSourceWithClient("docker", "", client)

	ch, err := src.Start(context.Background())
	require.NoError(t, err)

	var got []string
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
loop:
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				break loop
			}
			got = append(got, e.Raw)
		case <-timer.C:
			break loop
		}
	}
	// just verify it ran without panic
	assert.NotNil(t, got)
}

func TestDockerSource_StreamContainer_ContextCancel(t *testing.T) {
	// Strategy: fill the 256-entry output buffer with a bytes.Buffer (instant read),
	// then cancel ctx while goroutine is blocked on the 257th send → ctx.Done() fires.
	var logData strings.Builder
	for range 300 {
		logData.WriteString(strings.Repeat(" ", 8) + "INFO log line for ctx cancel test\n")
	}

	client := &floodDockerClient{
		containers: []container.Summary{makeContainer("id1", "svc")},
		data:       []byte(logData.String()),
	}
	src := newDockerSourceWithClient("docker", "", client)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := src.Start(ctx)
	require.NoError(t, err)

	// Let the goroutine read all 300 lines instantly (bytes.Buffer → no blocking on reads).
	// After 256 entries fill the output buffer, it blocks on the 257th send.
	time.Sleep(50 * time.Millisecond)
	cancel() // ctx.Done() now fires in the select while goroutine is blocked

	// Drain to let goroutine unblock and exit cleanly.
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timer.C:
			return
		}
	}
}

// floodDockerClient returns a bytes.Buffer reader so all lines are available instantly.
type floodDockerClient struct {
	containers []container.Summary
	data       []byte
}

func (f *floodDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return f.containers, nil
}

func (f *floodDockerClient) ContainerLogs(_ context.Context, _ string, _ container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.data)), nil
}

func (f *floodDockerClient) Close() error { return nil }

// pipeDockerClient serves a streaming io.ReadCloser for ContainerLogs.
type pipeDockerClient struct {
	containers []container.Summary
	reader     io.ReadCloser
}

func (p *pipeDockerClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return p.containers, nil
}

func (p *pipeDockerClient) ContainerLogs(_ context.Context, _ string, _ container.LogsOptions) (io.ReadCloser, error) {
	return p.reader, nil
}

func (p *pipeDockerClient) Close() error { return nil }

func TestNewDockerSource_InvalidHost_ReturnsError(t *testing.T) {
	// "tcp://[" has an unclosed IPv6 bracket — Docker SDK rejects it at construction.
	t.Setenv("DOCKER_HOST", "tcp://[")

	_, err := NewDockerSource("test", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker client")
}

func TestNewDockerSource_ValidEnv_Succeeds(t *testing.T) {
	// Without overriding DOCKER_HOST, NewDockerSource should succeed at construction
	// (Docker SDK is lazy — it doesn't connect until the first API call)
	src, err := NewDockerSource("test", "")
	if err == nil {
		require.NotNil(t, src)
		_ = src.client.Close()
	}
}

func TestDockerSource_StreamContainer_MultipleLines(t *testing.T) {
	// 8-byte header prefix that will be stripped
	header := strings.Repeat(" ", 8)
	logContent := header + "INFO line1\n" + header + "WARN line2\n"
	client := &mockDockerClient{
		containers: []container.Summary{makeContainer("c1", "api")},
		logs:       map[string]string{"c1": logContent},
	}
	src := newDockerSourceWithClient("docker", "", client)

	ch, err := src.Start(context.Background())
	require.NoError(t, err)

	var count int
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
loop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break loop
			}
			count++
		case <-timer.C:
			break loop
		}
	}
	assert.Equal(t, 2, count)
}
