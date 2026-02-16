# T-006: Version Command & Build Info

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-005
**Phase:** 1 - Foundation

---

## Description

Implement the `harvx version` subcommand that displays version, git commit SHA, build date, Go version, and OS/architecture. Create an `internal/buildinfo` package to hold build-time variables injected via ldflags, and update the Makefile to inject into this package instead of `main`.

## User Story

As a developer or CI system, I want to run `harvx version` so that I can verify which build is deployed and report version info in bug reports.

## Acceptance Criteria

- [ ] `internal/buildinfo/buildinfo.go` exports package-level variables: `Version`, `Commit`, `Date`, `GoVersion`
- [ ] Variables have sensible defaults for development builds (e.g., `Version = "dev"`, `Commit = "unknown"`, `Date = "unknown"`)
- [ ] `harvx version` subcommand is registered on the root command
- [ ] Output format (human-readable):
  ```
  harvx version dev
    commit:     abc1234
    built:      2026-02-16T10:00:00Z
    go version: go1.24.13
    os/arch:    darwin/arm64
  ```
- [ ] `harvx version --json` outputs machine-readable JSON:
  ```json
  {
    "version": "dev",
    "commit": "abc1234",
    "date": "2026-02-16T10:00:00Z",
    "goVersion": "go1.24.13",
    "os": "darwin",
    "arch": "arm64"
  }
  ```
- [ ] Makefile ldflags updated to inject into `internal/buildinfo` package (update from T-002's `main` package targeting)
- [ ] `runtime.GOOS` and `runtime.GOARCH` are used for OS/architecture info
- [ ] Unit tests pass for both human and JSON output formats

## Technical Notes

- The `internal/buildinfo` package pattern:
  ```go
  package buildinfo

  var (
      Version   = "dev"
      Commit    = "unknown"
      Date      = "unknown"
      GoVersion = "unknown"
  )
  ```
- Makefile ldflags update:
  ```makefile
  LDFLAGS = -ldflags "-X github.com/yourusername/harvx/internal/buildinfo.Version=$(VERSION) \
    -X github.com/yourusername/harvx/internal/buildinfo.Commit=$(COMMIT) \
    -X github.com/yourusername/harvx/internal/buildinfo.Date=$(DATE) \
    -X github.com/yourusername/harvx/internal/buildinfo.GoVersion=$(shell go version | cut -d' ' -f3)"
  ```
- The version command in `internal/cli/version.go`:
  ```go
  var versionCmd = &cobra.Command{
      Use:   "version",
      Short: "Show version and build information",
      Run: func(cmd *cobra.Command, args []string) { ... },
  }
  ```
- Per PRD Section 5.9: `harvx version` shows "version, build info, supported languages, and tokenizer info." For Phase 1, we only show version/build info. Supported languages and tokenizer info will be added when those features are implemented.
- Reference: PRD Section 5.9

## Files to Create/Modify

- `internal/buildinfo/buildinfo.go` - Build-time variables
- `internal/cli/version.go` - Version subcommand
- `internal/cli/version_test.go` - Unit tests
- `internal/cli/root.go` - Register version subcommand
- `Makefile` - Update ldflags target package

## Testing Requirements

- Unit test: version command outputs correct format
- Unit test: `--json` flag outputs valid JSON with expected keys
- Unit test: default values are "dev"/"unknown" when not injected
- Unit test: `runtime.GOOS` and `runtime.GOARCH` appear in output
- Build test: `make build && bin/harvx version` shows injected version info
