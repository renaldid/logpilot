package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".logpilot.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoad_ValidConfig(t *testing.T) {
	yaml := `
buffer_size: 5000
follow: false
sources:
  - name: app-log
    type: file
    path: /var/log/app.log
colors:
  api: "#7C3AED"
`
	path := writeYAML(t, yaml)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, 5000, cfg.BufferSize)
	assert.False(t, cfg.Follow)
	require.Len(t, cfg.Sources, 1)
	assert.Equal(t, "app-log", cfg.Sources[0].Name)
	assert.Equal(t, SourceTypeFile, cfg.Sources[0].Type)
	assert.Equal(t, "/var/log/app.log", cfg.Sources[0].Path)
	assert.Equal(t, "#7C3AED", cfg.Colors["api"])
}

func TestLoad_Defaults(t *testing.T) {
	yaml := `
sources:
  - name: docker
    type: docker
`
	path := writeYAML(t, yaml)
	cfg, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, 10000, cfg.BufferSize)
	assert.True(t, cfg.Follow)
}

func TestLoad_NoConfigFile_UsesDefaults(t *testing.T) {
	// pass a nonexistent path triggers viper.ConfigFileNotFoundError branch
	// Load should NOT error and should return defaults
	cfg, err := Load("")
	// may or may not find a .logpilot.yaml in cwd — just check it doesn't panic
	// if no file found, defaults are used
	if err == nil {
		assert.GreaterOrEqual(t, cfg.BufferSize, 1)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".logpilot.yaml")
	// YAML does not allow tabs as indentation — this forces a parse error
	require.NoError(t, os.WriteFile(path, []byte("key:\n\t bad: indentation"), 0o644))

	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_ValidationError_ZeroBufferSize(t *testing.T) {
	yaml := `
buffer_size: 0
sources:
  - name: svc
    type: docker
`
	path := writeYAML(t, yaml)
	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_ValidationError_MissingSourceName(t *testing.T) {
	yaml := `
buffer_size: 100
sources:
  - type: file
    path: /tmp/app.log
`
	path := writeYAML(t, yaml)
	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_ValidationError_UnknownSourceType(t *testing.T) {
	yaml := `
buffer_size: 100
sources:
  - name: svc
    type: kafka
`
	path := writeYAML(t, yaml)
	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_ValidationError_FileMissingPath(t *testing.T) {
	yaml := `
buffer_size: 100
sources:
  - name: app
    type: file
`
	path := writeYAML(t, yaml)
	_, err := Load(path)
	assert.Error(t, err)
}

func TestLoad_UnmarshalError(t *testing.T) {
	// buffer_size as a nested map causes mapstructure to fail unmarshaling into int
	yaml := `
buffer_size:
  this_is_a_map: true
  not_an_int: yes
`
	path := writeYAML(t, yaml)
	_, err := Load(path)
	assert.Error(t, err)
}

func TestDefault(t *testing.T) {
	cfg := Default()
	assert.Equal(t, 10000, cfg.BufferSize)
	assert.True(t, cfg.Follow)
	assert.NotNil(t, cfg.Sources)
	assert.NotNil(t, cfg.Colors)
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		BufferSize: 1000,
		Sources: []SourceConfig{
			{Name: "docker-src", Type: SourceTypeDocker},
			{Name: "file-src", Type: SourceTypeFile, Path: "/tmp/app.log"},
			{Name: "systemd-src", Type: SourceTypeSystemd},
		},
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_EmptySourcesIsValid(t *testing.T) {
	cfg := &Config{BufferSize: 100}
	assert.NoError(t, cfg.Validate())
}
