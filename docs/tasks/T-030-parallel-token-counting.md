# T-030: Parallel Per-File Token Counting

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-029
**Phase:** 2 - Intelligence (Relevance & Tokens)

---

## Description

Implement parallel token counting that runs concurrently with file reading, counting tokens for each file's content as it is loaded rather than as a separate pass. Each `FileDescriptor` gets its `TokenCount` field populated. This task also computes the total token count across all files and accounts for overhead from the summary/header section that itself consumes tokens.

## User Story

As a developer, I want Harvx to count tokens efficiently using parallel processing so that even large codebases get accurate token counts in under a second.

## Acceptance Criteria

- [ ] `TokenCounter` struct wraps a `Tokenizer` and provides methods for counting files
- [ ] `CountFile(fd *FileDescriptor)` populates `fd.TokenCount` from `fd.Content`
- [ ] `CountFiles(ctx context.Context, files []*FileDescriptor) (int, error)` counts all files in parallel using `errgroup` with bounded concurrency (`runtime.NumCPU()`)
- [ ] Returns total token count across all files
- [ ] Token counting runs on the **processed** content (after redaction, after compression if applicable) -- not the raw content
- [ ] Accounts for summary section overhead: the output header, file tree, and per-file headers consume tokens. Provide `EstimateOverhead(fileCount int, treeSize int) int` that estimates this overhead
- [ ] Overhead estimate is subtracted from available budget before file content budgeting (in T-031)
- [ ] Supports `context.Context` for cancellation
- [ ] Goroutine-safe: multiple files counted concurrently
- [ ] Unit tests achieve 90%+ coverage

## Technical Notes

- Create in `internal/tokenizer/counter.go`
- Uses `x/sync/errgroup` with `SetLimit(runtime.NumCPU())` for bounded parallelism
- The key insight from the PRD: "Token counting runs in parallel with file reading (count as files are loaded, not as a separate pass)." In practice, this means the content loading pipeline should call `CountFile` on each `FileDescriptor` as soon as its `Content` field is populated.
- For the overhead estimate, use the `none` estimator (char/4) even when using a real tokenizer, because the summary section is generated after counting. A conservative estimate is acceptable:
  - Per-file header: ~20-30 tokens (path, size, tier label, code fence)
  - File tree line: ~5-10 tokens per file
  - Output header: ~100-200 tokens (project name, metadata, etc.)
  - Formula: `overhead = 200 + (fileCount * 35)`

### Integration Point

The `CountFiles` function should be called after content loading (and after redaction/compression) but before budget enforcement. The pipeline flow is:

```
Discovery -> Relevance Sort -> Content Loading (+redaction/compression) -> Token Counting -> Budget Enforcement -> Output
```

In practice, token counting can be integrated into the content loading phase: as each file's content is loaded, its tokens are counted immediately. This avoids a second pass over all files.

### Dependencies & Versions

| Package/Library | Version | Purpose |
|-----------------|---------|---------|
| golang.org/x/sync | latest | errgroup for parallel counting |

## Files to Create/Modify

- `internal/tokenizer/counter.go` - TokenCounter struct, CountFile(), CountFiles(), EstimateOverhead()
- `internal/tokenizer/counter_test.go` - Unit tests

## Testing Requirements

- Unit test: CountFile populates TokenCount on a FileDescriptor
- Unit test: CountFile with empty content sets TokenCount to 0
- Unit test: CountFiles with 5 files returns correct total
- Unit test: CountFiles with 0 files returns 0
- Unit test: CountFiles respects context cancellation (cancel after 2 files, verify error)
- Unit test: EstimateOverhead returns reasonable values (e.g., 10 files -> ~550 tokens)
- Unit test: EstimateOverhead(0, 0) returns base overhead (~200)
- Benchmark: CountFiles on 1K FileDescriptors with 1KB content each
- Test that TokenCount reflects processed content (simulate a file with content "hello [REDACTED:key]" -- count tokens on that, not original)