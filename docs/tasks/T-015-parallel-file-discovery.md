# T-015: Parallel File Discovery Engine (Walker with errgroup)

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-003, T-004, T-007, T-011, T-012, T-013, T-014
**Phase:** 1 - Foundation

---

## Description

Implement the core file discovery engine in `internal/discovery/walker.go` that ties together all filtering components (gitignore, harvxignore, defaults, binary detection, pattern filtering, git-tracked-only, symlink handling, size limits) into a parallel directory walker using `filepath.WalkDir` with `x/sync/errgroup`. This is the workhorse of the discovery pipeline stage, producing a sorted slice of `FileDescriptor` structs ready for downstream processing. The walker reads file contents in parallel using bounded concurrency.

## User Story

As a developer, I want Harvx to discover and read all relevant files in my project quickly and correctly, using multiple CPU cores, so that context generation completes in under 1 second for typical repositories.

## Acceptance Criteria

- [ ] `golang.org/x/sync` is added to `go.mod`
- [ ] `internal/discovery/walker.go` defines a `Walker` type and `Walk(ctx context.Context, cfg WalkerConfig) (*pipeline.DiscoveryResult, error)` method
- [ ] `WalkerConfig` accepts:
  - `Root` (string): target directory
  - `GitignoreMatcher` (Ignorer): from T-011
  - `HarvxignoreMatcher` (Ignorer): from T-012
  - `DefaultIgnorer` (Ignorer): from T-012
  - `PatternFilter` (*PatternFilter): from T-014
  - `GitTrackedOnly` (bool): from T-014
  - `SkipLargeFiles` (int64): bytes threshold from T-013
  - `Concurrency` (int): max parallel workers (default: `runtime.NumCPU()`)
- [ ] Discovery phase (walking):
  1. Uses `filepath.WalkDir` to traverse the directory tree
  2. For each entry, checks all ignore sources via `CompositeIgnorer`
  3. If `--git-tracked-only`, also checks against the git-tracked whitelist
  4. Skips directories early (`fs.SkipDir`) when the directory itself is ignored
  5. Detects and skips binary files (T-013)
  6. Detects and skips large files (T-013)
  7. Applies pattern filter (T-014)
  8. Handles symlinks safely (T-014)
  9. Creates `FileDescriptor` for each passing file with path, size, and metadata
- [ ] Content loading phase (parallel):
  1. Uses `errgroup.WithContext()` with `errgroup.SetLimit(runtime.NumCPU())`
  2. Each worker reads a file's content using buffered I/O
  3. Content is stored in `FileDescriptor.Content`
  4. Per-file errors are captured in `FileDescriptor.Error` (not fatal to the overall walk)
- [ ] Results are sorted by path (alphabetically) for deterministic output
- [ ] Returns a `DiscoveryResult` with:
  - `Files []FileDescriptor` -- all discovered files
  - `TotalFound int` -- total files found before filtering
  - `TotalSkipped int` -- total files skipped (with breakdown by reason)
  - `Errors []error` -- non-fatal per-file errors
- [ ] If more than 0 files had errors but some succeeded, this is a partial success scenario
- [ ] `context.Context` cancellation stops the walk and all workers promptly
- [ ] Logging: debug-level logs for each skipped file (with reason), info-level for summary stats
- [ ] Performance: processes 1,000 files in < 1 second on typical hardware
- [ ] Unit tests with a synthetic test repo under `testdata/`
- [ ] Integration test with the `testdata/sample-repo/` fixture

## Technical Notes

- The errgroup pattern for bounded concurrency:
  ```go
  g, ctx := errgroup.WithContext(ctx)
  g.SetLimit(cfg.Concurrency)

  // Phase 1: Walk and collect file descriptors (single goroutine)
  var files []*pipeline.FileDescriptor
  err := filepath.WalkDir(cfg.Root, func(path string, d fs.DirEntry, err error) error {
      // ... filtering logic ...
      files = append(files, &pipeline.FileDescriptor{Path: relPath, AbsPath: absPath, Size: size})
      return nil
  })

  // Phase 2: Read contents in parallel
  for _, fd := range files {
      fd := fd // capture
      g.Go(func() error {
          content, err := readFile(ctx, fd.AbsPath)
          if err != nil {
              fd.Error = err
              return nil // non-fatal: capture error, continue
          }
          fd.Content = content
          return nil
      })
  }

  if err := g.Wait(); err != nil {
      return nil, err
  }
  ```
- Per PRD Section 5.1: "Processes files in parallel using `x/sync/errgroup` with bounded concurrency for maximum throughput."
- Per PRD Section 6.4: "Use `x/sync/errgroup.WithContext()` with `SetLimit(runtime.NumCPU())` for bounded parallelism with proper error propagation and cancellation via `context.Context`."
- Per PRD Section 5.1: "Maintain deterministic file ordering (sorted by path) for reproducible output."
- Per PRD Section 5.1: "File reading should use buffered I/O with configurable max file size limit."
- The walker should NOT do relevance sorting, token counting, or content processing -- those are separate pipeline stages. The walker only discovers and reads files.
- For large repos, the two-phase approach (walk then read) is more efficient than reading during the walk because the walk can skip entire directories without any I/O.
- Create a representative `testdata/sample-repo/` with: Go files, TypeScript files, a nested `.gitignore`, a `.harvxignore`, a binary file, a large file, a symlink, and a `node_modules/` directory.
- Reference: PRD Sections 5.1, 6.3, 6.4

## Files to Create/Modify

- `go.mod` / `go.sum` - Add `golang.org/x/sync`
- `internal/discovery/walker.go` - Main walker implementation
- `internal/discovery/walker_test.go` - Unit tests
- `testdata/sample-repo/` - Representative test repository:
  - `testdata/sample-repo/main.go`
  - `testdata/sample-repo/README.md`
  - `testdata/sample-repo/src/app.ts`
  - `testdata/sample-repo/src/utils.ts`
  - `testdata/sample-repo/src/test.spec.ts`
  - `testdata/sample-repo/.gitignore` (ignores `dist/`, `node_modules/`)
  - `testdata/sample-repo/.harvxignore` (ignores `docs/internal/`)
  - `testdata/sample-repo/dist/bundle.js` (should be ignored)
  - `testdata/sample-repo/node_modules/pkg/index.js` (should be ignored)
  - `testdata/sample-repo/docs/internal/notes.md` (should be harvxignored)
  - `testdata/sample-repo/image.png` (binary, should be detected)

## Testing Requirements

- Unit test: walker discovers all text files in sample-repo
- Unit test: `.gitignore` patterns are respected (dist/, node_modules/ skipped)
- Unit test: `.harvxignore` patterns are respected
- Unit test: default ignore patterns are applied (`.git/` skipped)
- Unit test: binary files are detected and skipped
- Unit test: large files exceeding threshold are skipped
- Unit test: `--git-tracked-only` mode restricts to git index
- Unit test: include patterns filter correctly
- Unit test: exclude patterns filter correctly
- Unit test: extension filters work
- Unit test: results are sorted alphabetically by path
- Unit test: empty directory returns empty result (not error)
- Unit test: non-existent directory returns error
- Unit test: context cancellation stops walk promptly
- Unit test: per-file read errors are captured (not fatal)
- Unit test: partial success scenario (some files error, some succeed)
- Benchmark: walk 1,000 files in < 1 second
- Benchmark: walk 10,000 files in < 3 seconds (with mock file reads)
- Integration test: end-to-end walk of sample-repo produces expected file list
