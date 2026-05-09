package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/renaldid/logpilot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".logpilot.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestRunRoot_WithValidConfig(t *testing.T) {
	mockTUI(t)
	yaml := `
buffer_size: 1000
follow: true
sources: []
`
	path := writeConfig(t, yaml)

	root := NewRootCmd()
	root.SetArgs([]string{"--config", path})

	err := root.Execute()
	assert.NoError(t, err)
}

func TestRunRoot_InvalidConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".logpilot.yaml")
	// tab indent = invalid YAML
	require.NoError(t, os.WriteFile(path, []byte("key:\n\t bad: indentation"), 0o644))

	root := NewRootCmd()
	root.SetArgs([]string{"--config", path})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	assert.Error(t, err)
}

func TestRunRoot_FollowFlag_Overrides(t *testing.T) {
	mockTUI(t)
	yaml := `
buffer_size: 1000
follow: true
sources: []
`
	path := writeConfig(t, yaml)

	root := NewRootCmd()
	root.SetArgs([]string{"--config", path, "--follow=false"})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	assert.NoError(t, err)
}

// — buildSources unit tests —

func TestBuildSources_Empty(t *testing.T) {
	cfg := &config.Config{Sources: nil}
	sources, err := buildSources(cfg)
	assert.NoError(t, err)
	assert.Empty(t, sources)
}

func TestBuildSources_FileSource(t *testing.T) {
	cfg := &config.Config{Sources: []config.SourceConfig{
		{Name: "logs", Type: config.SourceTypeFile, Path: "/var/log/app.log"},
	}}
	sources, err := buildSources(cfg)
	assert.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "logs", sources[0].Name())
}

func TestBuildSources_SystemdSource_Skips(t *testing.T) {
	cfg := &config.Config{Sources: []config.SourceConfig{
		{Name: "sshd", Type: config.SourceTypeSystemd},
	}}
	var buf bytes.Buffer
	orig := os.Stderr
	// redirect Stderr to capture the warning
	r, w, _ := os.Pipe()
	os.Stderr = w

	sources, err := buildSources(cfg)

	w.Close()
	os.Stderr = orig
	buf.ReadFrom(r)

	assert.NoError(t, err)
	assert.Empty(t, sources)
	assert.Contains(t, buf.String(), "skipping")
}

func TestBuildSources_DockerSource_ConstructionError(t *testing.T) {
	// Malformed URL causes NewClientWithOpts to fail at construction time.
	t.Setenv("DOCKER_HOST", "tcp://[")
	cfg := &config.Config{Sources: []config.SourceConfig{
		{Name: "app", Type: config.SourceTypeDocker, ComposeFile: "docker-compose.yml"},
	}}
	_, err := buildSources(cfg)
	assert.Error(t, err)
}

// TestRunRoot_BuildSourcesError exercises the runRoot error path when
// buildSources fails (docker source with invalid DOCKER_HOST).
func TestRunRoot_BuildSourcesError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "tcp://[")
	yaml := `
buffer_size: 1000
sources:
  - name: app
    type: docker
    compose_file: docker-compose.yml
`
	path := writeConfig(t, yaml)
	root := NewRootCmd()
	root.SetArgs([]string{"--config", path})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	assert.Error(t, err)
}

// TestRunRoot_AggregatorStartError exercises the agg.Start error path.
// The Docker source is created successfully (lazy client), but Start fails
// when ContainerList can't reach the non-existent socket.
func TestRunRoot_AggregatorStartError(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///tmp/logpilot-test-no-such-socket.sock")
	yaml := `
buffer_size: 1000
sources:
  - name: app
    type: docker
    compose_file: docker-compose.yml
`
	path := writeConfig(t, yaml)
	root := NewRootCmd()
	root.SetArgs([]string{"--config", path})
	root.SetErr(&bytes.Buffer{})

	err := root.Execute()
	assert.Error(t, err)
}
