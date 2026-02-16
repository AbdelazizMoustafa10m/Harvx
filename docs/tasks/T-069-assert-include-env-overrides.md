# T-069: Assert-Include Coverage Checks and Environment Variable Overrides

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-066 (Pipeline Library API), T-017 (Multi-Source Config Merging)
**Phase:** 5 - Workflows

---

## Description

Implement `--assert-include <pattern>` for verifying that critical files are present in the output (coverage checks), and ensure comprehensive environment variable overrides with the `HARVX_` prefix work across all configuration keys. Assert-include is essential for CI pipelines that need to fail if important files (e.g., auth middleware, schema definitions) are accidentally excluded by profile rules.

## User Story

As a CI pipeline maintainer, I want to assert that critical files like `middleware.ts` and `prisma/schema.prisma` are included in the Harvx output so that my review agents never receive context missing key architectural files.

## Acceptance Criteria

- [ ] `--assert-include <pattern>` flag accepts glob patterns (repeatable) and fails with exit code 1 if any pattern matches zero included files
- [ ] Error message on assertion failure includes: the pattern that failed, total files scanned, and a suggestion to check profile ignore/relevance rules
- [ ] Multiple `--assert-include` flags are supported: `--assert-include "middleware.*" --assert-include "prisma/**"`
- [ ] Assert-include runs after the relevance sorting stage but before content loading (fail fast)
- [ ] Assert-include patterns use the same glob engine as relevance tier patterns (`bmatcuk/doublestar/v4`)
- [ ] `assert_include` can also be specified in profile config:
  ```toml
  [profile.finvault]
  assert_include = ["middleware.*", "prisma/schema.prisma", "lib/services/**"]
  ```
- [ ] Environment variable overrides with `HARVX_` prefix are comprehensive:
  - `HARVX_PROFILE` - profile name selection
  - `HARVX_MAX_TOKENS` - token budget (parsed as int)
  - `HARVX_FORMAT` - output format (markdown/xml)
  - `HARVX_TOKENIZER` - tokenizer encoding
  - `HARVX_OUTPUT` - output file path
  - `HARVX_TARGET` - LLM target preset
  - `HARVX_COMPRESS` - enable compression (parsed as bool)
  - `HARVX_REDACT` - enable/disable redaction (parsed as bool)
  - `HARVX_STDOUT` - enable stdout mode (parsed as bool)
  - `HARVX_LOG_FORMAT` - log format: text or json
  - `HARVX_DEBUG` - enable debug mode (dumps resolved config, timings)
- [ ] Env vars sit between config file and CLI flags in precedence (config < env < CLI flags)
- [ ] Invalid env var values produce clear error messages (e.g., `HARVX_MAX_TOKENS=abc` -> "HARVX_MAX_TOKENS must be a positive integer")
- [ ] Unit tests achieve 90%+ coverage for assert-include and env var parsing

## Technical Notes

- Assert-include matching uses `doublestar.Match()` against each `FileDescriptor.Path` (relative path)
- If a pattern matches at least one included file, it passes; if it matches zero, it fails
- When multiple patterns fail, report ALL failures (not just the first one) so the user can fix everything in one pass
- Env var parsing should use koanf's env provider (configured in T-017) with the `HARVX_` prefix and `_` separator
- Boolean env vars accept: `true`, `1`, `yes` (case-insensitive) for true; `false`, `0`, `no` for false
- `HARVX_LOG_FORMAT=json` switches slog to use `slog.NewJSONHandler(os.Stderr, nil)` for CI log parsing
- `HARVX_DEBUG=1` dumps: effective resolved config, per-stage timings, top N slowest files
- Reference: PRD Sections 5.9 (`--assert-include`), 5.10 (env var overrides), 5.11.1 (assert-include for review pipelines)

## Files to Create/Modify

- `internal/pipeline/assert.go` - Assert-include matching logic
- `internal/pipeline/assert_test.go` - Coverage check tests
- `internal/config/env.go` - Enhanced env var parsing (extends T-017)
- `internal/config/env_test.go` - Env var parsing tests
- `internal/config/types.go` - Add `AssertInclude []string` field to Profile struct
- `internal/cli/root.go` - Register `--assert-include` flag

## Testing Requirements

- Unit test: Single pattern matching one file passes
- Unit test: Single pattern matching zero files fails with exit code 1
- Unit test: Multiple patterns -- all pass
- Unit test: Multiple patterns -- some fail, error lists all failures
- Unit test: Glob patterns with `**` and `*` wildcards work correctly
- Unit test: Assert-include from profile config merges with CLI flag patterns
- Unit test: Each HARVX_ env var correctly overrides its config value
- Unit test: Invalid env var values produce descriptive errors
- Unit test: Boolean env vars accept all expected truthy/falsy values
- Unit test: HARVX_LOG_FORMAT=json switches to JSON handler
- Edge case: Assert-include pattern with no files in repo returns clear error
- Edge case: Empty assert-include list (no patterns) is a no-op