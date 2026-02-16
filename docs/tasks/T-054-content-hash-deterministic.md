# T-054: Content Hashing (XXH3) and Deterministic Output

**Priority:** Must Have
**Effort:** Small (3-4hrs)
**Dependencies:** None
**Phase:** 4 - Output & Rendering

---

## Description

Implement the content hashing module that computes an XXH3 64-bit hash over all included file contents (sorted by path) to produce a deterministic fingerprint for the output. This hash enables LLM prompt caching -- identical codebase state produces identical hashes, letting callers detect when context has not changed. The module also provides incremental hashing for the streaming output pipeline.

## User Story

As a developer with an automated review pipeline, I want deterministic output with a content hash so that I can leverage LLM prompt caching and avoid re-submitting unchanged context.

## Acceptance Criteria

- [ ] `ContentHasher` in `internal/output/hash.go` uses `zeebo/xxh3` package for XXH3 64-bit hashing
- [ ] `ComputeContentHash(files []FileHashEntry) (uint64, error)` hashes all file contents in deterministic order (sorted by relative path)
- [ ] Hash input per file: `path + "\x00" + content` (null byte separator to prevent path/content collision)
- [ ] `NewIncrementalHasher() *IncrementalHasher` provides streaming hash computation via `io.Writer` interface
- [ ] `IncrementalHasher.Write(p []byte) (int, error)` feeds bytes into the running hash
- [ ] `IncrementalHasher.Sum64() uint64` returns the final hash value
- [ ] `FormatHash(h uint64) string` returns lowercase hex representation (16 characters, zero-padded)
- [ ] Same input always produces same hash regardless of platform (endianness-safe)
- [ ] Hash value is included in output header block (rendered by Markdown/XML renderers from T-052/T-053)
- [ ] Unit tests achieve >= 95% coverage

## Technical Notes

- **Package choice**: The PRD specifies `cespare/xxhash` for XXH3, but `cespare/xxhash` actually implements XXH64, not XXH3. Use `zeebo/xxh3` (https://pkg.go.dev/github.com/zeebo/xxh3) which provides a proper XXH3 implementation in pure Go with SIMD optimizations. The `zeebo/xxh3` package provides `xxh3.Hash(b []byte) uint64` and `xxh3.New()` which implements `hash.Hash`.
- **Determinism**: Files MUST be sorted by relative path before hashing. The sort is case-sensitive (byte order) to be platform-independent.
- **Null byte separator**: Using `\x00` between path and content prevents collisions where path suffix matches content prefix.
- **Streaming support**: The `IncrementalHasher` wraps `xxh3.Hasher` and is useful when computing the hash during output writing rather than as a separate pass.
- **Performance**: XXH3 is extremely fast (>10 GB/s on modern hardware). No need for parallelism in the hash computation itself.
- Reference: PRD Section 5.7 (deterministic output, content hash)

## Files to Create/Modify

- `internal/output/hash.go` - `ContentHasher`, `IncrementalHasher`, `ComputeContentHash`, `FormatHash`
- `internal/output/hash_test.go` - Unit tests
- `go.mod` - Add `github.com/zeebo/xxh3` dependency

## Testing Requirements

- Unit test: `ComputeContentHash` with known inputs produces expected hash (regression test with pinned value)
- Unit test: hash changes when any file content changes
- Unit test: hash changes when a file path changes (even if content is same)
- Unit test: hash is stable across multiple calls with same input
- Unit test: file order does not matter (internal sorting ensures determinism)
- Unit test: `FormatHash` produces 16-character zero-padded hex string
- Unit test: `IncrementalHasher` produces same hash as `ComputeContentHash` for equivalent input
- Unit test: empty file list produces a consistent hash (not zero)
- Edge case: files with empty content hash correctly
- Edge case: files with identical content but different paths produce different hashes
- Edge case: file paths with Unicode characters hash consistently