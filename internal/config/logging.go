// Package config provides configuration loading, validation, and logging
// setup for the harvx CLI tool. This package is a foundational cross-cutting
// concern used by every other internal package.
//
// The logging subsystem uses Go's stdlib log/slog package exclusively. All log
// output is directed to os.Stderr to keep stdout clean for piped output.
package config

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// SetupLogging configures the global slog default logger with the given log
// level and format. The format parameter should be "json" for JSON output or
// any other value (including empty string) for human-readable text output. All
// log output is directed to os.Stderr.
//
// This function is safe to call multiple times (idempotent). Each call
// replaces the previous global logger configuration.
func SetupLogging(level slog.Level, format string) {
	SetupLoggingWithWriter(level, format, os.Stderr)
}

// SetupLoggingWithWriter configures the global slog default logger with the
// given log level, format, and output writer. This variant exists primarily
// for testing, allowing log output to be captured in a buffer rather than
// written to os.Stderr.
//
// The format parameter should be "json" for JSON output or any other value
// for human-readable text output. This function is safe to call multiple
// times (idempotent).
func SetupLoggingWithWriter(level slog.Level, format string, w io.Writer) {
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// ResolveLogLevel determines the appropriate slog.Level based on CLI flags and
// environment variables. The priority order (highest to lowest) is:
//
//  1. HARVX_DEBUG=1 environment variable -> slog.LevelDebug
//  2. verbose flag (--verbose) -> slog.LevelDebug
//  3. quiet flag (--quiet) -> slog.LevelError
//  4. Default -> slog.LevelInfo
//
// If both verbose and quiet are true, verbose wins (debug level). The
// HARVX_DEBUG environment variable always takes highest priority.
func ResolveLogLevel(verbose, quiet bool) slog.Level {
	if os.Getenv("HARVX_DEBUG") == "1" {
		return slog.LevelDebug
	}

	if verbose {
		return slog.LevelDebug
	}

	if quiet {
		return slog.LevelError
	}

	return slog.LevelInfo
}

// ResolveLogFormat reads the HARVX_LOG_FORMAT environment variable and returns
// the log format string. Returns "json" if the environment variable is set to
// "json" (case-insensitive), otherwise returns "text".
func ResolveLogFormat() string {
	if strings.EqualFold(os.Getenv("HARVX_LOG_FORMAT"), "json") {
		return "json"
	}
	return "text"
}

// NewLogger returns a child logger derived from the global default logger with
// a "component" attribute set to the given name. This allows log output to be
// filtered or identified by subsystem (e.g., "discovery", "cli", "security").
//
// Example usage:
//
//	logger := config.NewLogger("discovery")
//	logger.Info("walking directory", "root", "/path/to/repo")
//	// Output: level=INFO msg="walking directory" component=discovery root=/path/to/repo
func NewLogger(component string) *slog.Logger {
	return slog.Default().With("component", component)
}
