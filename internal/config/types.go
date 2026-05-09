package config

import "errors"

// SourceType identifies which adapter handles a source.
type SourceType string

const (
	SourceTypeDocker  SourceType = "docker"
	SourceTypeFile    SourceType = "file"
	SourceTypeSystemd SourceType = "systemd"
)

// SourceConfig describes a single log source in the config file.
type SourceConfig struct {
	Name        string     `mapstructure:"name"`
	Type        SourceType `mapstructure:"type"`
	Path        string     `mapstructure:"path"`         // file sources
	ComposeFile string     `mapstructure:"compose_file"` // docker sources
	Unit        string     `mapstructure:"unit"`         // systemd sources
}

// Config is the top-level configuration loaded from .logpilot.yaml.
type Config struct {
	BufferSize int                `mapstructure:"buffer_size"`
	Follow     bool               `mapstructure:"follow"`
	Sources    []SourceConfig     `mapstructure:"sources"`
	Colors     map[string]string  `mapstructure:"colors"`
}

// Validate returns an error if any required field is missing or invalid.
func (c *Config) Validate() error {
	if c.BufferSize < 1 {
		return errors.New("buffer_size must be >= 1")
	}
	for i, s := range c.Sources {
		if s.Name == "" {
			return errors.New("each source must have a name")
		}
		switch s.Type {
		case SourceTypeDocker, SourceTypeFile, SourceTypeSystemd:
		default:
			return errors.New("source[" + string(rune('0'+i)) + "]: unknown type " + string(s.Type))
		}
		if s.Type == SourceTypeFile && s.Path == "" {
			return errors.New("file source " + s.Name + " requires a path")
		}
		if s.Type == SourceTypeSystemd && s.Unit == "" {
			return errors.New("systemd source " + s.Name + " requires a unit")
		}
	}
	return nil
}
