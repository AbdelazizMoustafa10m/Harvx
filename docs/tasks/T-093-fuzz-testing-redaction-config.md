# T-093: Fuzz Testing for Redaction & Config Parsing

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** Redaction package (Phase 3), Config package (Phase 2)
**Phase:** 6 - Polish & Distribution

---

## Description

Implement Go native fuzz tests (`testing.F`) for the two most security-sensitive and input-diverse subsystems: secret redaction and TOML config parsing. Fuzz testing discovers edge cases that handwritten tests miss -- malformed inputs, unusual Unicode sequences, high-entropy strings that look like secrets, and pathological TOML structures that cause panics or excessive resource consumption. This is critical for a security tool that processes untrusted file content.

## User Story

As a security-conscious developer, I want fuzz tests to continuously probe the redaction engine and config parser with random inputs so that we discover crashes, false negatives, and panics before users do.

## Acceptance Criteria

- [ ] `internal/security/fuzz_test.go` contains fuzz tests for the redaction pipeline:
  - `FuzzRedactContent` -- feeds random strings through the full redaction pipeline, asserts no panics and valid UTF-8 output
  - `FuzzRedactHighEntropy` -- generates high-entropy random strings, verifies entropy analyzer doesn't panic and returns a valid confidence score
  - `FuzzRedactEnvFile` -- generates synthetic `.env` file content (`KEY=VALUE` pairs with random values), verifies all high-confidence patterns are caught
  - `FuzzRedactMixedContent` -- generates content mixing code, secrets, and Unicode, verifies output is valid and redaction markers are well-formed
- [ ] `internal/config/fuzz_test.go` contains fuzz tests for config parsing:
  - `FuzzParseConfig` -- feeds random bytes as TOML input, asserts no panics (graceful error handling)
  - `FuzzProfileInheritance` -- generates random profile names and `extends` chains, verifies no infinite loops or stack overflows
  - `FuzzGlobPattern` -- feeds random strings as glob patterns, verifies the pattern matcher doesn't panic
- [ ] Seed corpus files in `testdata/fuzz/` with known edge cases:
  - Realistic AWS keys, GitHub tokens, Stripe keys (known patterns)
  - Unicode strings (emoji, CJK, RTL, zero-width characters)
  - Extremely long lines (100K+ characters)
  - Binary-looking content (null bytes mixed with text)
  - Nested TOML with 100+ depth
  - TOML with duplicate keys
  - Empty and whitespace-only inputs
- [ ] Property invariants verified by all fuzz tests:
  - Output is always valid UTF-8
  - Output length >= 0 (no negative lengths)
  - Redaction markers are well-formed: `[REDACTED:xxx]` where xxx is a non-empty alphanumeric+underscore string
  - Config parsing either returns a valid config or a non-nil error (never both nil)
  - No goroutine leaks (fuzz body completes in bounded time)
- [ ] Fuzz tests can run for at least 30 seconds without finding a crash on the initial corpus
- [ ] `make fuzz` target runs all fuzz tests for a configurable duration (default 30s per test)
- [ ] Any crashes discovered during fuzzing are added to the corpus as regression tests
- [ ] Fuzz corpus directory: `testdata/fuzz/<FuzzTestName>/` following Go convention

## Technical Notes

- Use Go's built-in `testing.F` for fuzz testing (available since Go 1.18). No external fuzzing framework needed.
- Fuzz test structure:
  ```go
  func FuzzRedactContent(f *testing.F) {
      f.Add("password=mysecret123")
      f.Add("AKIA1234567890ABCDEF")
      f.Add("normal code without secrets")
      f.Fuzz(func(t *testing.T, input string) {
          output, matches, err := redactor.Redact(ctx, input, "test.go")
          if err != nil { t.Skip("expected error for malformed input") }
          if !utf8.ValidString(output) { t.Error("output is not valid UTF-8") }
      })
  }
  ```
- Seed corpus should include inputs from the existing unit test suite (reuse test fixtures).
- For the `.env` fuzzer, generate realistic key-value pairs: `f.Add("API_KEY=" + randomHighEntropyString)`.
- For config fuzzing, the seed corpus should include valid minimal TOML, invalid TOML, and boundary cases (empty string, just `[`, nested tables).
- Run fuzzing in CI with a time limit (e.g., `-fuzz=. -fuzztime=60s`) on a schedule (nightly) rather than on every PR.
- Crashes found by fuzzing are automatically saved to `testdata/fuzz/<TestName>/` and become permanent regression tests.
- Reference: PRD Section 9.4 (Fuzz & Property-Based Tests), Go fuzz docs (https://go.dev/doc/security/fuzz/)

## Files to Create/Modify

- `internal/security/fuzz_test.go` - Redaction fuzz tests
- `internal/config/fuzz_test.go` - Config parsing fuzz tests
- `testdata/fuzz/FuzzRedactContent/` - Seed corpus for redaction fuzzing
- `testdata/fuzz/FuzzRedactHighEntropy/` - Seed corpus for entropy fuzzing
- `testdata/fuzz/FuzzRedactEnvFile/` - Seed corpus for .env fuzzing
- `testdata/fuzz/FuzzParseConfig/` - Seed corpus for config fuzzing
- `testdata/fuzz/FuzzGlobPattern/` - Seed corpus for glob pattern fuzzing
- `Makefile` - Add `fuzz` target (modify)
- `.github/workflows/ci.yml` - Add nightly fuzz job (modify)

## Testing Requirements

- All fuzz tests compile and the seed corpus runs without crashes
- `go test -fuzz=FuzzRedactContent -fuzztime=30s` finds no crashes on initial run
- `go test -fuzz=FuzzParseConfig -fuzztime=30s` finds no crashes on initial run
- Property invariants hold for all seed corpus entries
- Redaction output is always valid UTF-8 regardless of input
- Config parsing never panics on any input (returns error for invalid TOML)
- Glob pattern matching never panics on any input string
- Any pre-existing crashes from development are fixed and added to regression corpus