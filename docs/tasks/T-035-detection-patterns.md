# T-035: Gitleaks-Inspired Secret Detection Patterns

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-034
**Phase:** 3 - Security

---

## Description

Implement the full set of compiled regex-based secret detection patterns modeled after the Gitleaks ruleset. This task registers all pattern rules into the `PatternRegistry` defined in T-034, covering AWS keys, GitHub tokens, Stripe keys, OpenAI keys, private key blocks, connection strings, password/secret assignments, cloud provider credentials, JWT tokens, and generic API key patterns. Each rule includes keywords for pre-filter optimization, a confidence level, and a descriptive secret type used in the `[REDACTED:type]` replacement.

## User Story

As a developer using Harvx, I want the tool to detect a comprehensive set of real-world secret formats so that I never accidentally expose credentials like AWS keys, GitHub tokens, or database connection strings when sharing context with LLMs.

## Acceptance Criteria

- [ ] All patterns compile successfully with Go's standard `regexp` package (RE2 syntax, no lookaheads)
- [ ] **High-confidence patterns** (must detect with near-zero false negatives):
  - [ ] AWS access key IDs: `AKIA[A-Z0-9]{16}`, `ASIA[A-Z0-9]{16}` (also ABIA, ACCA, A3T prefixes)
  - [ ] AWS secret access keys: 40-character base64 strings near AWS context keywords
  - [ ] GitHub personal access tokens: `ghp_[A-Za-z0-9]{36}`, `gho_[A-Za-z0-9]{36,}`, `ghs_[A-Za-z0-9]{36,}`, `ghr_[A-Za-z0-9]{36,}`
  - [ ] GitHub fine-grained PATs: `github_pat_[A-Za-z0-9_]{22,}`
  - [ ] Private key blocks: `-----BEGIN [A-Z ]*PRIVATE KEY-----` through `-----END`
  - [ ] Stripe live keys: `sk_live_[A-Za-z0-9]{24,}`, `pk_live_[A-Za-z0-9]{24,}`, `rk_live_[A-Za-z0-9]{24,}`
- [ ] **Medium-confidence patterns**:
  - [ ] OpenAI API keys: `sk-[A-Za-z0-9]{20,}` (with keyword context to reduce FPs from other sk- prefixed strings)
  - [ ] Connection strings: `(?:postgres|postgresql|mysql|mongodb|mongodb\+srv|redis|amqp|amqps)://[^\s'"]+`
  - [ ] GCP service account JSON: `"type"\s*:\s*"service_account"` with `"private_key"` nearby
  - [ ] Azure connection strings: `DefaultEndpointsProtocol=https;AccountName=`
  - [ ] JWT tokens: `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`
  - [ ] Generic API key assignments: `(?i)(api[_-]?key|apikey|api[_-]?secret)\s*[=:]\s*['"]?[A-Za-z0-9_\-]{16,}['"]?`
  - [ ] Slack tokens: `xox[bpors]-[A-Za-z0-9-]{10,}`
  - [ ] Twilio keys: `SK[a-f0-9]{32}`
  - [ ] SendGrid keys: `SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}`
- [ ] **Low-confidence patterns** (higher false positive rate, useful as signals):
  - [ ] Password assignments: `(?i)(password|passwd|pwd)\s*[=:]\s*['"][^\s'"]{8,}['"]`
  - [ ] Secret/token assignments: `(?i)(secret|token|credential)\s*[=:]\s*['"][^\s'"]{8,}['"]`
  - [ ] Bearer tokens: `(?i)bearer\s+[A-Za-z0-9_\-.]{20,}`
  - [ ] Hex-encoded secrets: `(?i)(secret|key|token|password)\s*[=:]\s*['"]?[0-9a-f]{32,}['"]?`
- [ ] Each rule has `Keywords` populated for pre-filter optimization (e.g., AWS rules have keywords `["AKIA", "ASIA", "aws"]`)
- [ ] `NewDefaultRegistry()` returns a registry pre-loaded with all patterns
- [ ] Unit tests for every single pattern with positive matches (real-format test strings) and negative matches (similar but non-secret strings)
- [ ] A `patterns_test.go` file serves as the beginning of the regression test corpus

## Technical Notes

- **Pattern source reference**: Gitleaks TOML config at https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml -- use as inspiration but rewrite all patterns in RE2 syntax (no lookaheads/lookbehinds).
- **Go regexp does not support lookaheads.** Where gitleaks uses `(?!...)` or `(?=...)`, replace with keyword pre-filtering or post-match validation in code.
- **Keyword pre-filtering**: Before applying a regex to a line, check if any of the rule's keywords appear in the line (case-insensitive for key names, case-sensitive for key value prefixes like `AKIA`). Use `strings.Contains` or `bytes.Contains` for O(n) check. This optimization is critical: gitleaks processes millions of lines and attributes significant speedup to keyword filtering.
- **Pattern compilation**: All regexes must be compiled at package init time using `regexp.MustCompile`. Store compiled patterns in the registry. Benchmark: compiled regex match is ~244ns/op vs ~9478ns/op for per-call compilation.
- **Test corpus safety**: Test strings must be clearly synthetic/fake. Use patterns like `AKIAIOSFODNN7EXAMPLE` (AWS's own example key), `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef12` (fake), etc. Never use real credentials.
- **Connection string patterns**: Must handle URL-encoded passwords in connection strings (e.g., `postgres://user:p%40ssw0rd@host/db`).
- **JWT detection**: The `eyJ` prefix is the base64 encoding of `{"` which starts all JWT headers. Validate that the matched string has exactly 2 dots separating 3 base64url segments. Implement a `validateJWT(match string) bool` post-match check to reduce false positives.
- **Private key blocks**: These can span multiple lines. The pattern should match the BEGIN line, and the redactor (T-036) will handle multi-line block replacement.
- Place all pattern definitions in `internal/security/patterns.go` with a clear, categorized structure.

## Files to Create/Modify

- `internal/security/patterns.go` - All pattern definitions organized by category, `NewDefaultRegistry()` function
- `internal/security/patterns_test.go` - Comprehensive test corpus with positive and negative matches per pattern
- `internal/security/validate.go` - Post-match validators (e.g., `validateJWT`, `validateAWSKeyID`)
- `internal/security/validate_test.go` - Tests for validators

## Testing Requirements

- **Per-pattern positive tests**: At least 2-3 positive match examples per rule using synthetic but realistic secret formats
- **Per-pattern negative tests**: At least 2-3 strings that look similar but should NOT match (e.g., `AKIAIOSFODNN7` with only 13 chars, `sk_test_...` for Stripe test keys which should not be redacted)
- **Keyword pre-filter tests**: Verify that keyword filtering correctly narrows candidate lines
- **Edge cases**:
  - Secrets at start/end of line
  - Secrets embedded in JSON values
  - Secrets in YAML config files
  - Secrets with surrounding whitespace
  - Multiple secrets on the same line
  - Secrets in comments (// or #)
  - URL-encoded characters in connection strings
  - Stripe test keys (`sk_test_`) should NOT be flagged (only live keys)
  - `AKIAIOSFODNN7EXAMPLE` should be flagged (it is the AWS example but still a valid key format)
- **JWT validation tests**: Valid 3-segment JWT vs strings that start with `eyJ` but are not valid JWTs
- **Regression corpus file**: Create `testdata/secrets/patterns_corpus.go` containing all test cases as a structured Go test table that can be extended over time
