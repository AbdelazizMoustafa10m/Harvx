# T-092: Integration Test Suite Against Real OSS Repos

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** Core pipeline complete (discovery, relevance, tokenizer, redaction, compression, output)
**Phase:** 6 - Polish & Distribution

---

## Description

Create an integration test suite that exercises the full Harvx pipeline against real open-source repositories, verifying end-to-end correctness across different languages, project structures, and scales. Tests cover shell script invocation, pipe chains, exit code handling, environment variable overrides, and output format validation. This catches integration issues that unit tests miss, especially around gitignore edge cases, binary detection, and large-scale file processing.

## User Story

As a developer releasing Harvx, I want confidence that the tool works correctly on real-world codebases so that users don't encounter failures on common project structures.

## Acceptance Criteria

- [ ] Integration test framework in `tests/integration/` (separate from unit tests)
- [ ] Tests clone or use cached copies of real OSS repos:
  - A Go CLI project (~500 files) -- e.g., a small cobra-based tool
  - A TypeScript/Next.js project (~2K files) -- e.g., a moderately sized Next.js app
  - A Python project (~1K files) -- e.g., a Django or FastAPI project
  - A monorepo with multiple packages (~3K files)
- [ ] Test repos are cached in CI to avoid re-cloning on every run
- [ ] End-to-end test scenarios:
  - `harvx` with default profile produces valid Markdown output
  - `harvx --format xml --target claude` produces valid XML output
  - `harvx --compress` produces compressed output with expected token reduction
  - `harvx --fail-on-redaction` exits with code 1 if repo contains test secrets
  - `harvx preview --json` produces valid JSON metadata
  - `harvx --stdout | wc -l` produces non-empty output on stdout
  - `harvx --max-tokens 10000` respects token budget (output tokens <= budget)
  - `harvx --git-tracked-only` produces output with only tracked files
  - `HARVX_MAX_TOKENS=5000 harvx` respects environment variable override
- [ ] Shell invocation tests (run `harvx` as a subprocess):
  - Exit code 0 on success
  - Exit code 1 on `--fail-on-redaction` with secrets
  - Exit code 2 on partial failure (some files unreadable)
  - Stderr contains progress/log output, stdout is clean when using `--stdout`
- [ ] Output validation:
  - Markdown output contains expected sections (header, file summary, directory tree, file contents)
  - XML output is well-formed (parseable by `encoding/xml`)
  - Token count in metadata matches actual tokenization of output
  - Content hash is deterministic (run twice, same hash)
- [ ] Pipe chain test: `harvx --stdout | head -100` works without broken pipe errors
- [ ] Large file handling: repos with files > 1MB are correctly handled (skipped or included per config)
- [ ] Tests tagged with `//go:build integration` to exclude from regular test runs
- [ ] `make test-integration` target runs the integration suite
- [ ] CI runs integration tests on a schedule (weekly) or on release branches

## Technical Notes

- Use `os/exec.Command` to invoke `harvx` as a subprocess for true end-to-end testing.
- Build the binary first: `go build -o ./bin/harvx ./cmd/harvx/`, then invoke `./bin/harvx` in tests.
- For repo caching in CI, use GitHub Actions cache with a hash of the test repo list as the key.
- Alternatively, use small, curated test repos embedded in `testdata/` (but these are less realistic than real OSS repos).
- For the "real OSS repo" approach, create a `tests/integration/repos.go` that defines repo URLs and expected characteristics.
- Consider using `testcontainers` or simple shell scripts for repo setup.
- XML validation: use `encoding/xml.Decoder` to parse and verify well-formedness.
- Deterministic hash test: run pipeline twice with same input, compare content hashes.
- Broken pipe handling: ensure `--stdout` mode handles `SIGPIPE` gracefully (don't panic on write to closed pipe).
- Reference: PRD Section 9.5 (Integration Tests), Section 5.10 (Pipeline Integration)

## Files to Create/Modify

- `tests/integration/setup_test.go` - Test setup (build binary, cache repos)
- `tests/integration/repos.go` - Test repo definitions and caching logic
- `tests/integration/default_profile_test.go` - Default profile tests across repo types
- `tests/integration/format_test.go` - Output format validation tests (markdown, xml)
- `tests/integration/cli_flags_test.go` - CLI flag and env var override tests
- `tests/integration/pipeline_test.go` - End-to-end pipeline tests (exit codes, pipe chains)
- `tests/integration/redaction_test.go` - Redaction integration tests
- `tests/integration/compression_test.go` - Compression integration tests
- `Makefile` - Add `test-integration` target (modify)
- `.github/workflows/ci.yml` - Add integration test job (modify)

## Testing Requirements

- All integration tests pass against the defined OSS repos
- Default profile produces valid output for Go, TypeScript, Python, and monorepo projects
- XML output is well-formed and parseable
- Token budget is respected (output tokens <= `--max-tokens` value)
- Content hash is deterministic across runs
- Exit codes are correct for all scenarios (0, 1, 2)
- Environment variable overrides work correctly
- `--stdout` produces clean output suitable for piping
- Integration tests complete within 5 minutes for the full suite