//go:build linux

package cmd

import "github.com/renaldid/logpilot/internal/source"

func newSystemdSource(name, unit string) (source.LogSource, error) {
	return source.NewSystemdSource(name, unit), nil
}
