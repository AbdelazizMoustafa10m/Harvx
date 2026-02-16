# T-010: Exit Code Handling

**Priority:** Must Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-003, T-005
**Phase:** 1 - Foundation

---

## Description

Implement structured exit code handling throughout the CLI, ensuring that Harvx exits with well-defined codes for different outcomes: 0 for success, 1 for errors (including `--fail-on-redaction`), and 2 for partial success (some files failed but output was generated). Define a custom error type that carries exit code information, and wire it through the Cobra command execution path from `main.go`.

## User Story

As a CI pipeline integrator, I want Harvx to return meaningful exit codes so that my scripts can programmatically detect success, failure, and partial-success scenarios.

## Acceptance Criteria

- [ ] Exit code constants are used from `internal/pipeline/types.go` (defined in T-003): `ExitSuccess = 0`, `ExitError = 1`, `ExitPartial = 2`
- [ ] `internal/pipeline/errors.go` defines a `HarvxError` type that wraps an error with an exit code:
  ```go
  type HarvxError struct {
      Code    int
      Message string
      Err     error
  }
  ```
- [ ] `HarvxError` implements the `error` interface
- [ ] `HarvxError` implements `Unwrap()` for `errors.Is()` / `errors.As()` compatibility
- [ ] Convenience constructors exist: `NewError(msg string, err error)`, `NewPartialError(msg string, err error)`, `NewRedactionError(msg string)`
- [ ] `cmd/harvx/main.go` extracts the exit code from errors returned by `cli.Execute()`:
  - If error is `*HarvxError`, use its `Code`
  - If error is generic, use `ExitError` (1)
  - If nil, use `ExitSuccess` (0)
- [ ] Error messages are logged to stderr (not stdout) before exiting
- [ ] When `--fail-on-redaction` is triggered (future implementation), the error returned is a `HarvxError` with code 1
- [ ] Partial errors (exit code 2) include a summary of which files failed and why
- [ ] Unit tests verify exit code extraction for all three scenarios
- [ ] Unit tests verify error wrapping/unwrapping

## Technical Notes

- The exit code flow:
  ```
  main.go -> cli.Execute() -> rootCmd.RunE -> pipeline.Run()
       |                                           |
       |<-- returns *HarvxError or error ----------|
       |
       os.Exit(extractExitCode(err))
  ```
- The `HarvxError` type:
  ```go
  package pipeline

  type HarvxError struct {
      Code    int
      Message string
      Err     error
  }

  func (e *HarvxError) Error() string {
      if e.Err != nil {
          return fmt.Sprintf("%s: %v", e.Message, e.Err)
      }
      return e.Message
  }

  func (e *HarvxError) Unwrap() error {
      return e.Err
  }

  func NewError(msg string, err error) *HarvxError {
      return &HarvxError{Code: ExitError, Message: msg, Err: err}
  }

  func NewPartialError(msg string, err error) *HarvxError {
      return &HarvxError{Code: ExitPartial, Message: msg, Err: err}
  }
  ```
- Per PRD Section 5.9: "Exit codes: 0 (success), 1 (error or `--fail-on-redaction` triggered), 2 (partial -- some files failed but output was generated)."
- Per PRD Section 8.1: "Fail fast. Invalid config exits immediately with a clear message. Partial success (5/1000 files failed) returns exit code 2 with a summary."
- Do not use `log.Fatal` or `os.Exit` anywhere except `main.go`. All other code returns errors that bubble up.
- Reference: PRD Sections 5.9, 8.1

## Files to Create/Modify

- `internal/pipeline/errors.go` - HarvxError type and constructors
- `internal/pipeline/errors_test.go` - Unit tests
- `cmd/harvx/main.go` - Exit code extraction logic
- `internal/cli/root.go` - Ensure errors propagate correctly

## Testing Requirements

- Unit test: `NewError("msg", err)` has Code == 1
- Unit test: `NewPartialError("msg", err)` has Code == 2
- Unit test: `HarvxError.Error()` returns formatted message
- Unit test: `HarvxError.Unwrap()` returns wrapped error
- Unit test: `errors.Is(harvxErr, wrappedErr)` works correctly
- Unit test: `errors.As(err, &harvxErr)` extracts HarvxError from wrapped chain
- Unit test: exit code extraction function returns 0 for nil, 1 for generic error, 1 for HarvxError{Code:1}, 2 for HarvxError{Code:2}
- Integration test: compiled binary returns correct exit code for error scenario
