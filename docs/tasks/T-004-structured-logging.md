# T-004: Structured Logging with slog

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-001
**Phase:** 1 - Foundation

---

## Description

Set up the structured logging infrastructure using Go's stdlib `log/slog` package. Implement log level configuration driven by `--verbose` and `--quiet` flags, support for JSON log format via `HARVX_LOG_FORMAT=json` environment variable, and ensure all log output goes to stderr (keeping stdout clean for piped output). This is a foundational cross-cutting concern used by every other package.

## User Story

As a developer using Harvx in a CI pipeline, I want structured JSON logs written to stderr so that I can parse them with my log aggregation tools without polluting the context output on stdout.

## Acceptance Criteria

- [ ] `internal/config/logging.go` provides a `SetupLogging(level slog.Level, format string)` function that configures the global slog default logger
- [ ] Default log level is `slog.LevelInfo`
- [ ] When `--verbose` is active, level is `slog.LevelDebug`
- [ ] When `--quiet` is active, level is `slog.LevelError`
- [ ] `HARVX_LOG_FORMAT=json` environment variable switches to `slog.NewJSONHandler(os.Stderr, opts)`
- [ ] Default (non-JSON) format uses `slog.NewTextHandler(os.Stderr, opts)` for human-readable output
- [ ] All log output is directed to `os.Stderr`, never `os.Stdout`
- [ ] `HARVX_DEBUG=1` environment variable sets level to `slog.LevelDebug` and enables additional diagnostic attributes (functions as a secondary verbose trigger)
- [ ] Provides a `NewLogger(component string) *slog.Logger` helper that returns a child logger with a `component` attribute (e.g., `component=discovery`, `component=cli`)
- [ ] Logger configuration is safe to call multiple times (idempotent)
- [ ] Unit tests verify log level selection logic
- [ ] Unit tests verify JSON vs text handler selection
- [ ] Unit tests verify stderr output (not stdout)

## Technical Notes

- `log/slog` is available since Go 1.21 and is the recommended stdlib structured logging solution. No third-party logging library is needed.
- Reference: https://pkg.go.dev/log/slog, https://go.dev/blog/slog
- The handler selection pattern:
  ```go
  var handler slog.Handler
  opts := &slog.HandlerOptions{Level: level}
  if format == "json" {
      handler = slog.NewJSONHandler(os.Stderr, opts)
  } else {
      handler = slog.NewTextHandler(os.Stderr, opts)
  }
  slog.SetDefault(slog.New(handler))
  ```
- Per PRD Section 6.6: "Progress output and logs go to stderr so stdout remains clean for piping."
- Per PRD Section 6.6: Log level mapping is `default=info`, `--verbose=debug`, `--quiet=error`.
- The `HARVX_LOG_FORMAT` and `HARVX_DEBUG` env vars are read directly with `os.Getenv()` -- no Viper dependency at this stage.
- This package should NOT import anything from `internal/cli/` -- it must be usable by all internal packages without circular imports.
- Reference: PRD Section 6.6

## Files to Create/Modify

- `internal/config/logging.go` - Logging setup functions
- `internal/config/logging_test.go` - Unit tests

## Testing Requirements

- Unit test: default level is `slog.LevelInfo`
- Unit test: verbose flag sets `slog.LevelDebug`
- Unit test: quiet flag sets `slog.LevelError`
- Unit test: `HARVX_LOG_FORMAT=json` produces JSON handler
- Unit test: `HARVX_DEBUG=1` activates debug level
- Unit test: `NewLogger("discovery")` returns logger with component attribute
- Unit test: output is written to stderr (capture and verify using a buffer)
