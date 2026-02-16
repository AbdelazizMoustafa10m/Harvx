# T-091: Performance Benchmarking Suite

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** T-001 (project structure), core pipeline packages (discovery, tokenizer, compression, output)
**Phase:** 6 - Polish & Distribution

---

## Description

Create a comprehensive performance benchmarking suite that measures Harvx's throughput and resource usage across repository sizes (1K, 10K, 50K files). Benchmarks cover each pipeline stage (discovery, tokenization, compression, output rendering) independently and the full end-to-end pipeline. Results are compared against the PRD's performance SLOs (< 1s for 1K files, < 3s for 10K files) and tracked over time to detect regressions.

## User Story

As a developer contributing to Harvx, I want an automated benchmark suite so that I can detect performance regressions before they reach users and ensure we meet our published performance targets.

## Acceptance Criteria

- [ ] `internal/benchmark/` package contains all benchmark definitions
- [ ] Benchmark fixtures: synthetic repos generated programmatically with 1K, 10K, and 50K files
  - Files have realistic sizes (100B to 50KB), realistic extensions (.go, .ts, .py, .md, .json)
  - Directory depth varies (flat, 3 levels, 10 levels deep)
  - Includes some large files (1MB) to test `--skip-large-files`
- [ ] Fixture generation script: `internal/benchmark/fixtures.go` with `GenerateTestRepo(dir string, fileCount int)` function
- [ ] Go benchmarks using `testing.B` for each pipeline stage:
  - `BenchmarkDiscovery_1K`, `BenchmarkDiscovery_10K`, `BenchmarkDiscovery_50K`
  - `BenchmarkTokenization_1K`, `BenchmarkTokenization_10K`
  - `BenchmarkCompression_1K` (tree-sitter WASM on supported files)
  - `BenchmarkRedaction_1K`, `BenchmarkRedaction_10K`
  - `BenchmarkOutputRendering_1K`, `BenchmarkOutputRendering_10K`
  - `BenchmarkFullPipeline_1K`, `BenchmarkFullPipeline_10K`, `BenchmarkFullPipeline_50K`
- [ ] Each benchmark reports: time (ns/op), memory allocations (allocs/op, bytes/op)
- [ ] SLO verification tests (not benchmarks -- these are `Test` functions that fail if SLO is exceeded):
  - `TestSLO_1KFiles_Under1Second`
  - `TestSLO_10KFiles_Under3Seconds`
- [ ] Memory usage tracking: peak memory during full pipeline on 10K files should be < 500MB
- [ ] `make bench` target runs all benchmarks and outputs results
- [ ] `make bench-compare` target compares current results against a baseline file using `benchstat`
- [ ] Baseline results stored in `testdata/benchmarks/baseline.txt`
- [ ] Benchmark results include P50/P95 latencies via multiple iterations
- [ ] TUI-specific benchmark: token recalculation for 1K included files completes in < 300ms (PRD SLO)

## Technical Notes

- Use Go's built-in `testing.B` for benchmarks. Report allocations with `b.ReportAllocs()`.
- Fixture generation: use `os.MkdirTemp` for isolated test directories. Fill files with realistic content (Go source, TypeScript, Python, Markdown).
- For 50K file benchmarks, generate fixtures once and cache in a temp directory. Use `testing.TB.TempDir()` or `sync.Once` for single-generation.
- Use `benchstat` (golang.org/x/perf/cmd/benchstat) for comparing benchmark runs and computing statistical significance.
- SLO tests should use `testing.T` (not `testing.B`) with a timeout. Run the pipeline once and assert wall-clock time.
- Memory tracking: use `runtime.MemStats` before and after pipeline execution to measure `TotalAlloc` and `Sys`.
- For streaming output verification: ensure `Sys` memory doesn't grow linearly with file count (should plateau due to streaming).
- Tag benchmarks with `//go:build bench` to exclude them from regular `go test ./...` runs. Use `-tags bench` to include.
- Reference: PRD Section 4 (Success Metrics), Section 9.3 (Performance Benchmarks), Section 9.6 (Performance Diagnostics)

## Files to Create/Modify

- `internal/benchmark/fixtures.go` - Synthetic repo generator
- `internal/benchmark/discovery_bench_test.go` - Discovery benchmarks
- `internal/benchmark/tokenizer_bench_test.go` - Tokenization benchmarks
- `internal/benchmark/compression_bench_test.go` - Compression benchmarks
- `internal/benchmark/redaction_bench_test.go` - Redaction benchmarks
- `internal/benchmark/output_bench_test.go` - Output rendering benchmarks
- `internal/benchmark/pipeline_bench_test.go` - Full pipeline benchmarks
- `internal/benchmark/slo_test.go` - SLO verification tests
- `internal/benchmark/memory_test.go` - Memory usage tests
- `testdata/benchmarks/baseline.txt` - Baseline benchmark results
- `Makefile` - Add `bench`, `bench-compare`, `bench-update-baseline` targets (modify)

## Testing Requirements

- All benchmarks compile and run without panics
- SLO test for 1K files passes (< 1 second)
- SLO test for 10K files passes (< 3 seconds)
- Memory usage for 10K files stays under 500MB
- TUI token recalculation for 1K files under 300ms
- Fixture generator creates correct number of files with expected properties
- `benchstat` comparison produces meaningful output between two runs
- Benchmarks are excluded from regular `go test ./...` (build tag)