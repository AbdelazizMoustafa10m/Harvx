# T-041: Secret Detection Regression Test Corpus and Fuzz Testing

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-035, T-037, T-040
**Phase:** 3 - Security

---

## Description

Build a comprehensive regression test corpus under `testdata/secrets/` containing realistic (but synthetic) secret fixtures across all supported secret types. This corpus serves as the definitive validation suite for the redaction pipeline -- every release must pass it with zero known misses for high-confidence patterns. Additionally, implement fuzz tests for the redactor to discover edge cases, and create golden test files that verify end-to-end redaction output determinism.

## User Story

As a developer maintaining Harvx, I want a curated regression test corpus that catches any regressions in secret detection, so that pattern changes or new features never accidentally reduce detection coverage.

## Acceptance Criteria

- [ ] **Test corpus directory**: `testdata/secrets/` containing categorized fixture files
- [ ] **Fixture files by category** (each file contains multiple examples of that secret type embedded in realistic code/config context):
  - [ ] `testdata/secrets/aws_keys.txt` - AWS access key IDs and secret keys in various contexts
  - [ ] `testdata/secrets/github_tokens.txt` - All GitHub token formats (ghp_, gho_, ghs_, ghr_, github_pat_)
  - [ ] `testdata/secrets/stripe_keys.txt` - Stripe live keys (sk_live_, pk_live_, rk_live_) and test keys (sk_test_ should NOT match)
  - [ ] `testdata/secrets/openai_keys.txt` - OpenAI API keys (sk-...) in config files
  - [ ] `testdata/secrets/private_keys.txt` - RSA, DSA, EC, Ed25519 private key blocks
  - [ ] `testdata/secrets/connection_strings.txt` - postgres://, mysql://, mongodb://, redis:// URLs with credentials
  - [ ] `testdata/secrets/jwt_tokens.txt` - Valid and invalid JWT patterns
  - [ ] `testdata/secrets/cloud_credentials.txt` - GCP service account JSON, Azure connection strings
  - [ ] `testdata/secrets/generic_assignments.txt` - password=, secret=, token=, api_key= in various config formats (YAML, TOML, JSON, .env, .properties)
  - [ ] `testdata/secrets/mixed_file.go` - Realistic Go source file with secrets embedded in code
  - [ ] `testdata/secrets/mixed_file.ts` - Realistic TypeScript source file with secrets
  - [ ] `testdata/secrets/mixed_file.py` - Realistic Python source file with secrets
  - [ ] `testdata/secrets/config.env` - Realistic .env file with multiple secret types
  - [ ] `testdata/secrets/docker-compose.yml` - Docker Compose with embedded credentials
- [ ] **False positive fixtures** (files that should NOT trigger redaction):
  - [ ] `testdata/secrets/false_positives.txt` - Strings that resemble secrets but are not:
    - Stripe test keys (`sk_test_...`)
    - AWS example key (`AKIAIOSFODNN7EXAMPLE`)
    - Placeholder values (`YOUR_API_KEY_HERE`, `<insert-token>`, `xxx`, `TODO`)
    - Hash outputs in documentation (SHA256 in README)
    - Base64-encoded non-secret data (e.g., encoded image data)
    - UUIDs
    - Long hex color codes repeated
  - [ ] `testdata/secrets/test_fixtures/sample_test.go` - Test file that references secrets in test assertions (should be excluded by path patterns)
- [ ] **Expected results files**: Each fixture file has a corresponding `.expected` file defining:
  - Number of expected redactions
  - Line numbers of expected redactions
  - Expected secret types for each redaction
- [ ] **Golden test runner**: A Go test function that:
  1. Reads each fixture file
  2. Runs it through the full `StreamRedactor`
  3. Compares the redacted output against the expected results
  4. Fails with a clear diff if any expected redaction is missed or any unexpected redaction occurs
- [ ] **Fuzz tests** for the redactor:
  - `FuzzRedactRandomContent`: Generate random strings and verify the redactor never panics and always returns valid UTF-8
  - `FuzzRedactEnvFile`: Generate random `.env`-style content (KEY=VALUE lines) and verify the redactor handles all inputs gracefully
  - `FuzzRedactHighEntropy`: Generate high-entropy random strings and verify entropy analyzer produces consistent results
- [ ] **Zero known misses for high-confidence patterns**: The golden test suite must pass with 100% detection of high-confidence patterns
- [ ] **False positive rate < 5%**: Of all flagged items in the false_positives.txt file, fewer than 5% should be false alarms
- [ ] **Performance regression test**: Benchmark the full corpus processing and assert it completes within a time bound (e.g., all fixtures in under 500ms)

## Technical Notes

- **Synthetic secrets**: ALL secrets in the corpus must be clearly synthetic. Use patterns like:
  - AWS: `AKIAIOSFODNN7EXAMPLE`, `AKIAI44QH8DHBEXAMPLE`
  - GitHub: `ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef12`
  - Stripe: `sk_test_FAKE00000000000000000000000`
  - Private keys: Generate using `openssl genrsa 512` (small, insecure key that is obviously not real)
  - Connection strings: `postgres://testuser:testpass123@localhost:5432/testdb`
  - JWTs: Encode test payloads with a known key, clearly labeled as test tokens
- **Expected results format** (`.expected` JSON):
  ```json
  {
    "expected_redactions": [
      {"line": 5, "secret_type": "aws_access_key", "confidence": "high"},
      {"line": 12, "secret_type": "connection_string", "confidence": "medium"}
    ],
    "expected_false_negative_count": 0,
    "expected_false_positive_count": 0
  }
  ```
- **Golden test implementation**: Use Go's `testing` package with `testdata` embedding. The test reads all `*.txt`, `*.go`, `*.ts`, `*.py`, `*.env`, `*.yml` files in `testdata/secrets/`, processes each through the redactor, and compares against the `.expected` file. Use `github.com/stretchr/testify/assert` for clear assertion messages.
- **Fuzz testing**: Use Go 1.18+ native fuzzing (`func FuzzXxx(f *testing.F)`). Seed the fuzzer with real-world examples from the corpus. Key invariants to test:
  - Redactor never panics on any input
  - Output is always valid UTF-8
  - Output length is >= input length minus max-secret-length * num-redactions (rough bound)
  - Redaction markers are well-formed: `[REDACTED:<type>]` where type is non-empty
- **Updating the corpus**: Document the process for adding new patterns:
  1. Add a new fixture entry to the appropriate category file
  2. Add the expected result to the `.expected` file
  3. Run the golden test -- it should fail
  4. Implement the pattern in `patterns.go`
  5. Run the golden test again -- it should pass
- **CI integration**: The golden tests should run as part of `go test ./internal/security/...`. No special CI configuration needed.
- Reference: PRD Section 9.1 (redaction tests), Section 9.4 (fuzz tests), `testdata/secrets/` directory in PRD Section 6.2

## Files to Create/Modify

- `testdata/secrets/aws_keys.txt` + `.expected`
- `testdata/secrets/github_tokens.txt` + `.expected`
- `testdata/secrets/stripe_keys.txt` + `.expected`
- `testdata/secrets/openai_keys.txt` + `.expected`
- `testdata/secrets/private_keys.txt` + `.expected`
- `testdata/secrets/connection_strings.txt` + `.expected`
- `testdata/secrets/jwt_tokens.txt` + `.expected`
- `testdata/secrets/cloud_credentials.txt` + `.expected`
- `testdata/secrets/generic_assignments.txt` + `.expected`
- `testdata/secrets/mixed_file.go` + `.expected`
- `testdata/secrets/mixed_file.ts` + `.expected`
- `testdata/secrets/mixed_file.py` + `.expected`
- `testdata/secrets/config.env` + `.expected`
- `testdata/secrets/docker-compose.yml` + `.expected`
- `testdata/secrets/false_positives.txt` + `.expected`
- `testdata/secrets/test_fixtures/sample_test.go`
- `testdata/secrets/README.md` - Documentation on corpus structure and how to add new entries
- `internal/security/golden_test.go` - Golden test runner
- `internal/security/fuzz_test.go` - Fuzz tests for redactor
- `internal/security/bench_test.go` - Performance regression benchmarks for full corpus

## Testing Requirements

This task IS the testing task. The deliverables are:

- **Golden tests**: Automated comparison of redacted output vs expected results for every fixture file
- **Fuzz tests**: At minimum 3 fuzz functions covering random content, .env content, and high-entropy strings
- **Performance benchmarks**: Full corpus processing benchmark with time assertion
- **False positive measurement**: Count false positives in the false_positives.txt file and assert < 5%
- **Coverage target**: The golden tests should exercise every pattern in `patterns.go` at least once (verify by checking that every rule ID appears in at least one `.expected` file)
- **Documentation**: `testdata/secrets/README.md` explains the corpus structure, how to add new test cases, and the expected results format
