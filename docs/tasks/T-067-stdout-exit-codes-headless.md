# T-067: Clean Stdout Mode, Structured Exit Codes, and Non-Interactive Defaults

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-066 (Pipeline Library API), T-002 (Cobra CLI Setup)
**Phase:** 5 - Workflows

---

## Description

Implement the `--stdout` flag for piping output to other tools, enforce structured exit codes across all commands, and ensure Harvx is non-interactive by default (no prompts unless explicitly requested). This makes Harvx a reliable building block in shell script pipelines and CI/CD workflows.

## User Story

As a developer with a multi-agent review pipeline, I want Harvx to output cleanly to stdout with predictable exit codes so that my shell scripts can consume its output and branch on success/failure without parsing human-readable messages.

## Acceptance Criteria

- [ ] `--stdout` flag sends context output to stdout instead of writing to a file
- [ ] When `--stdout` is active, ALL user-facing messages (progress, warnings, summary) go to stderr via `log/slog`
- [ ] Structured exit codes are enforced across all commands:
  - `0` - Success (all files processed)
  - `1` - Error (fatal failure, or `--fail-on-redaction` triggered)
  - `2` - Partial success (some files failed but output was generated)
- [ ] Exit codes are returned from the pipeline library (not via `os.Exit` in library code) and translated to `os.Exit` only at the CLI boundary in `cmd/harvx/main.go`
- [ ] Harvx is non-interactive by default: no confirmation prompts, no TUI launch, no user input required
- [ ] The `--yes` flag is accepted (for forward compatibility) but is a no-op since non-interactive is already the default
- [ ] Progress output (bars, spinners) is auto-disabled when stdout is a pipe (detected via `os.Stdout.Stat()` checking for `ModeCharDevice`)
- [ ] Color output via lipgloss is auto-disabled when stderr is piped
- [ ] `HARVX_STDOUT=true` environment variable works as an alternative to `--stdout`
- [ ] Unit tests verify exit code mapping for all scenarios
- [ ] Integration test: `harvx --stdout | wc -l` produces valid piped output

## Technical Notes

- Use `os.Stdout.Stat()` to detect pipe mode: if `fi.Mode()&os.ModeCharDevice == 0`, output is being piped
- lipgloss already auto-detects terminal capabilities; ensure this detection uses stderr (not stdout) when `--stdout` is active
- Exit code translation pattern: `pipeline.Run()` returns `RunResult` with an `ExitCode` field; the CLI layer calls `os.Exit(result.ExitCode)`
- Progress bars (`schollz/progressbar`) should write to stderr and be suppressed when stderr is also piped
- The `--quiet` flag should suppress ALL stderr output except fatal errors
- Reference: PRD Sections 5.9 (exit codes), 5.10 (clean stdout, non-interactive default), 8.1 (minimal output by default)

## Files to Create/Modify

- `internal/cli/root.go` - Add `--stdout`, `--yes` flags; exit code handling at CLI boundary
- `internal/cli/output.go` - Pipe detection, stderr routing logic
- `internal/output/writer.go` - Output destination selection (file vs stdout)
- `cmd/harvx/main.go` - `os.Exit()` call based on pipeline result exit code
- `internal/cli/output_test.go` - Pipe detection and stderr routing tests
- `internal/pipeline/result.go` - Ensure ExitCode field exists (from T-066)

## Testing Requirements

- Unit test: `--stdout` routes context to stdout and messages to stderr
- Unit test: Exit code 0 for successful generation
- Unit test: Exit code 1 for fatal errors
- Unit test: Exit code 2 for partial success (some files failed)
- Unit test: Exit code 1 when `--fail-on-redaction` detects secrets
- Unit test: Pipe detection returns correct value for terminal vs pipe
- Unit test: `--quiet` suppresses all non-fatal stderr output
- Integration test: Pipeline output can be piped to another command
- Edge case: `--stdout` combined with `--output` flag returns error (mutually exclusive)