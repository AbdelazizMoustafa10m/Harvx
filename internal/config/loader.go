package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/BurntSushi/toml"
)

// LoadFromFile reads and parses a TOML configuration file at path. It returns
// a fully decoded *Config on success. Unknown TOML keys produce slog warnings
// (not errors) to maintain forward compatibility with future schema additions.
// Invalid TOML syntax causes an error that includes the file path and line
// information from the TOML decoder.
func LoadFromFile(path string) (*Config, error) {
	var cfg Config
	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	warnUndecodedKeys(meta, path)

	return &cfg, nil
}

// LoadFromString parses TOML configuration from an in-memory string. It
// behaves identically to LoadFromFile except the source is a string rather
// than a file. The name parameter is used in log messages and error output.
func LoadFromString(data, name string) (*Config, error) {
	var cfg Config
	meta, err := toml.Decode(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config %s: %w", name, err)
	}

	warnUndecodedKeys(meta, name)

	return &cfg, nil
}

// warnUndecodedKeys logs a warning for each key in the TOML document that did
// not map to any field in the Config struct. This allows users to add new
// fields to their config files without breaking older versions of harvx.
func warnUndecodedKeys(meta toml.MetaData, source string) {
	undecoded := meta.Undecoded()
	if len(undecoded) == 0 {
		return
	}

	keys := make([]string, 0, len(undecoded))
	for _, k := range undecoded {
		keys = append(keys, k.String())
	}

	slog.Warn("unknown config keys will be ignored",
		"source", source,
		"keys", strings.Join(keys, ", "),
	)
}
