//go:build linux

package cmd

import (
	"testing"

	"github.com/renaldid/logpilot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSources_SystemdSource_Creates(t *testing.T) {
	cfg := &config.Config{Sources: []config.SourceConfig{
		{Name: "sshd", Type: config.SourceTypeSystemd, Unit: "sshd.service"},
	}}
	sources, err := buildSources(cfg)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "sshd", sources[0].Name())
}
