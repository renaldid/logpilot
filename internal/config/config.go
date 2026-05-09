package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// defaults applied when a key is absent from the config file.
var defaults = map[string]any{
	"buffer_size": 10000,
	"follow":      true,
}

// Load reads the config file at path and returns a validated Config.
// If path is empty, Load searches for .logpilot.yaml in the current directory.
func Load(path string) (*Config, error) {
	v := viper.New()
	for k, val := range defaults {
		v.SetDefault(k, val)
	}

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName(".logpilot")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// no config file — use defaults only, which is valid
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Default returns a Config populated with default values and no sources.
func Default() *Config {
	return &Config{
		BufferSize: 10000,
		Follow:     true,
		Sources:    []SourceConfig{},
		Colors:     map[string]string{},
	}
}
