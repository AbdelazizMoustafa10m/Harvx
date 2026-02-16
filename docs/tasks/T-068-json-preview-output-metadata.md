# T-068: JSON Preview Output and Metadata Sidecar

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-066 (Pipeline Library API), T-067 (Stdout/Exit Codes)
**Phase:** 5 - Workflows

---

## Description

Implement `harvx preview --json` for machine-readable pipeline metadata output and `--output-metadata` for generating a `.meta.json` sidecar file alongside context output. These features enable orchestration scripts to make programmatic decisions based on Harvx's analysis (file counts, token budgets, tier breakdowns, redaction counts) without parsing human-readable text.

## User Story

As a developer orchestrating a multi-agent review pipeline, I want to query Harvx for structured metadata (total files, tokens, tiers) in JSON format so that my scripts can dynamically adjust agent parameters based on context size.

## Acceptance Criteria

- [ ] `harvx preview --json` outputs a JSON object to stdout matching this schema:
  ```json
  {
    "total_files": 342,
    "total_tokens": 89420,
    "tokenizer": "o200k_base",
    "tiers": {"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
    "redactions": 3,
    "estimated_time_ms": 850,
    "content_hash": "a1b2c3d4e5f6",
    "profile": "finvault",
    "budget_utilization_percent": 44.7,
    "files_truncated": 0,
    "files_omitted": 12
  }
  ```
- [ ] `--output-metadata` flag generates a `.meta.json` sidecar file alongside the context output file
- [ ] Sidecar filename is derived from output filename: `harvx-output.md` produces `harvx-output.meta.json`
- [ ] Metadata sidecar includes: version, profile, tokenizer, content hash, per-file stats (path, tier, tokens, redacted count), and aggregate statistics
- [ ] JSON output uses Go's `encoding/json` with proper struct tags and `json.MarshalIndent` for readability
- [ ] `preview --json` exits with code 0 and produces valid JSON even when the repo has issues (warnings go to stderr)
- [ ] `preview` (without `--json`) continues to produce human-readable text output to stderr
- [ ] Both `brief` and `review-slice` commands also support `--json` for metadata output
- [ ] Budget reporting in metadata always includes: tokenizer used, total tokens, budget utilization percentage, and whether any files were truncated or omitted
- [ ] Unit tests verify JSON schema compliance and roundtrip marshaling

## Technical Notes

- Define `PreviewResult` and `MetadataSidecar` structs in `internal/pipeline/result.go` with `json` struct tags
- `preview --json` runs the pipeline in discovery+relevance+tokenization mode (skips full content loading and rendering) for speed
- The sidecar per-file stats should include: `path`, `tier`, `tokens`, `size_bytes`, `is_compressed`, `redactions`
- Content hash uses XXH3 (`cespare/xxhash`) formatted as lowercase hex string
- Use `json.MarshalIndent(result, "", "  ")` for human-readable JSON output
- `estimated_time_ms` is based on actual pipeline execution time (not a prediction)
- The `--json` flag on `brief` and `review-slice` outputs metadata about the generated artifact, not the artifact content itself
- Reference: PRD Sections 5.7 (metadata sidecar), 5.10 (JSON preview), 5.11.4 (budget reporting)

## Files to Create/Modify

- `internal/pipeline/result.go` - Add PreviewResult, MetadataSidecar structs
- `internal/output/metadata.go` - Metadata sidecar generation and writing
- `internal/cli/preview.go` - Add `--json` flag handling to preview command
- `internal/output/metadata_test.go` - JSON schema and roundtrip tests
- `internal/cli/preview_test.go` - Preview --json integration tests
- `testdata/expected-output/preview.json` - Golden test for preview JSON output

## Testing Requirements

- Unit test: PreviewResult marshals to expected JSON schema
- Unit test: MetadataSidecar includes all required fields
- Unit test: Sidecar filename derivation from various output filenames
- Unit test: Per-file stats are complete and correctly aggregated
- Unit test: Budget utilization percentage calculation is accurate
- Golden test: `preview --json` output matches expected fixture
- Integration test: `preview --json | jq .total_tokens` returns a valid number
- Edge case: Preview of empty directory returns valid JSON with zero counts
- Edge case: `--output-metadata` without `--output` uses default output path for sidecar naming