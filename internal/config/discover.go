package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

// maxSearchDepth is the maximum number of parent directories to search
// when looking for harvx.toml, to prevent runaway traversal.
const maxSearchDepth = 20

// DiscoverRepoConfig walks up the directory tree from startDir, looking for a
// harvx.toml file. It returns the absolute path of the first harvx.toml found,
// or an empty string if no file is found. The search stops at the filesystem
// root, at a .git directory boundary (repo root), or after maxSearchDepth
// levels, whichever comes first.
//
// Symlinks in the directory chain are resolved before walking to prevent loops.
func DiscoverRepoConfig(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("abs path for %s: %w", startDir, err)
	}

	// Resolve symlinks to avoid loops and get the canonical path.
	// If resolution fails (e.g. dir doesn't exist), fall back to the abs path.
	if resolved, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		abs = resolved
	} else {
		slog.Debug("symlink eval failed, using unresolved path",
			"dir", abs,
			"err", evalErr,
		)
	}

	dir := abs
	for depth := 0; depth < maxSearchDepth; depth++ {
		configPath := filepath.Join(dir, "harvx.toml")
		if _, statErr := os.Stat(configPath); statErr == nil {
			slog.Debug("discovered repo config",
				"path", configPath,
				"depth", depth,
			)
			return configPath, nil
		}

		// Check for .git boundary: if .git exists here, we are at the repo
		// root. After checking for harvx.toml at this level (done above),
		// stop the search regardless.
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			slog.Debug("reached .git boundary, stopping search",
				"dir", dir,
				"depth", depth,
			)
			return "", nil
		}

		// Move to parent directory.
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the filesystem root.
			slog.Debug("reached filesystem root, no harvx.toml found")
			return "", nil
		}
		dir = parent
	}

	slog.Debug("reached max search depth without finding harvx.toml",
		"maxDepth", maxSearchDepth,
	)
	return "", nil
}

// DiscoverGlobalConfig returns the path to the global harvx configuration file,
// following XDG Base Directory conventions. It returns an empty string if the
// file does not exist. No error is returned for a missing file.
//
// Priority:
//   - $XDG_CONFIG_HOME/harvx/config.toml (if XDG_CONFIG_HOME is set)
//   - ~/.config/harvx/config.toml (Linux/macOS)
//   - %APPDATA%\harvx\config.toml (Windows)
func DiscoverGlobalConfig() (string, error) {
	configDir, err := globalConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining global config dir: %w", err)
	}

	path := filepath.Join(configDir, "harvx", "config.toml")

	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Debug("global config not found", "path", path)
			return "", nil
		}
		return "", fmt.Errorf("stat global config %s: %w", path, statErr)
	}

	slog.Debug("discovered global config", "path", path)
	return path, nil
}

// globalConfigDir returns the base configuration directory for the current OS,
// respecting XDG_CONFIG_HOME on non-Windows platforms.
func globalConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// Fall back to os.UserConfigDir on Windows if APPDATA is unset.
			dir, err := os.UserConfigDir()
			if err != nil {
				return "", fmt.Errorf("user config dir: %w", err)
			}
			return dir, nil
		}
		return appData, nil
	}

	// Non-Windows: check XDG_CONFIG_HOME first.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg, nil
	}

	// Fall back to ~/.config on Linux/macOS.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".config"), nil
}
