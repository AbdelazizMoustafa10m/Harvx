// Package config handles loading and validation of gosync configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration.
type Config struct {
	Sync    SyncConfig    `toml:"sync"`
	Logging LoggingConfig `toml:"logging"`
}

// SyncConfig holds synchronization-specific settings.
type SyncConfig struct {
	Ignore      []string `toml:"ignore"`
	MaxFileSize string   `toml:"max_file_size"`
	Checksum    bool     `toml:"checksum"`
	DeleteExtra bool     `toml:"delete_extra"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Sync: SyncConfig{
			Ignore:      []string{".git/", ".DS_Store", "*.tmp"},
			MaxFileSize: "100MB",
			Checksum:    true,
			DeleteExtra: false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Load reads configuration from the given path, or searches for a default config file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = findConfigFile()
	}

	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Validate checks the configuration for consistency.
func (c *Config) Validate() error {
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}

	if c.Logging.Level != "" && !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %q", c.Logging.Level)
	}

	if c.Logging.Format != "" && c.Logging.Format != "text" && c.Logging.Format != "json" {
		return fmt.Errorf("invalid log format: %q (must be text or json)", c.Logging.Format)
	}

	return nil
}

func findConfigFile() string {
	candidates := []string{".gosync.toml", "gosync.toml"}
	for _, name := range candidates {
		if abs, err := filepath.Abs(name); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}