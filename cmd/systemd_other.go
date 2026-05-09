//go:build !linux

package cmd

import (
	"fmt"
	"os"

	"github.com/renaldid/logpilot/internal/source"
)

func newSystemdSource(name, _ string) (source.LogSource, error) {
	fmt.Fprintf(os.Stderr, "warning: systemd source %q not supported on this platform, skipping\n", name)
	return nil, nil
}
