# T-057: Metadata JSON Sidecar Generation

**Priority:** Should Have
**Effort:** Small (3-4hrs)
**Dependencies:** T-054, T-055
**Phase:** 4 - Output & Rendering

---

## Description

Implement the `--output-metadata` flag that generates a `.meta.json` sidecar file alongside the main context output. The sidecar contains machine-readable metadata for pipeline consumption: harvx version, profile used, tokenizer encoding, content hash, total token count, per-file statistics (path, tier, token count, redaction count, compressed flag), and aggregate statistics. This enables downstream pipeline tools to make decisions based on output properties without parsing the context file itself.

## User Story

As a developer with a multi-agent review pipeline, I want a machine-readable metadata file alongside the context output so that my orchestration scripts can inspect token counts, file lists, and content hashes without parsing the Markdown or XML.

## Acceptance Criteria

- [ ] `--output-metadata` flag triggers sidecar generation
- [ ] Sidecar file path: `<output-path>.meta.json` (e.g., `harvx-output.md.meta.json`)
- [ ] JSON structure:
  ```json
  {
    "version": "1.0.0",
    "generated_at": "2026-02-16T10:30:00Z",
    "profile": "finvault",
    "tokenizer": "o200k_base",
    "format": "markdown",
    "target": "claude",
    "content_hash": "a1b2c3d4e5f6g7h8",
    "statistics": {
      "total_files": 342,
      "total_tokens": 89420,
      "total_bytes": 1234567,
      "budget_used_percent": 44.7,
      "max_tokens": 200000,
      "files_by_tier": {"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
      "redactions_total": 3,
      "redactions_by_type": {"aws_access_key": 2, "connection_string": 1},
      "compressed_files": 120,
      "generation_time_ms": 850
    },
    "files": [
      {
        "path": "src/main.go",
        "tier": 1,
        "tokens": 340,
        "bytes": 1024,
        "redactions": 0,
        "compressed": false,
        "language": "go"
      }
    ]
  }
  ```
- [ ] All JSON field names use `snake_case` for consistency
- [ ] `files` array is sorted by path (same order as in the output)
- [ ] Generation time is measured and included (milliseconds)
- [ ] Sidecar is written atomically (temp file + rename, same as main output)
- [ ] `MetadataGenerator` in `internal/output/metadata.go` accepts `RenderData` + `OutputResult` and produces the JSON
- [ ] Output is pretty-printed (indented with 2 spaces) for human readability
- [ ] Unit tests achieve >= 90% coverage

## Technical Notes

- **Go structs with JSON tags**: Define `OutputMetadata`, `Statistics`, `FileStats` structs with `json:"field_name"` tags. Use `json.MarshalIndent(data, "", "  ")` for pretty-printing.
- **Integration point**: The `OutputWriter` from T-055 calls the metadata generator after the main output is written (it needs the final content hash and generation time).
- **Timing**: Generation time should be measured from the start of the pipeline (discovery) to the end of output writing. The caller passes this timing data in.
- **Version**: Use a constant `Version` that matches the binary version. For now, hardcode `"1.0.0"` and update when version management is implemented.
- **Budget percent**: `(total_tokens / max_tokens) * 100`, or `null`/omitted if no max_tokens is set.
- Reference: PRD Section 5.7 (--output-metadata, .meta.json)

## Files to Create/Modify

- `internal/output/metadata.go` - `OutputMetadata` struct, `Statistics` struct, `FileStats` struct, `GenerateMetadata` function, `WriteMetadata` function
- `internal/output/metadata_test.go` - Unit tests
- `internal/output/writer.go` - Modify `Write` to call metadata generation when flag is set

## Testing Requirements

- Unit test: generated JSON is valid and parseable
- Unit test: all required fields are present in output
- Unit test: `files` array is sorted by path
- Unit test: `content_hash` matches the hash from rendering
- Unit test: `files_by_tier` counts match input data
- Unit test: `budget_used_percent` is calculated correctly (or omitted when no budget)
- Unit test: sidecar path is `<output>.meta.json`
- Unit test: pretty-printed output has 2-space indentation
- Unit test: empty file list produces valid JSON with zero counts
- Unit test: generation time is a positive integer
- Edge case: output path with multiple dots (e.g., `my.project.output.md`) produces correct sidecar name
- Edge case: no redactions produces `redactions_total: 0` and empty `redactions_by_type`
- Edge case: very large file list (10K+ files) produces valid JSON without truncation