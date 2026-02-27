package compression

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// CompressedMarker is prepended to compressed file content to signal to the LLM
// that function bodies are not included.
const CompressedMarker = "<!-- Compressed: signatures only -->"

// ProgressFunc is called after each file is processed (compressed, skipped, or
// failed). The arguments are the current file index (1-based) and total count.
type ProgressFunc func(current, total int)

// CompressibleFile represents a file that can be compressed. It is a
// package-local type to avoid import cycles with the pipeline package.
// The pipeline package is responsible for adapting FileDescriptor to/from
// this type via ToCompressibleFiles / ApplyCompressionResults.
type CompressibleFile struct {
	Path         string // Relative file path (used for language detection).
	Content      string // File content to compress.
	IsCompressed bool   // Set to true after successful compression.
	Language     string // Detected language identifier.
}

// CompressionConfig holds configuration for the orchestrator.
type CompressionConfig struct {
	Enabled        bool           // Whether compression is active.
	TimeoutPerFile time.Duration  // Max time per file (default 5s).
	Concurrency    int            // Number of parallel compression workers.
	Engine         CompressEngine // Which compression engine to use (default: auto).
}

// DefaultCompressionConfig returns a CompressionConfig with sensible defaults.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Enabled:        false,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    runtime.NumCPU(),
		Engine:         EngineAuto,
	}
}

// Orchestrator manages the compression pipeline. It coordinates language
// detection, compressor dispatch, timeout enforcement, and parallel execution
// via errgroup. The orchestrator supports multiple compression engines (AST,
// regex, auto) via the Engine field in CompressionConfig.
type Orchestrator struct {
	registry      *CompressorRegistry // Primary registry (AST state-machine compressors).
	regexRegistry *CompressorRegistry // Regex-based fallback compressors.
	config        CompressionConfig
	logger        *slog.Logger
	progressFn    ProgressFunc
	processed     int64 // atomic counter for progress reporting
}

// NewOrchestrator creates a compression orchestrator with all built-in
// compressors registered. Both the primary AST registry and the regex
// fallback registry are populated. The orchestrator is ready to use
// immediately.
func NewOrchestrator(config CompressionConfig) *Orchestrator {
	detector := NewLanguageDetector()

	// Primary registry: AST state-machine compressors.
	registry := NewCompressorRegistry(detector)
	registry.Register(NewTypeScriptCompressor())
	registry.Register(NewJavaScriptCompressor())
	registry.Register(NewGoCompressor())
	registry.Register(NewPythonCompressor())
	registry.Register(NewRustCompressor())
	registry.Register(NewJavaCompressor())
	registry.Register(NewCCompressor())
	registry.Register(NewCppCompressor())
	registry.Register(NewJSONCompressor())
	registry.Register(NewYAMLCompressor())
	registry.Register(NewTOMLCompressor())

	// Regex registry: heuristic regex-based compressors.
	regexReg := NewCompressorRegistry(detector)
	for lang := range regexPatternRegistry {
		regexReg.Register(NewRegexCompressor(lang))
	}

	return &Orchestrator{
		registry:      registry,
		regexRegistry: regexReg,
		config:        config,
		logger:        slog.Default(),
	}
}

// SetProgressFunc sets a callback for progress reporting. The callback is
// invoked after each file is processed (compressed, skipped, or failed).
func (o *Orchestrator) SetProgressFunc(fn ProgressFunc) {
	o.progressFn = fn
}

// Compress processes a slice of CompressibleFiles, replacing content with
// compressed output where possible. Files that cannot be compressed retain
// their original content. Context cancellation stops all in-flight work.
func (o *Orchestrator) Compress(ctx context.Context, files []*CompressibleFile) (*CompressionStats, error) {
	stats := &CompressionStats{}
	start := time.Now()

	if len(files) == 0 {
		return stats, nil
	}

	atomic.StoreInt64(&o.processed, 0)
	total := len(files)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(o.config.Concurrency)

	for _, f := range files {
		f := f
		g.Go(func() error {
			err := o.compressFile(gctx, f, stats)

			// Report progress regardless of outcome.
			current := int(atomic.AddInt64(&o.processed, 1))
			if o.progressFn != nil {
				o.progressFn(current, total)
			}

			return err
		})
	}

	if err := g.Wait(); err != nil {
		stats.TotalDuration = time.Since(start)
		return stats, err
	}

	stats.TotalDuration = time.Since(start)
	return stats, nil
}

// compressFile handles compression of a single file with timeout enforcement.
// On any failure (unsupported language, parse error, timeout), the file retains
// its original content. Only context cancellation (Ctrl+C) is propagated as a
// fatal error.
//
// Engine selection logic:
//   - EngineAST: use only the primary registry (state-machine parsers).
//   - EngineRegex: use only the regex registry.
//   - EngineAuto: try primary registry first; on error, retry with regex registry.
func (o *Orchestrator) compressFile(ctx context.Context, f *CompressibleFile, stats *CompressionStats) error {
	// Check parent context first.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("compression cancelled: %w", err)
	}

	switch o.config.Engine {
	case EngineAST:
		return o.compressFileWithRegistry(ctx, f, stats, o.registry, false)
	case EngineRegex:
		return o.compressFileWithRegistry(ctx, f, stats, o.regexRegistry, false)
	case EngineAuto:
		return o.compressFileAuto(ctx, f, stats)
	default:
		// Unknown engine -- treat as auto.
		return o.compressFileAuto(ctx, f, stats)
	}
}

// compressFileAuto implements the EngineAuto strategy: try AST first, then
// fall back to regex on failure.
func (o *Orchestrator) compressFileAuto(ctx context.Context, f *CompressibleFile, stats *CompressionStats) error {
	// Try the primary AST registry first.
	compressor := o.registry.Get(f.Path)
	if compressor == nil {
		// No AST compressor -- try regex registry directly.
		return o.compressFileWithRegistry(ctx, f, stats, o.regexRegistry, false)
	}

	// Enforce per-file timeout.
	fileCtx, cancel := context.WithTimeout(ctx, o.config.TimeoutPerFile)
	defer cancel()

	start := time.Now()
	output, err := compressor.Compress(fileCtx, []byte(f.Content))
	elapsed := time.Since(start)

	if err != nil {
		// Check if parent context was cancelled (Ctrl+C) -- propagate.
		if ctx.Err() != nil {
			return fmt.Errorf("compression cancelled: %w", ctx.Err())
		}

		// AST failed -- fall back to regex.
		o.logger.Debug("AST compression failed, falling back to regex",
			"file", f.Path,
			"language", compressor.Language(),
			"error", err,
			"elapsed", elapsed,
		)
		return o.compressFileWithRegistry(ctx, f, stats, o.regexRegistry, true)
	}

	// Check for fallback output.
	if IsFallback(output) {
		// Try regex instead.
		return o.compressFileWithRegistry(ctx, f, stats, o.regexRegistry, false)
	}

	return o.applyCompression(f, stats, output, compressor.Language(), elapsed)
}

// compressFileWithRegistry compresses a file using the specified registry.
// If isFallbackAttempt is true, failures are logged differently.
func (o *Orchestrator) compressFileWithRegistry(
	ctx context.Context,
	f *CompressibleFile,
	stats *CompressionStats,
	reg *CompressorRegistry,
	isFallbackAttempt bool,
) error {
	// Check parent context first.
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("compression cancelled: %w", err)
	}

	compressor := reg.Get(f.Path)
	if compressor == nil {
		// Unsupported language -- keep original content.
		stats.addSkipped()
		o.logger.Debug("compression skipped: unsupported language",
			"file", f.Path,
		)
		return nil
	}

	// Enforce per-file timeout.
	fileCtx, cancel := context.WithTimeout(ctx, o.config.TimeoutPerFile)
	defer cancel()

	start := time.Now()
	output, err := compressor.Compress(fileCtx, []byte(f.Content))
	elapsed := time.Since(start)

	if err != nil {
		// Check if parent context was cancelled (Ctrl+C) -- propagate.
		if ctx.Err() != nil {
			return fmt.Errorf("compression cancelled: %w", ctx.Err())
		}

		// Per-file timeout or parse error -- keep original content.
		if errors.Is(err, context.DeadlineExceeded) {
			stats.addTimedOut()
			o.logger.Warn("compression timed out, using original",
				"file", f.Path,
				"language", compressor.Language(),
				"timeout", o.config.TimeoutPerFile,
				"elapsed", elapsed,
			)
		} else {
			stats.addFailed()
			o.logger.Debug("compression failed, using original",
				"file", f.Path,
				"language", compressor.Language(),
				"error", err,
				"elapsed", elapsed,
				"fallback_attempt", isFallbackAttempt,
			)
		}
		return nil
	}

	// Check for fallback output (unsupported language that has a registered
	// compressor but returned passthrough content).
	if IsFallback(output) {
		stats.addSkipped()
		o.logger.Debug("compression skipped: fallback compressor",
			"file", f.Path,
		)
		return nil
	}

	return o.applyCompression(f, stats, output, compressor.Language(), elapsed)
}

// applyCompression applies a successful compression result to the file and
// updates statistics.
func (o *Orchestrator) applyCompression(
	f *CompressibleFile,
	stats *CompressionStats,
	output *CompressedOutput,
	language string,
	elapsed time.Duration,
) error {
	rendered := output.Render()
	compressed := CompressedMarker + "\n" + rendered
	originalLen := len(f.Content)

	f.Content = compressed
	f.IsCompressed = true
	f.Language = language

	stats.addCompressed()
	// Track byte counts (using content length as proxy; actual token
	// recounting happens in the tokenizer stage).
	stats.addTokens(originalLen, len(compressed))

	o.logger.Debug("file compressed",
		"file", f.Path,
		"language", language,
		"ratio", output.CompressionRatio(),
		"elapsed", elapsed,
		"original_bytes", originalLen,
		"compressed_bytes", len(compressed),
	)

	return nil
}
