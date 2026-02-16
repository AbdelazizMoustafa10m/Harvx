# T-040: CLI Redaction Flags and Profile Configuration

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-034, T-037, T-039
**Phase:** 3 - Security

---

## Description

Wire the redaction subsystem into the CLI layer and profile configuration system. This task adds the `--no-redact`, `--fail-on-redaction`, and `--redaction-report` flags to the CLI; adds the `[profile.*.redaction]` TOML configuration block to the profile parser; integrates custom redaction patterns from profile config; and connects the redaction pipeline into the main processing pipeline so it executes between file reading and token counting. This is the integration task that makes the security subsystem operational end-to-end.

## User Story

As a developer, I want to control redaction behavior through CLI flags and profile configuration, so that I can disable redaction when working locally, enforce it in CI, and customize patterns per project.

## Acceptance Criteria

- [ ] **`--no-redact` flag**: Disables secret redaction entirely. When set, the redaction pipeline is skipped and files pass through unmodified. Equivalent to `redaction.enabled = false` in profile.
- [ ] **`--fail-on-redaction` flag**: Exit with code 1 if any secrets are detected during scanning. Designed for CI enforcement. The output is still generated (with redactions applied), but the exit code signals failure.
  - Exit code 0: success, no secrets found (or redaction disabled)
  - Exit code 1: secrets detected and `--fail-on-redaction` is set
  - Exit code 2: partial failure (some files could not be processed)
- [ ] **`--redaction-report [path]` flag**: Generate a detailed redaction report (delegates to T-039). Optional path argument defaults to `harvx-redaction-report.json`.
- [ ] **Redaction enabled by default**: When no flags or config override, redaction runs with `confidence_threshold = "medium"`.
- [ ] **Profile TOML configuration** parsed correctly:
  ```toml
  [profile.myproject.redaction]
  enabled = true
  exclude_paths = ["**/*test*/**", "**/fixtures/**", "docs/**/*.md"]
  confidence_threshold = "high"  # "high", "medium", "low"
  override_sensitive_defaults = false
  
  [[profile.myproject.redaction.custom_patterns]]
  id = "internal-api-key"
  description = "Internal API key format"
  regex = "MYCO_[A-Z0-9]{32}"
  secret_type = "internal_api_key"
  confidence = "high"
  keywords = ["MYCO_"]
  ```
- [ ] **Custom patterns from profile**: Patterns defined in `redaction.custom_patterns` are compiled and registered in the `PatternRegistry` alongside the built-in patterns. Invalid regex in custom patterns produces a clear error at config load time.
- [ ] **Config precedence**: CLI flags override profile config: `--no-redact` overrides `redaction.enabled = true` in profile.
- [ ] **Pipeline integration**: The redactor is instantiated during pipeline setup (after config resolution) and injected into the content loading stage. Each file worker calls `redactor.Redact()` before passing content to token counting.
- [ ] **Summary in CLI output**: The post-generation summary (printed to stderr) includes the redaction line:
  ```
  Redactions:  3 (2 API keys, 1 connection string)
  ```
  Or when no secrets found:
  ```
  Redactions:  0
  ```
- [ ] **Verbose mode**: When `--verbose` is set, log each redaction as it happens: `slog.Debug("secret redacted", "file", path, "line", lineNum, "type", secretType, "confidence", confidence)`
- [ ] Environment variable overrides: `HARVX_NO_REDACT=1`, `HARVX_FAIL_ON_REDACTION=1`
- [ ] Unit and integration tests for all flag combinations

## Technical Notes

- **CLI framework**: Flags are added to the root command or generate subcommand using `spf13/cobra`. The `--redaction-report` flag uses `cobra`'s `OptionalValue` pattern or a string flag with a default.
- **Config parsing**: The `[profile.*.redaction]` block is parsed by the existing TOML config loader (`BurntSushi/toml`). Add `RedactionConfig` as a field on the profile struct. Default values: `enabled=true`, `confidence_threshold="medium"`, `exclude_paths=[]`, `custom_patterns=[]`.
- **Custom pattern compilation**: When loading a profile, iterate over `custom_patterns`, compile each regex with `regexp.Compile` (not `MustCompile` -- return a clear error), and register them in the `PatternRegistry`.
- **Pipeline wiring**: In `internal/pipeline/pipeline.go`, after file discovery and relevance sorting, during the content loading phase, wrap the file reading with a redaction step:
  ```go
  content, matches, err := redactor.Redact(ctx, rawContent, file.Path)
  file.Content = content
  file.Redactions = len(matches)
  allMatches = append(allMatches, matches...)
  ```
- **Fail-on-redaction timing**: The exit code decision happens after all files are processed and the output is written. The pipeline returns the total redaction count, and the CLI layer checks `--fail-on-redaction` to decide the exit code.
- **Flag conflict handling**: `--no-redact` and `--fail-on-redaction` are mutually exclusive. If both are set, emit a warning and `--no-redact` takes precedence (you cannot fail on redaction if redaction is disabled).
- Reference: PRD Section 5.9 for CLI flag definitions, Section 5.5 for redaction config.

## Files to Create/Modify

- `internal/cli/root.go` or `internal/cli/generate.go` - Add `--no-redact`, `--fail-on-redaction`, `--redaction-report` flags
- `internal/config/config.go` - Add `RedactionConfig` to profile struct, parse TOML section
- `internal/config/validate.go` - Validate redaction config (valid confidence threshold, compilable custom regexes)
- `internal/pipeline/pipeline.go` - Integrate redactor into content loading stage
- `internal/security/custom.go` - Custom pattern compilation from profile config
- `internal/security/custom_test.go` - Tests for custom pattern loading
- `internal/cli/root_test.go` or integration tests - Flag handling tests

## Testing Requirements

- **Flag tests**:
  - `--no-redact` skips redaction entirely
  - `--fail-on-redaction` returns exit code 1 when secrets found
  - `--fail-on-redaction` returns exit code 0 when no secrets found
  - `--no-redact --fail-on-redaction` together: warning emitted, no-redact wins
  - `--redaction-report` without value uses default path
  - `--redaction-report=custom.json` uses custom path
- **Profile config tests**:
  - Parse valid `[redaction]` block with all fields
  - Parse minimal `[redaction]` block (enabled only)
  - Default values when no `[redaction]` block (enabled=true, threshold=medium)
  - Custom patterns with valid regex compile successfully
  - Custom patterns with invalid regex produce clear error message
  - `exclude_paths` glob patterns are valid
- **Config precedence tests**:
  - Profile says `enabled = true`, CLI says `--no-redact` -> disabled
  - Profile says `confidence_threshold = "low"`, no CLI override -> low
  - No profile, no flags -> enabled with medium threshold (defaults)
- **Pipeline integration tests**:
  - File with a secret is redacted in the output
  - File matching exclude_paths is not redacted
  - Token count reflects redacted content (not original)
  - `FileDescriptor.Redactions` is populated correctly
- **Environment variable tests**:
  - `HARVX_NO_REDACT=1` disables redaction
  - `HARVX_FAIL_ON_REDACTION=1` enables CI mode
- **Exit code tests**:
  - Normal run with no secrets -> exit 0
  - Run with secrets, no `--fail-on-redaction` -> exit 0
  - Run with secrets, `--fail-on-redaction` -> exit 1
  - Run with errors in some files -> exit 2
