# T-020: Configuration Validation and Lint Engine

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-016 (Config Types & Defaults), T-019 (Profile Inheritance)
**Phase:** 2 - Intelligence (Profiles)

---

## Description

Implement a validation engine that checks loaded profiles for correctness at load time, providing clear actionable error messages. This also powers the `harvx profiles lint` subcommand which runs deeper analysis (overlapping tier rules, unreachable tiers, pattern syntax validation). Validation catches misconfigurations early and prevents confusing runtime behavior.

## User Story

As a developer configuring Harvx for my project, I want immediate, clear feedback when my configuration has errors or potential issues, so that I can fix problems before running context generation.

## Acceptance Criteria

- [ ] `internal/config/validate.go` implements `Validate(cfg *Config) []ValidationError`:
  - Returns a slice of errors/warnings (not just the first one)
  - Each `ValidationError` has: severity (error/warning), field path, message, suggestion
- [ ] **Hard errors** (prevent execution):
  - Invalid `format` value (not one of: markdown, xml, plain)
  - Invalid `tokenizer` value (not one of: cl100k_base, o200k_base, none)
  - Invalid `target` value (not one of: claude, chatgpt, generic, empty)
  - Invalid `confidence_threshold` (not one of: high, medium, low)
  - Negative `max_tokens` value
  - `max_tokens` exceeding 2,000,000 (sanity cap)
  - Invalid glob pattern syntax in relevance tiers, ignore, include, or priority_files
  - Circular profile inheritance (also caught in T-019 but validated here too)
  - Missing parent in `extends` chain
- [ ] **Warnings** (allow execution, shown to user):
  - Overlapping glob patterns across tiers (same pattern matches multiple tiers)
  - Empty relevance tiers that could be removed
  - `priority_files` entries that also appear in `ignore` (contradictory)
  - `priority_files` with glob patterns (should be exact paths)
  - `exclude_paths` in redaction that overlap with `ignore` (redundant)
  - Inheritance depth > 3 levels
  - `max_tokens` > 500,000 (unusually large, might be a mistake)
  - Output path outside the current directory tree (potential misconfiguration)
- [ ] Glob pattern syntax validation uses `bmatcuk/doublestar` for compilation check
- [ ] `internal/config/validate.go` also implements `Lint(cfg *Config) []LintResult`:
  - All validation checks plus deeper analysis
  - Detects unreachable tiers (tier N patterns are subset of tier N-1)
  - Detects tier patterns that match no common file extensions
  - Reports profile configuration complexity score
- [ ] Error messages follow the pattern: "what went wrong" + "how to fix it"
  - e.g., `profile 'work': format "html" is invalid. Valid formats: markdown, xml, plain`
- [ ] Validation runs automatically during config resolution (T-017)
- [ ] Unit tests cover all error and warning conditions

## Technical Notes

- Use `bmatcuk/doublestar` v4 for glob pattern compilation/validation -- call `doublestar.Match(pattern, "")` and check for error to validate syntax without actually matching
- The `ValidationError` struct should implement `error` interface and also have structured fields:
  ```go
  type ValidationError struct {
      Severity string // "error" or "warning"
      Field    string // e.g., "profile.finvault.format"
      Message  string
      Suggest  string // fix suggestion
  }
  ```
- Overlapping tier detection: for each pattern in tier N, check if it also matches in any tier < N. Use a simple heuristic (exact string match) since full glob overlap detection is NP-hard
- Lint results include everything from Validate plus additional analysis
- Validation should be fast (< 10ms for typical configs) since it runs on every invocation
- Consider using `log/slog` to emit warnings at `slog.LevelWarn`

## Files to Create/Modify

- `internal/config/validate.go` - Validation engine and lint logic
- `internal/config/validate_test.go` - Comprehensive validation tests
- `internal/config/errors.go` - ValidationError type and formatting
- `testdata/config/invalid_format.toml` - Config with invalid format value
- `testdata/config/overlapping_tiers.toml` - Config with overlapping tier patterns
- `testdata/config/contradictory.toml` - Config with priority_files in ignore list

## Testing Requirements

- Unit test: Invalid format returns error with valid options listed
- Unit test: Invalid tokenizer returns error
- Unit test: Invalid target returns error
- Unit test: Negative max_tokens returns error
- Unit test: Valid config returns no errors
- Unit test: Overlapping tier patterns returns warning
- Unit test: Priority file in ignore list returns warning
- Unit test: Invalid glob syntax (`[invalid`) returns error with field path
- Unit test: Multiple errors returned (not just first)
- Unit test: Error messages include fix suggestions
- Unit test: Lint detects unreachable tiers
- Unit test: Empty config (all defaults) passes validation
- Unit test: Huge max_tokens (> 500K) returns warning
- Unit test: Config with all valid fields passes cleanly
- Edge case: Patterns with special characters validate correctly
- Edge case: Unicode in file patterns handled

## References

- [bmatcuk/doublestar v4](https://github.com/bmatcuk/doublestar)
- PRD Section 5.2 - "`harvx profiles lint` validates patterns, warns on overlapping tier rules or unreachable tiers"
- PRD Section 5.2 - "Validate profiles at load time and provide clear error messages for invalid configurations"
- PRD Section 8.1 - "Helpful errors. Every error message should include what went wrong, why, and how to fix it."
