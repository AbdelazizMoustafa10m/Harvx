# T-005: Cobra CLI Framework & Root Command

**Priority:** Must Have
**Effort:** Medium (6-8hrs)
**Dependencies:** T-001, T-002, T-003, T-004
**Phase:** 1 - Foundation

---

## Description

Set up the Cobra CLI framework with the root command (`harvx`), integrate structured logging initialization, and wire up `cmd/harvx/main.go` as the entry point. The root command, when invoked with no subcommands, will eventually run context generation with the default profile (same as `harvx generate`), but for this task it prints a welcome message and usage help. This establishes the command hierarchy skeleton for all subsequent CLI work.

## User Story

As a developer, I want to run `harvx` and see helpful usage information so that I understand the tool's capabilities and available subcommands.

## Acceptance Criteria

- [ ] `spf13/cobra` is added as a dependency in `go.mod` (latest stable v1.8.x)
- [ ] `internal/cli/root.go` defines the root `cobra.Command` with:
  - Use: `harvx`
  - Short: "Harvest your context."
  - Long: Multi-line description explaining what Harvx does
  - SilenceUsage: true (do not print usage on errors)
  - SilenceErrors: true (handle errors manually for proper exit codes)
- [ ] `cmd/harvx/main.go` calls `cli.Execute()` which runs `rootCmd.Execute()`
- [ ] If `rootCmd.Execute()` returns an error, the process exits with the appropriate exit code (1 for errors, 2 for partial -- uses exit code constants from T-003)
- [ ] Root command's `PersistentPreRunE` initializes logging based on `--verbose` / `--quiet` flags (integrated with T-004)
- [ ] Running `harvx` with no arguments prints usage/help (temporary behavior until generate subcommand is wired as default)
- [ ] Running `harvx --help` shows well-formatted help text with the tagline
- [ ] `go build ./cmd/harvx/` produces a working binary
- [ ] `go test ./internal/cli/...` passes

## Technical Notes

- Cobra v1.8.x is the latest stable release as of 2025-2026 (ref: https://github.com/spf13/cobra). Install with `go get github.com/spf13/cobra@latest`.
- The root command structure in `internal/cli/root.go`:
  ```go
  package cli

  import "github.com/spf13/cobra"

  var rootCmd = &cobra.Command{
      Use:   "harvx",
      Short: "Harvest your context.",
      Long:  `Harvx packages codebases into LLM-optimized context documents.`,
  }

  func Execute() int {
      if err := rootCmd.Execute(); err != nil {
          return pipeline.ExitError
      }
      return pipeline.ExitSuccess
  }
  ```
- `main.go` should be minimal:
  ```go
  func main() {
      os.Exit(cli.Execute())
  }
  ```
- Do NOT add global flags in this task -- that is T-007. This task only sets up the Cobra skeleton and logging integration.
- Per PRD Section 5.9: "Use `spf13/cobra` for CLI framework -- provides subcommands, auto-generated help, shell completions, and man pages."
- Per PRD Section 8.1: "SilenceUsage" and "SilenceErrors" are essential for clean error handling -- Cobra's default behavior of printing usage on every error is noisy.
- Reference: PRD Sections 5.9, 6.1, 6.2, 8.1

## Files to Create/Modify

- `go.mod` / `go.sum` - Add cobra dependency
- `internal/cli/root.go` - Root command definition
- `internal/cli/root_test.go` - Unit tests
- `cmd/harvx/main.go` - Wire up cli.Execute()

## Testing Requirements

- Unit test: root command exists and has correct `Use` field
- Unit test: `Execute()` returns `ExitSuccess` (0) when run with `--help`
- Unit test: root command has `SilenceUsage` and `SilenceErrors` set to true
- Unit test: running with an unknown flag returns a non-zero exit code
- Integration test: compiled binary runs and shows help text
