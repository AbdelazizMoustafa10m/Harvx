# T-066: Core Pipeline as Go Library API

**Priority:** Must Have
**Effort:** Large (14-20hrs)
**Dependencies:** T-003 (Central Data Types), T-016 (Config Types), T-017 (Multi-Source Config Merging)
**Phase:** 5 - Workflows

---

## Description

Refactor the core processing pipeline into a clean, composable Go library API that can be invoked programmatically -- not just through CLI glue. This is the foundation that enables all workflow commands (`brief`, `review-slice`, `slice`, `workspace`) and the future MCP server to share the same engine. The library exposes a `Pipeline` struct with functional options, accepting a resolved config and returning structured results.

## User Story

As a developer building workflow commands and integrations, I want the core pipeline to be a callable Go library so that `brief`, `review-slice`, `slice`, and future MCP tools all use the same processing engine without duplicating logic.

## Acceptance Criteria

- [ ] `internal/pipeline/pipeline.go` exposes a `Pipeline` struct with a `Run(ctx context.Context, opts RunOptions) (*RunResult, error)` method
- [ ] `RunOptions` encapsulates: target directory, resolved config/profile, subcommand-specific parameters (e.g., paths filter, git refs, max tokens override)
- [ ] `RunResult` contains: slice of `FileDescriptor`, aggregate stats (total files, total tokens, tokenizer used, tier breakdown, redaction count), content hash (XXH3), timing info per stage
- [ ] Pipeline stages are composable: callers can select which stages to run (discovery only, discovery+relevance, full pipeline)
- [ ] `Pipeline` constructor accepts functional options: `WithDiscovery(...)`, `WithRelevance(...)`, `WithTokenizer(...)`, `WithRedactor(...)`, `WithCompressor(...)`, `WithRenderer(...)`
- [ ] Each stage interface is defined: `DiscoveryService`, `RelevanceService`, `TokenBudgeter`, `Redactor`, `Compressor`, `Renderer`
- [ ] The pipeline threads `context.Context` through all stages for cancellation support
- [ ] All stderr/logging output goes through `log/slog` -- the pipeline itself never writes to stdout
- [ ] Exit codes are returned as part of `RunResult` (not via `os.Exit`)
- [ ] Unit tests verify pipeline composition with mock stage implementations
- [ ] `go vet ./internal/pipeline/...` passes

## Technical Notes

- This is the architectural keystone referenced in PRD Sections 5.10 and 6.7: "Design the core pipeline as a Go library (not just CLI glue)"
- The existing CLI commands become thin wrappers that construct a `Pipeline`, call `Run()`, and handle output/exit codes
- Each stage interface should be small and composable per PRD Section 6.7: `DiscoveryService`, `TokenBudgeter`, `Redactor`, `Compressor`, `Renderer`
- Use functional options pattern (e.g., `func WithDiscovery(d DiscoveryService) PipelineOption`) for clean construction
- `RunResult` must include enough metadata for JSON serialization (used by `--output-metadata` and `preview --json`)
- The pipeline must be reentrant and thread-safe: no shared mutable state between `Run()` calls
- Timing data per stage enables `--verbose` diagnostics without coupling to logging
- Reference: PRD Sections 5.10, 6.3 (Processing Pipeline), 6.5 (Central Data Types), 6.7 (Internal API Boundaries)

## Files to Create/Modify

- `internal/pipeline/pipeline.go` - Pipeline struct, Run method, functional options
- `internal/pipeline/options.go` - PipelineOption type and With* constructors
- `internal/pipeline/result.go` - RunResult, RunOptions, StageTimings structs
- `internal/pipeline/interfaces.go` - Stage interfaces (DiscoveryService, RelevanceService, etc.)
- `internal/pipeline/pipeline_test.go` - Unit tests with mock stages
- `internal/pipeline/result_test.go` - JSON serialization roundtrip tests

## Testing Requirements

- Unit test: Pipeline with all mock stages runs successfully and returns correct RunResult
- Unit test: Pipeline with only discovery stage returns partial result
- Unit test: Context cancellation propagates and aborts pipeline mid-stage
- Unit test: RunResult JSON marshaling/unmarshaling roundtrip is correct
- Unit test: Missing required stages return clear error
- Unit test: Stage timing data is populated for each executed stage
- Unit test: Multiple sequential Run() calls on the same Pipeline produce independent results