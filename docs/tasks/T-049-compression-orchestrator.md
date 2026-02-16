# T-049: Compression Orchestrator and Pipeline Integration

**Priority:** Must Have
**Effort:** Medium (10-14hrs)
**Dependencies:** T-042, T-043, T-044, T-045, T-046, T-048
**Phase:** 3 - Security & Compression

---

## Description

Build the compression orchestrator -- the top-level component that ties together the grammar registry, language detection, compressor registry, and the main Harvx processing pipeline. The orchestrator is responsible for: (1) receiving a list of `FileDescriptor` objects that survived token budget filtering, (2) detecting each file's language, (3) dispatching to the appropriate compressor, (4) enforcing per-file compression timeouts, (5) adding the `<!-- Compressed: signatures only -->` marker, (6) handling errors gracefully (fall back to full content on any failure), and (7) running compression in parallel using `errgroup`. This is the glue layer that makes compression a seamless step in the pipeline between content loading/redaction and token counting.

## User Story

As a developer running `harvx --compress`, I want compression to happen automatically on all supported files that survive budget filtering so that I get maximum token reduction without manual intervention, with graceful handling of parse failures and timeouts.

## Acceptance Criteria

- [ ] Orchestrator accepts `[]FileDescriptor` and returns `[]FileDescriptor` with compressed content
- [ ] Only compresses files that survived token budget filtering (lazy loading -- never compress files that will be omitted)
- [ ] Language detection runs on each file to determine compressor
- [ ] Dispatches to correct `LanguageCompressor` per file
- [ ] Falls back to full content if: language unsupported, parse error, timeout exceeded
- [ ] Per-file timeout enforcement via `context.WithTimeout` (default 5000ms, configurable via `--compress-timeout`)
- [ ] Compressed files have `<!-- Compressed: signatures only -->` header prepended to content
- [ ] `FileDescriptor.IsCompressed` is set to `true` only for successfully compressed files
- [ ] `FileDescriptor.Content` is replaced with compressed content
- [ ] `FileDescriptor.TokenCount` is recalculated on compressed content (not original)
- [ ] Compression runs in parallel using `errgroup.SetLimit(runtime.NumCPU())`
- [ ] `context.Context` cancellation is respected (Ctrl+C stops compression)
- [ ] Progress reporting via callback or channel (for TUI and CLI progress bars)
- [ ] Compression summary statistics are collected (files compressed, failed, skipped, total savings)
- [ ] Integration with `--compress` flag and `compression = true` profile setting
- [ ] Logging: per-file compression result (language, ratio, time) at `debug` level

## Technical Notes

### Orchestrator Interface

```go
package compression

import (
    "context"
    "time"

    "github.com/harvx/harvx/internal/pipeline"
)

// CompressionStats holds aggregate statistics for a compression run.
type CompressionStats struct {
    FilesCompressed  int
    FilesFailed      int
    FilesSkipped     int     // Unsupported language
    FilesTimedOut    int
    OriginalTokens   int
    CompressedTokens int
    AverageRatio     float64 // Average compression ratio across compressed files
    TotalDuration    time.Duration
}

// CompressionConfig holds configuration for the orchestrator.
type CompressionConfig struct {
    Enabled        bool          // Whether compression is active
    TimeoutPerFile time.Duration // Max time per file (default 5s)
    Concurrency    int           // Number of parallel compression workers
}

// Orchestrator manages the compression pipeline.
type Orchestrator struct {
    registry *CompressorRegistry
    grammar  *GrammarRegistry
    config   CompressionConfig
}

// NewOrchestrator creates a new compression orchestrator.
func NewOrchestrator(ctx context.Context, config CompressionConfig) (*Orchestrator, error) {
    grammar, err := NewGrammarRegistry(ctx)
    if err != nil {
        return nil, fmt.Errorf("initializing grammar registry: %w", err)
    }

    detector := NewLanguageDetector()
    registry := NewCompressorRegistry(detector)

    // Register all built-in compressors
    registry.Register(NewTypeScriptCompressor(grammar))
    registry.Register(NewJavaScriptCompressor(grammar))
    registry.Register(NewGoCompressor(grammar))
    registry.Register(NewPythonCompressor(grammar))
    registry.Register(NewRustCompressor(grammar))
    registry.Register(NewJavaCompressor(grammar))
    registry.Register(NewCCompressor(grammar))
    registry.Register(NewCppCompressor(grammar))
    registry.Register(NewJSONCompressor())
    registry.Register(NewYAMLCompressor())
    registry.Register(NewTOMLCompressor())

    return &Orchestrator{
        registry: registry,
        grammar:  grammar,
        config:   config,
    }, nil
}

// Compress processes a slice of FileDescriptors, replacing content with
// compressed output where possible. Files that cannot be compressed retain
// their original content.
func (o *Orchestrator) Compress(ctx context.Context, files []*pipeline.FileDescriptor) (*CompressionStats, error) {
    stats := &CompressionStats{}

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(o.config.Concurrency)

    for _, f := range files {
        f := f
        g.Go(func() error {
            return o.compressFile(ctx, f, stats)
        })
    }

    if err := g.Wait(); err != nil {
        return stats, err
    }

    return stats, nil
}

// compressFile handles compression of a single file with timeout.
func (o *Orchestrator) compressFile(ctx context.Context, f *pipeline.FileDescriptor, stats *CompressionStats) error {
    compressor := o.registry.Get(f.Path)
    if compressor == nil {
        // Unsupported language -- keep original content
        atomic.AddInt32(&stats.FilesSkipped, 1)
        return nil
    }

    fileCtx, cancel := context.WithTimeout(ctx, o.config.TimeoutPerFile)
    defer cancel()

    output, err := compressor.Compress(fileCtx, []byte(f.Content))
    if err != nil {
        // Compression failed -- keep original content
        slog.Debug("compression failed, using original",
            "file", f.Path,
            "language", compressor.Language(),
            "error", err,
        )
        if errors.Is(err, context.DeadlineExceeded) {
            atomic.AddInt32(&stats.FilesTimedOut, 1)
        } else {
            atomic.AddInt32(&stats.FilesFailed, 1)
        }
        return nil // Not a fatal error
    }

    // Apply compressed content
    compressed := "<!-- Compressed: signatures only -->\n" + output.Render()
    f.Content = compressed
    f.IsCompressed = true
    // TokenCount will be recalculated by the token counting stage
    
    atomic.AddInt32(&stats.FilesCompressed, 1)
    return nil
}

// Close releases all WASM resources.
func (o *Orchestrator) Close(ctx context.Context) error {
    return o.grammar.Close(ctx)
}
```

### Pipeline Integration Points

The compression orchestrator plugs into the Harvx pipeline between content loading/redaction and token counting:

```
Discovery -> Relevance Sorting -> Content Loading -> Redaction
                                                        |
                                                        v
                                               [Token Budget Pre-estimate]
                                                        |
                                                        v
                                               [Filter: budget survivors only]
                                                        |
                                                        v
                                               *** COMPRESSION (this task) ***
                                                        |
                                                        v
                                               [Token Counting (on compressed content)]
                                                        |
                                                        v
                                               [Final Budget Enforcement]
                                                        |
                                                        v
                                               Output Rendering
```

Key integration requirement: compression is applied ONLY to files that survive initial budget filtering. This means a rough token estimate (byte-based or character-based) is done first, budget survivors are identified, and only those files are sent to the compressor. After compression, exact token counting runs on the compressed content.

### Compressed Output Markers

Each compressed file's content starts with:
```
<!-- Compressed: signatures only -->
```

This marker:
- Tells the LLM that the file has been compressed
- Signals that function bodies are not included
- Is consistent across all languages

### Configuration Integration

The orchestrator is activated by:
1. `--compress` CLI flag
2. `compression = true` in profile configuration
3. `--compress-timeout <ms>` sets `CompressionConfig.TimeoutPerFile` (default: 5000ms)

When compression is disabled, the orchestrator is never instantiated (no WASM overhead).

### Error Handling Philosophy

Compression failures are NEVER fatal. The pipeline continues with original content:
- Parse error -> use original content, log at debug level
- Timeout -> use original content, log at warn level
- WASM runtime error -> use original content, log at error level
- Context cancellation (Ctrl+C) -> propagate cancellation (this IS fatal)

### Concurrency Safety

- `GrammarRegistry` modules are shared read-only after compilation
- Each `Compress()` call creates its own parser instance (no shared mutable state)
- `CompressionStats` uses atomic operations for concurrent counter updates
- File content replacement is safe because each goroutine operates on a unique `FileDescriptor`

## Files to Create/Modify

- `internal/compression/orchestrator.go` -- Orchestrator implementation
- `internal/compression/orchestrator_test.go` -- Unit tests
- `internal/compression/stats.go` -- CompressionStats type and helpers
- `internal/pipeline/pipeline.go` -- Add compression step to pipeline (modify existing)
- `internal/cli/root.go` -- Add `--compress` and `--compress-timeout` flags (modify existing)
- `internal/config/config.go` -- Add compression config fields (modify existing)

## Testing Requirements

- Unit test: Orchestrator compresses TypeScript files correctly
- Unit test: Orchestrator falls back on unsupported language (.md file)
- Unit test: Per-file timeout triggers fallback after deadline
- Unit test: Context cancellation stops all in-flight compression
- Unit test: Compression stats are accumulated correctly
- Unit test: `<!-- Compressed: signatures only -->` marker is prepended
- Unit test: Concurrent compression of 20+ files completes without race conditions
- Integration test: Full pipeline with `--compress` produces output with compressed files
- Integration test: Mixed-language project (TS + Go + JSON + .md) compresses correctly
- Benchmark: Compression throughput on 100 files (target: < 2s total)
- Benchmark: Memory usage during parallel compression

## References

- PRD Section 5.6: "Lazy loading: only parse files that survive token budget filtering"
- PRD Section 5.6: "Supports `--compress-timeout <ms>` flag to abandon slow parsing operations (default: 5000ms)"
- PRD Section 5.6: "Compressed output is clearly marked: `<!-- Compressed: signatures only -->` header per file"
- PRD Section 6.3: Processing Pipeline (Compression step in Content Loading stage)
- PRD Section 6.4: Concurrency Model (errgroup with SetLimit)
- PRD Section 6.7: Internal API Boundaries (Compressor interface)