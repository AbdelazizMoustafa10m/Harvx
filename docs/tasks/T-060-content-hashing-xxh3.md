# T-060: Content Hashing with XXH3

**Priority:** Should Have
**Effort:** Small (2-4hrs)
**Dependencies:** T-059 (State Snapshot Types)
**Phase:** 4 - State & Diff

---

## Description

Implement the content hashing layer that computes XXH3 64-bit hashes for file contents. This is the core mechanism that enables efficient change detection between runs -- two files with the same hash are considered identical, avoiding expensive byte-by-byte comparison. The hasher is also used by the output rendering layer (PRD Section 5.7) for deterministic content fingerprinting. This task defines a `Hasher` interface and the concrete XXH3 implementation using `zeebo/xxh3`.

**Important technical note:** The PRD references `cespare/xxhash` for XXH3, but `cespare/xxhash` v2 implements XXH64, not XXH3. The correct Go package for XXH3 is `zeebo/xxh3` (v1.0.2). This task uses `zeebo/xxh3` accordingly.

## User Story

As a developer, I want Harvx to quickly detect which files have changed since the last run so that I get differential output in seconds, even in large repositories.

## Acceptance Criteria

- [ ] `internal/diff/hasher.go` defines a `Hasher` interface with `HashBytes(data []byte) uint64` and `HashString(s string) uint64` methods
- [ ] `internal/diff/xxh3.go` implements the `XXH3Hasher` struct that satisfies the `Hasher` interface using `zeebo/xxh3`
- [ ] `HashFile(path string) (uint64, error)` reads a file and returns its XXH3 hash, using buffered I/O (not reading the entire file into memory at once)
- [ ] `HashFileDescriptors(fds []FileDescriptor) error` hashes a slice of FileDescriptors in place, populating their `ContentHash` field (from `internal/pipeline/types.go`)
- [ ] Hashing is deterministic: same content always produces the same hash
- [ ] Empty file produces a valid, non-zero hash (XXH3 of empty input)
- [ ] Large files (100MB+) can be hashed without excessive memory allocation -- uses streaming `io.Reader` approach via `zeebo/xxh3.Hasher`
- [ ] `go.mod` updated with `github.com/zeebo/xxh3` v1.0.2 dependency
- [ ] Unit tests achieve 95%+ coverage
- [ ] Benchmark test demonstrates hashing throughput (should exceed 5 GB/s on modern hardware)

## Technical Notes

- **Package choice:** Use `github.com/zeebo/xxh3` v1.0.2 (latest stable). This is the correct XXH3 implementation for Go. The PRD mentions `cespare/xxhash` but that library implements XXH64, not XXH3. Both are fast non-cryptographic hashes, but XXH3 is the newer algorithm with better performance on small inputs. If `cespare/xxhash` is already used elsewhere in the codebase (e.g., for output content hashing in the renderer), that is fine -- this task specifically implements the state hashing layer.
- **API surface of zeebo/xxh3:**
  - `xxh3.Hash(b []byte) uint64` -- one-shot hash of a byte slice
  - `xxh3.HashString(s string) uint64` -- one-shot hash of a string (no allocation)
  - `xxh3.New()` returns a `*xxh3.Hasher` that implements `hash.Hash` for streaming use with `io.Copy`
- **Streaming approach for large files:**
  ```go
  h := xxh3.New()
  f, _ := os.Open(path)
  io.Copy(h, f)  // streams 32KB chunks, no full-file allocation
  hash := h.Sum64()
  ```
- The `Hasher` interface allows swapping implementations for testing (mock hasher) or future algorithm changes
- Hash values are `uint64` in Go and serialized as hex strings in JSON (handled by T-059's FileState marshaling)
- This hasher is called during the Content Loading pipeline stage (PRD Section 6.3) and during state snapshot creation
- Buffer size for file reading should use `io.Copy` which defaults to 32KB chunks -- this is optimal for XXH3

## Files to Create/Modify

- `internal/diff/hasher.go` - Hasher interface definition
- `internal/diff/xxh3.go` - XXH3Hasher implementation using zeebo/xxh3
- `internal/diff/xxh3_test.go` - Unit tests and benchmarks
- `go.mod` - Add `github.com/zeebo/xxh3` dependency

## Testing Requirements

- Unit test: Hash a known string and verify against reference XXH3 value
- Unit test: Hash empty input and verify it produces a consistent non-zero value
- Unit test: Hash the same content twice and verify identical results (determinism)
- Unit test: Hash two different strings and verify different results (collision resistance is not guaranteed, but trivially different inputs should differ)
- Unit test: `HashFile` on a test fixture file returns expected hash
- Unit test: `HashFile` on non-existent file returns appropriate error
- Unit test: `HashFile` on a large file (generate in-test) uses bounded memory
- Benchmark: `BenchmarkHashBytes` with 1KB, 64KB, 1MB inputs
- Benchmark: `BenchmarkHashFile` with a temp file

## References

- [zeebo/xxh3 GitHub](https://github.com/zeebo/xxh3) -- XXH3 algorithm in Go, v1.0.2
- [zeebo/xxh3 pkg.go.dev](https://pkg.go.dev/github.com/zeebo/xxh3) -- API documentation
- [cespare/xxhash](https://github.com/cespare/xxhash) -- XXH64 (not XXH3), referenced in PRD but not used here
- PRD Section 5.8 (Content hashing: XXH3 for speed)
- PRD Section 5.7 (Output content hash for deterministic fingerprinting)
