//go:build !linux

package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/renaldid/logpilot/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildSources_SystemdSource_Skips(t *testing.T) {
	cfg := &config.Config{Sources: []config.SourceConfig{
		{Name: "sshd", Type: config.SourceTypeSystemd, Unit: "sshd.service"},
	}}

	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w

	sources, err := buildSources(cfg)

	w.Close()
	os.Stderr = orig
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
	assert.Empty(t, sources)
	assert.Contains(t, buf.String(), "skipping")
}
