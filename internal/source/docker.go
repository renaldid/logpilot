package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// DockerClient abstracts the Docker SDK so DockerSource can be tested without
// a real Docker daemon.
type DockerClient interface {
	ContainerList(ctx context.Context, opts container.ListOptions) ([]container.Summary, error)
	ContainerLogs(ctx context.Context, containerID string, opts container.LogsOptions) (io.ReadCloser, error)
	Close() error
}

// DockerSource streams logs from all containers in a Docker Compose project.
type DockerSource struct {
	name        string
	composeFile string // used as filter label value; empty = all containers
	client      DockerClient

	started atomic.Bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewDockerSource creates a DockerSource connected to the local Docker daemon.
func NewDockerSource(name, composeFile string) (*DockerSource, error) {
	c, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return newDockerSourceWithClient(name, composeFile, c), nil
}

// newDockerSourceWithClient injects a custom DockerClient (for testing).
func newDockerSourceWithClient(name, composeFile string, client DockerClient) *DockerSource {
	return &DockerSource{name: name, composeFile: composeFile, client: client}
}

func (d *DockerSource) Name() string { return d.name }

func (d *DockerSource) Start(ctx context.Context) (<-chan logentry.LogEntry, error) {
	if !d.started.CompareAndSwap(false, true) {
		return nil, ErrAlreadyStarted
	}

	containers, err := d.listContainers(ctx)
	if err != nil {
		d.started.Store(false)
		return nil, fmt.Errorf("list containers: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	out := make(chan logentry.LogEntry, 256)

	for _, c := range containers {
		d.wg.Add(1)
		go d.streamContainer(ctx, c.ID, containerName(c), out)
	}

	go func() {
		d.wg.Wait()
		close(out)
	}()

	return out, nil
}

func (d *DockerSource) Stop() error {
	if !d.started.Load() {
		return ErrNotStarted
	}
	d.cancel()
	d.wg.Wait()
	_ = d.client.Close()
	d.started.Store(false)
	return nil
}

func (d *DockerSource) listContainers(ctx context.Context) ([]container.Summary, error) {
	f := filters.NewArgs()
	if d.composeFile != "" {
		f.Add("label", "com.docker.compose.project")
	}
	return d.client.ContainerList(ctx, container.ListOptions{
		All:     false,
		Filters: f,
	})
}

func (d *DockerSource) streamContainer(ctx context.Context, id, name string, out chan<- logentry.LogEntry) {
	defer d.wg.Done()

	rc, err := d.client.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 8 {
			// Docker multiplexed stream: first 8 bytes are a header
			line = line[8:]
		}
		entry := logentry.Parse(name, line)
		select {
		case <-ctx.Done():
			return
		case out <- entry:
		}
	}
}

func containerName(c container.Summary) string {
	if len(c.Names) > 0 {
		name := c.Names[0]
		if len(name) > 0 && name[0] == '/' {
			return name[1:]
		}
		return name
	}
	return c.ID[:12]
}
