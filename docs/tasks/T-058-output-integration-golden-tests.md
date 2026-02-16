# T-058: Output Pipeline Integration and Golden Tests

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-051, T-052, T-053, T-054, T-055, T-056, T-057
**Phase:** 4 - Output & Rendering

---

## Description

Wire together all output components (tree builder, Markdown renderer, XML renderer, content hasher, output writer, splitter, metadata generator) into a cohesive output pipeline stage that the main `internal/pipeline` orchestrator can call. Build comprehensive golden tests that exercise the full output rendering path end-to-end with realistic sample data, producing deterministic expected outputs for both Markdown and XML formats. This task validates that all output components work together correctly and establishes regression protection for the output format.

## User Story

As a developer contributing to Harvx, I want comprehensive integration tests for the output pipeline so that changes to any output component are caught before they break the output format that users and pipelines depend on.

## Acceptance Criteria

- [ ] `RenderOutput(ctx context.Context, cfg OutputConfig, files []FileDescriptor) (*OutputResult, error)` function in `internal/output/pipeline.go` orchestrates the full output flow
- [ ] The pipeline: (1) builds tree from file paths, (2) computes content hash, (3) assembles `RenderData`, (4) dispatches to correct renderer, (5) writes output (file or stdout), (6) optionally splits, (7) optionally generates metadata sidecar
- [ ] `OutputConfig` aggregates all output-related settings: format, target, output path, stdout flag, split size, show line numbers, show metadata, tree depth, show tree metadata
- [ ] `FileDescriptor` (from `internal/pipeline/types.go`) is the input -- the output pipeline does not do file reading, redaction, or token counting
- [ ] **Golden test: Markdown basic** -- 5-file sample repo with tier 0-2 files, default settings, verified byte-for-byte against expected output
- [ ] **Golden test: Markdown with line numbers** -- same sample with `--line-numbers`, verified
- [ ] **Golden test: XML basic** -- same sample repo rendered as XML, verified
- [ ] **Golden test: Markdown with diff** -- includes change summary section
- [ ] **Golden test: split output** -- sample repo split into 2 parts, each part verified
- [ ] **Golden test: metadata sidecar** -- verify JSON structure and field correctness
- [ ] Golden test update mechanism: provide a `-update` flag (or `HARVX_UPDATE_GOLDEN=1` env) that regenerates expected files
- [ ] All golden tests are in `testdata/expected-output/` and tracked in git
- [ ] Integration test verifying Markdown and XML produce same content hash for same input
- [ ] Integration test verifying stdout mode produces identical output to file mode
- [ ] Test helper: `testutil.LoadGoldenFile(t, path)` and `testutil.CompareGolden(t, got, goldenPath)` functions

## Technical Notes

- **Golden test pattern in Go**: Use `testutil.CompareGolden(t, gotBytes, "testdata/expected-output/markdown-basic.md")`. On mismatch, diff the output and fail. When `HARVX_UPDATE_GOLDEN=1` is set, overwrite the expected file instead of failing.
- **Sample data**: Create a realistic but small `testdata/sample-repo/` fixture with:
  - `go.mod` (tier 0, config)
  - `README.md` (tier 4, docs)
  - `cmd/main.go` (tier 1, source)
  - `internal/handler.go` (tier 1, source)
  - `internal/handler_test.go` (tier 3, test)
  - Each file should have known content so token counts are deterministic.
- **Determinism**: All timestamps in golden tests must be fixed (pass a known time, do not use `time.Now()`). Content hashes are derived from file content, so they are naturally deterministic.
- **Test organization**: Put integration tests in `internal/output/integration_test.go` with build tag `//go:build integration` or in a `_test.go` file that runs with the regular test suite (since these are fast, golden-file tests).
- **Regression safety**: Any change to output format must update golden files. The CI pipeline should fail if golden files are stale.
- Reference: PRD Sections 5.7, 5.12, 9.2 (golden tests)

## Files to Create/Modify

- `internal/output/pipeline.go` - `RenderOutput` orchestration function, `OutputConfig` struct
- `internal/output/pipeline_test.go` - Integration tests
- `internal/output/testutil_test.go` - Golden test helpers (`CompareGolden`, `LoadGoldenFile`)
- `testdata/sample-repo/go.mod` - Sample fixture file
- `testdata/sample-repo/README.md` - Sample fixture file
- `testdata/sample-repo/cmd/main.go` - Sample fixture file
- `testdata/sample-repo/internal/handler.go` - Sample fixture file
- `testdata/sample-repo/internal/handler_test.go` - Sample fixture file
- `testdata/expected-output/markdown-basic.md` - Golden expected output (may update from T-052)
- `testdata/expected-output/markdown-line-numbers.md` - Golden expected output (may update from T-052)
- `testdata/expected-output/xml-basic.xml` - Golden expected output (may update from T-053)
- `testdata/expected-output/markdown-with-diff.md` - Golden expected output
- `testdata/expected-output/split-part-001.md` - Golden expected output for split
- `testdata/expected-output/split-part-002.md` - Golden expected output for split
- `testdata/expected-output/metadata-sidecar.json` - Golden expected metadata

## Testing Requirements

- Golden test: Markdown basic output matches expected file byte-for-byte
- Golden test: Markdown with line numbers matches expected file
- Golden test: XML basic output matches expected file
- Golden test: Markdown with diff section matches expected file
- Golden test: split Part 1 and Part 2 match expected files
- Golden test: metadata JSON matches expected file (with tolerance for generation_time_ms)
- Integration test: Markdown and XML produce identical content hashes for same file set
- Integration test: stdout output equals file output (bytes comparison)
- Integration test: `RenderOutput` returns correct `OutputResult` with path, hash, token count, bytes written
- Integration test: pipeline handles zero files gracefully (produces valid but empty output)
- Integration test: pipeline handles single file correctly
- Integration test: pipeline with all options enabled (line numbers + metadata + split) works correctly
- Regression test: changing file content changes the content hash
- Regression test: adding a file changes the tree and content hash