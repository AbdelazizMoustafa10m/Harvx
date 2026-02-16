# T-078: Workflow Integration Tests and End-to-End Pipeline Validation

**Priority:** Must Have
**Effort:** Medium (10-14hrs)
**Dependencies:** T-067 (Stdout/Exit Codes), T-068 (JSON Preview), T-069 (Assert-Include), T-070 (Brief), T-071 (Review Slice), T-072 (Slice), T-073 (Workspace), T-075 (Verify)
**Phase:** 5 - Workflows

---

## Description

Create a comprehensive integration test suite that validates the complete workflow pipeline end-to-end: shell script invocation, pipe chains, exit code handling, environment variable overrides, and the full review pipeline flow (brief -> review-slice -> verify). This is the final quality gate for Phase 5 ensuring all workflow commands compose correctly and behave predictably in automation scenarios.

## User Story

As a developer shipping Harvx to production, I want end-to-end integration tests proving that the complete workflow pipeline works correctly in real shell environments so that I can confidently recommend Harvx for CI/CD and multi-agent review pipelines.

## Acceptance Criteria

- [ ] Integration tests cover the complete review pipeline flow:
  ```bash
  harvx brief --profile test -o /tmp/brief.md
  harvx review-slice --base HEAD~1 --head HEAD --profile test -o /tmp/slice.md
  harvx verify --path src/main.go
  ```
- [ ] Shell script invocation tests verify:
  - `harvx --stdout | wc -l` produces valid piped output with non-zero line count
  - `harvx preview --json | jq .total_files` returns a valid number
  - Exit code propagation: `harvx --fail-on-redaction; echo $?` returns expected code
  - `harvx brief --stdout | harvx review-slice --stdin` (if applicable) or composition via files
- [ ] Environment variable override tests verify:
  - `HARVX_PROFILE=test harvx brief` uses the correct profile
  - `HARVX_MAX_TOKENS=5000 harvx brief` respects the token limit
  - `HARVX_FORMAT=xml harvx brief` produces XML output
  - `HARVX_STDOUT=true harvx brief` sends output to stdout
  - `HARVX_LOG_FORMAT=json harvx brief 2>/tmp/log.json` produces JSON logs on stderr
- [ ] Exit code tests verify all documented codes:
  - Code 0: Successful generation
  - Code 1: Fatal error (invalid profile, invalid git ref, assert-include failure)
  - Code 2: Partial success (some files failed)
- [ ] `--assert-include` integration tests:
  - Passes when critical files exist
  - Fails with exit code 1 when critical files are missing
  - Error message includes the failing pattern
- [ ] Determinism tests: running the same command twice produces identical content hashes
- [ ] Output format tests: `--target claude` produces valid XML, `--format markdown` produces valid Markdown
- [ ] Workspace command integration test: reads `.harvx/workspace.toml` and produces output
- [ ] All tests use the `testdata/sample-repo/` fixture directory with a known file set
- [ ] Tests run in CI (no external network calls, no user interaction)

## Technical Notes

- Use Go's `os/exec` package for shell invocation tests
- Use `testing.T` subtests for organized test grouping
- Create a comprehensive test fixture in `testdata/sample-repo/` that includes:
  - Go source files with imports (for neighbor discovery testing)
  - A `Makefile` with targets
  - A `README.md`
  - A `.harvx/workspace.toml` with test repos
  - A `harvx.toml` with test profiles
  - Files with mock secrets (for redaction testing)
  - A git repository (initialize in test setup with `git init`, `git add`, `git commit`)
- For git-dependent tests (`review-slice`), create a temporary git repo in the test with at least two commits
- Test execution should be parallelizable where possible (`t.Parallel()`)
- Consider using `testscript` (`github.com/rogpeppe/go-internal/testscript`) for shell-like test scripts
- Integration tests should be tagged `//go:build integration` so they can be run separately from unit tests
- Set a timeout on all integration tests (30 seconds max per test)
- Reference: PRD Sections 5.10 (Integration testing), 9.5 (Integration Tests)

## Files to Create/Modify

- `tests/integration/workflow_test.go` - Main integration test file
- `tests/integration/pipeline_test.go` - Pipeline composition tests
- `tests/integration/env_test.go` - Environment variable override tests
- `tests/integration/exitcode_test.go` - Exit code validation tests
- `tests/integration/determinism_test.go` - Output determinism tests
- `tests/integration/testdata/` - Integration test fixtures (symlink or copy of testdata/)
- `tests/integration/setup_test.go` - Test helpers (temp git repo creation, fixture loading)
- `testdata/sample-repo/harvx.toml` - Test profile configuration
- `testdata/sample-repo/.harvx/workspace.toml` - Test workspace configuration
- `testdata/sample-repo/src/main.go` - Test Go source file
- `testdata/sample-repo/src/auth/middleware.go` - Test file for neighbor discovery
- `testdata/sample-repo/src/auth/middleware_test.go` - Test file for test discovery
- `testdata/sample-repo/Makefile` - Test Makefile for brief extraction
- `testdata/sample-repo/README.md` - Test README for brief extraction

## Testing Requirements

- Integration test: Full pipeline (brief -> review-slice -> verify) completes without error
- Integration test: Stdout piping produces valid output
- Integration test: JSON preview output is valid JSON parseable by jq
- Integration test: All HARVX_ environment variables work correctly
- Integration test: All exit codes match specification
- Integration test: Assert-include catches missing files
- Integration test: Deterministic output (content hash matches across runs)
- Integration test: XML output is well-formed XML
- Integration test: Workspace command produces output from fixture config
- Integration test: Review-slice with git refs produces correct changed file set
- Integration test: Slice with --path produces scoped output
- Performance test: Brief generation completes in under 5 seconds for sample repo
- Performance test: Preview --json completes in under 2 seconds for sample repo