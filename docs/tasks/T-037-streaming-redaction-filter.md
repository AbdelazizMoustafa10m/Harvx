# T-037: Streaming Redaction Filter Pipeline

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-034, T-035, T-036
**Phase:** 3 - Security

---

## Description

Implement the core redaction engine that processes file content through the detection patterns and entropy analyzer, replacing detected secrets with `[REDACTED:type]` markers. This is the central execution component of the security subsystem. It operates as a streaming filter in the processing pipeline: content passes through the redactor between file reading and token counting. The redactor supports path-based exclusions, confidence threshold filtering, heightened scanning for sensitive files (.env, .pem), and multi-line secret handling (private key blocks).

## User Story

As a developer, I want Harvx to automatically scan every file's content and replace detected secrets with redaction markers before the content appears in the output, so that I can safely share my codebase context with cloud-hosted LLMs without worrying about credential exposure.

## Acceptance Criteria

- [ ] `StreamRedactor` struct implements the `Redactor` interface from T-034
- [ ] `Redact(ctx context.Context, content string, filePath string) (string, []RedactionMatch, error)` processes content and returns redacted output with match metadata
- [ ] **Two-phase detection per line**:
  1. Keyword pre-filter: skip regex evaluation if no keywords from any rule appear in the line
  2. Regex matching: apply compiled patterns to lines that pass keyword filter
- [ ] **Multi-line handling**: Detect `-----BEGIN * PRIVATE KEY-----` blocks and redact the entire block through `-----END * PRIVATE KEY-----`
- [ ] **Entropy analysis integration**: After regex pass, run entropy analysis on unmatched tokens in suspicious contexts (variable assignments to `key`, `secret`, `token`, `password`, `credential` names)
- [ ] **Path exclusion support**: Skip redaction entirely for files matching `exclude_paths` glob patterns from config (e.g., `**/*test*/**`, `**/fixtures/**`)
- [ ] **Confidence threshold filtering**: Only apply rules at or above the configured confidence threshold (e.g., if threshold is `high`, skip medium and low rules)
- [ ] **Heightened scanning for sensitive files**: When file path matches `.env`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, or patterns like `*secret*`, `*credential*`, lower the confidence threshold by one level and enable entropy scanning on all tokens
- [ ] **Replacement format**: `[REDACTED:secret_type]` where secret_type comes from the matching rule (e.g., `[REDACTED:aws_access_key]`, `[REDACTED:private_key_block]`, `[REDACTED:connection_string]`)
- [ ] **Multiple matches per line**: Handle cases where a single line contains multiple secrets (e.g., `AWS_KEY=AKIA... AWS_SECRET=...`)
- [ ] **Redaction is idempotent**: Running the redactor twice on the same content produces the same output (already-redacted markers are not re-processed)
- [ ] **Context preservation**: Redaction preserves line structure, indentation, and surrounding non-secret content
- [ ] Collects all `RedactionMatch` instances for reporting (T-039)
- [ ] Returns aggregated `RedactionSummary` via a `Summary()` method
- [ ] Thread-safe: multiple goroutines can call `Redact` concurrently on different files (the redactor holds no mutable shared state after initialization)
- [ ] Unit tests achieve >= 90% coverage on the redaction engine
- [ ] Performance: processes a 10,000-line file in under 50ms with all patterns active

## Technical Notes

- **Architecture**: The `StreamRedactor` is constructed with a `PatternRegistry` (from T-035), an `EntropyAnalyzer` (from T-036), and a `RedactionConfig` (from T-034). It is stateless per-call -- match results are returned, not accumulated internally.
- **Processing flow per file**:
  ```
  1. Check if file path matches exclude_paths -> skip if yes
  2. Detect if file is a sensitive file type -> set heightened mode
  3. Split content into lines
  4. For each line:
     a. Run keyword pre-filter against all active rules
     b. For matching rules, apply regex to the line
     c. For each regex match, record RedactionMatch and apply replacement
     d. If heightened mode or entropy enabled, tokenize remaining content and run entropy analysis
  5. Handle multi-line patterns (private key blocks)
  6. Join lines back into content string
  7. Return redacted content + all matches
  ```
- **Keyword pre-filter implementation**: Build a single pass through the line checking for all keywords. Use `strings.ToLower` for case-insensitive keyword matching on key names, but preserve case sensitivity for value prefixes like `AKIA`, `ghp_`, `sk_live_`.
- **Multi-line private key handling**: Use a state machine approach. When `-----BEGIN` is detected, enter "block mode" and redact all subsequent lines until `-----END` is found. Replace the entire block with a single `[REDACTED:private_key_block]` marker.
- **Path exclusion matching**: Use `bmatcuk/doublestar/v4` for glob matching against the file's relative path. This is the same glob engine used by the discovery and relevance systems.
- **Sensitive file detection**: Compile a list of patterns at init time: `*.env`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, `*secret*`, `*credential*`, `*password*`, `.env.*`. Check using `doublestar.Match`.
- **Concurrency**: The redactor struct is immutable after construction (compiled regexes, config, analyzer are all read-only). Each `Redact` call operates on its own data. This makes it safe for concurrent use from the errgroup workers in the pipeline.
- **Pipeline integration point**: In the processing pipeline (Section 6.3 of PRD), redaction happens during the "Content Loading" stage. Each file worker calls `redactor.Redact(ctx, fileContent, filePath)` and stores the redacted content in `FileDescriptor.Content` and the match count in `FileDescriptor.Redactions`.
- **Performance optimization**: The keyword pre-filter should eliminate 90%+ of lines from regex evaluation in typical codebases. Only lines containing suspicious keywords undergo the more expensive regex pass.

## Files to Create/Modify

- `internal/security/redactor.go` - `StreamRedactor` struct, `NewStreamRedactor` constructor, `Redact` method, `Summary` method
- `internal/security/redactor_test.go` - Comprehensive unit tests
- `internal/security/sensitive.go` - Sensitive file detection helpers
- `internal/security/sensitive_test.go` - Tests for sensitive file detection
- `internal/pipeline/types.go` - Verify `FileDescriptor` has `Redactions int` field (from PRD Section 6.5)

## Testing Requirements

- **End-to-end redaction tests**: Full file content with embedded secrets, verify correct replacement
  - File with single AWS key -> `[REDACTED:aws_access_key]`
  - File with multiple different secret types -> each replaced correctly
  - File with private key block spanning 10+ lines -> single `[REDACTED:private_key_block]`
  - `.env` file with `DATABASE_URL=postgres://...` -> `[REDACTED:connection_string]`
- **Path exclusion tests**:
  - File matching `**/*test*/**` is not redacted
  - File matching `**/fixtures/**` is not redacted
  - File NOT matching any exclusion IS redacted
- **Confidence threshold tests**:
  - Threshold `high`: only high-confidence rules apply
  - Threshold `medium`: high and medium rules apply
  - Threshold `low`: all rules apply
- **Heightened scanning tests**:
  - `.env` file triggers heightened mode
  - `*.pem` file triggers heightened mode
  - Regular `.go` file does NOT trigger heightened mode
- **Idempotency tests**: Redact output of a previous redaction -> no double redaction
- **Multi-secret line tests**: Line with 2+ secrets -> all replaced correctly
- **Context preservation tests**: Indentation, surrounding code, comments all preserved
- **Empty/nil input tests**: Empty content returns empty content with zero matches
- **Performance benchmarks**:
  - `BenchmarkRedact/small_100_lines` (100-line file, 2 secrets)
  - `BenchmarkRedact/medium_1000_lines` (1000-line file, 10 secrets)
  - `BenchmarkRedact/large_10000_lines` (10K-line file, 50 secrets)
  - `BenchmarkRedact/no_secrets` (1000-line file, 0 secrets -- measures keyword filter efficiency)
