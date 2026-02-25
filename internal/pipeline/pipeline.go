package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/harvx/harvx/internal/compression"
	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/security"
)

// DefaultRedactionReportPath is the default file path for --redaction-report
// when the flag is set without an explicit path.
const DefaultRedactionReportPath = "harvx-redaction-report.json"

// Pipeline is the core processing engine that coordinates all pipeline stages.
// It is constructed via NewPipeline with functional options and executed via Run.
// Pipeline is reentrant: multiple sequential Run calls produce independent results.
//
// Callers configure which stages are active by providing service implementations
// through WithDiscovery, WithRelevance, WithTokenizer, WithBudget, WithRedactor,
// WithCompressor, and WithRenderer options. Stages without a configured service
// are skipped during Run.
type Pipeline struct {
	discovery   DiscoveryService
	relevance   RelevanceService
	tokenizer   TokenizerService
	budget      BudgetService
	redactor    RedactionService
	compressor  CompressionService
	renderer    RenderService
}

// NewPipeline constructs a Pipeline with the provided functional options.
// At minimum, a DiscoveryService should be provided for meaningful results.
func NewPipeline(opts ...PipelineOption) *Pipeline {
	p := &Pipeline{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Run executes the pipeline stages in order, respecting context cancellation
// and the stage selection in opts. Each stage enriches the FileDescriptor
// slice produced by discovery. Stages without a configured service are
// automatically skipped.
//
// The pipeline never writes to stdout. All diagnostic output goes through slog.
// Exit codes are returned as part of RunResult, never via os.Exit.
func (p *Pipeline) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	totalStart := time.Now()

	result := &RunResult{
		ExitCode: ExitSuccess,
		Stats: RunStats{
			TierBreakdown: make(map[int]int),
		},
	}

	stages := opts.Stages
	if stages == nil {
		stages = NewStageSelection()
	}

	slog.Info("pipeline starting",
		"dir", opts.Dir,
		"stages", fmt.Sprintf("%+v", *stages),
	)

	var files []FileDescriptor
	var filePtrs []*FileDescriptor

	// Stage 1: Discovery
	if stages.Discovery && p.discovery != nil {
		start := time.Now()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		discoveryOpts := DiscoveryOptions{
			RootDir: opts.Dir,
		}
		discoveryResult, err := p.discovery.Discover(ctx, discoveryOpts)
		if err != nil {
			return nil, fmt.Errorf("discovery: %w", err)
		}

		result.Timings.Discovery = time.Since(start)
		files = discoveryResult.Files
		result.Stats.DiscoveryTotal = discoveryResult.TotalFound
		result.Stats.DiscoverySkipped = discoveryResult.TotalSkipped

		slog.Debug("discovery complete",
			"files", len(files),
			"total_found", discoveryResult.TotalFound,
			"skipped", discoveryResult.TotalSkipped,
			"duration", result.Timings.Discovery,
		)
	}

	// Convert to pointer slice for stages that mutate in place.
	filePtrs = toPointerSlice(files)

	// Stage 2: Relevance
	if stages.Relevance && p.relevance != nil && len(filePtrs) > 0 {
		start := time.Now()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		filePtrs = p.relevance.Classify(filePtrs)
		result.Timings.Relevance = time.Since(start)

		slog.Debug("relevance complete",
			"files", len(filePtrs),
			"duration", result.Timings.Relevance,
		)
	}

	// Stage 3: Redaction
	if stages.Redaction && p.redactor != nil && len(filePtrs) > 0 {
		start := time.Now()

		for _, fd := range filePtrs {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			if fd.Content == "" {
				continue
			}

			redacted, count, err := p.redactor.Redact(ctx, fd.Content, fd.Path)
			if err != nil {
				fd.Error = fmt.Errorf("redacting %s: %w", fd.Path, err)
				result.ExitCode = ExitPartial
				continue
			}
			fd.Content = redacted
			fd.Redactions = count
		}

		result.Timings.Redaction = time.Since(start)

		slog.Debug("redaction complete",
			"duration", result.Timings.Redaction,
		)
	}

	// Stage 4: Compression
	if stages.Compression && p.compressor != nil && len(filePtrs) > 0 {
		start := time.Now()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if err := p.compressor.Compress(ctx, filePtrs); err != nil {
			slog.Warn("compression stage error", "error", err)
			// Non-fatal: compression errors degrade gracefully.
		}

		result.Timings.Compression = time.Since(start)

		slog.Debug("compression complete",
			"duration", result.Timings.Compression,
		)
	}

	// Stage 5: Tokenization
	if stages.Tokenize && p.tokenizer != nil && len(filePtrs) > 0 {
		start := time.Now()

		for _, fd := range filePtrs {
			if fd.Content != "" {
				fd.TokenCount = p.tokenizer.Count(fd.Content)
			}
		}

		result.Stats.TokenizerName = p.tokenizer.Name()
		result.Timings.Tokenize = time.Since(start)

		slog.Debug("tokenization complete",
			"tokenizer", p.tokenizer.Name(),
			"duration", result.Timings.Tokenize,
		)
	}

	// Stage 6: Budget enforcement
	if stages.Budget && p.budget != nil && len(filePtrs) > 0 {
		start := time.Now()

		budgetResult, err := p.budget.Enforce(filePtrs, opts.MaxTokens)
		if err != nil {
			return nil, fmt.Errorf("budget enforcement: %w", err)
		}

		filePtrs = budgetResult.Included
		result.Timings.Budget = time.Since(start)

		slog.Debug("budget enforcement complete",
			"included", len(budgetResult.Included),
			"skipped", len(budgetResult.Skipped),
			"total_tokens", budgetResult.TotalTokens,
			"budget_used", budgetResult.BudgetUsed,
			"duration", result.Timings.Budget,
		)
	}

	// Convert back to value slice for the result.
	files = toValueSlice(filePtrs)

	// Compute aggregate stats.
	result.Files = files
	result.Stats.TotalFiles = len(files)
	for _, fd := range files {
		result.Stats.TotalTokens += fd.TokenCount
		result.Stats.TierBreakdown[fd.Tier]++
		result.Stats.RedactionCount += fd.Redactions
		if fd.IsCompressed {
			result.Stats.CompressedFiles++
		}
		if fd.Error != nil {
			result.ExitCode = ExitPartial
		}
	}

	result.Timings.Total = time.Since(totalStart)

	slog.Info("pipeline complete",
		"files", result.Stats.TotalFiles,
		"tokens", result.Stats.TotalTokens,
		"redactions", result.Stats.RedactionCount,
		"compressed", result.Stats.CompressedFiles,
		"exit_code", result.ExitCode,
		"duration", result.Timings.Total,
	)

	return result, nil
}

// HasDiscovery reports whether a discovery service is configured.
func (p *Pipeline) HasDiscovery() bool {
	return p.discovery != nil
}

// HasRedactor reports whether a redaction service is configured.
func (p *Pipeline) HasRedactor() bool {
	return p.redactor != nil
}

// toPointerSlice converts a value slice to a pointer slice.
func toPointerSlice(files []FileDescriptor) []*FileDescriptor {
	ptrs := make([]*FileDescriptor, len(files))
	for i := range files {
		ptrs[i] = &files[i]
	}
	return ptrs
}

// toValueSlice converts a pointer slice to a value slice.
func toValueSlice(ptrs []*FileDescriptor) []FileDescriptor {
	vals := make([]FileDescriptor, len(ptrs))
	for i, p := range ptrs {
		vals[i] = *p
	}
	return vals
}

// --- Legacy CLI glue functions ---
// These functions support the existing CLI integration and will be refactored
// as workflow commands adopt the Pipeline API.

// RunLegacy executes the harvx context generation pipeline using the legacy
// CLI flag-based interface. It is the central orchestrator that coordinates
// discovery, filtering, relevance sorting, content loading, tokenization,
// redaction, compression, and rendering.
//
// Deprecated: Use Pipeline.Run for new code. This function is retained for
// backward compatibility with the existing CLI commands.
func RunLegacy(ctx context.Context, cfg *config.FlagValues) error {
	slog.Info("Starting Harvx context generation",
		"dir", cfg.Dir,
		"output", cfg.Output,
		"format", cfg.Format,
	)

	slog.Debug("resolved configuration",
		"dir", cfg.Dir,
		"output", cfg.Output,
		"format", cfg.Format,
		"target", cfg.Target,
		"filters", cfg.Filters,
		"includes", cfg.Includes,
		"excludes", cfg.Excludes,
		"git_tracked_only", cfg.GitTrackedOnly,
		"skip_large_files", cfg.SkipLargeFiles,
		"stdout", cfg.Stdout,
		"line_numbers", cfg.LineNumbers,
		"no_redact", cfg.NoRedact,
		"fail_on_redaction", cfg.FailOnRedaction,
	)

	// Build the redaction configuration.
	redactCfg := buildRedactionConfig(cfg)

	slog.Debug("redaction configuration",
		"enabled", redactCfg.Enabled,
		"confidence_threshold", redactCfg.ConfidenceThreshold,
		"exclude_paths", redactCfg.ExcludePaths,
		"custom_patterns", len(redactCfg.CustomPatterns),
	)

	// Instantiate the redactor for use in the content loading stage.
	redactor := security.NewStreamRedactor(nil, nil, redactCfg)

	// TODO: Implement discovery, filtering, rendering pipeline.
	// Each file worker will call redactor.Redact(ctx, rawContent, file.Path).

	// Retrieve the aggregated redaction summary after all files are processed.
	summary := redactor.Summary()

	// Print the redaction summary line to stderr.
	printRedactionSummary(summary)

	// Write the redaction report if requested.
	if err := maybeWriteReport(cfg, summary); err != nil {
		slog.Warn("failed to write redaction report", "error", err)
	}

	// Check --fail-on-redaction exit condition.
	if cfg.FailOnRedaction && !cfg.NoRedact && summary.TotalCount > 0 {
		return NewRedactionError(fmt.Sprintf("secrets detected: %d redaction(s) found; failing as requested by --fail-on-redaction", summary.TotalCount))
	}

	// Run compression if enabled.
	if cfg.Compress {
		compressionCfg := buildCompressionConfig(cfg)
		slog.Debug("compression configuration",
			"enabled", compressionCfg.Enabled,
			"timeout_per_file", compressionCfg.TimeoutPerFile,
			"concurrency", compressionCfg.Concurrency,
			"engine", compressionCfg.Engine,
		)
		// TODO: Apply compression to budget-surviving files once the full pipeline is wired.
		_ = compressionCfg
	}

	return nil
}

// buildRedactionConfig derives a security.RedactionConfig from the resolved
// CLI flags. Profile-level config will be merged in by a later pipeline task
// (T-041+). For now, flags are the sole source of truth.
//
// Default: redaction enabled with medium confidence threshold.
func buildRedactionConfig(cfg *config.FlagValues) security.RedactionConfig {
	enabled := !cfg.NoRedact

	return security.RedactionConfig{
		Enabled:             enabled,
		ConfidenceThreshold: security.ConfidenceMedium,
	}
}

// printRedactionSummary writes the redaction summary line to stderr.
// Format matches the CLI output spec:
//
//	Redactions:  3 (2 API keys, 1 connection string)
//	Redactions:  0
func printRedactionSummary(summary security.RedactionSummary) {
	gen := security.NewReportGenerator()
	if summary.TotalCount == 0 {
		fmt.Fprintf(os.Stderr, "Redactions:  0\n")
		return
	}
	line := gen.FormatInlineSummary(summary)
	fmt.Fprintf(os.Stderr, "Redactions:  %s\n", line)
}

// maybeWriteReport writes the detailed redaction report to disk when
// --redaction-report is set. Uses DefaultRedactionReportPath when the flag
// was set without a path value.
func maybeWriteReport(cfg *config.FlagValues, summary security.RedactionSummary) error {
	if cfg.RedactionReport == "" {
		return nil
	}

	path := cfg.RedactionReport
	if path == "true" || path == "1" {
		// Flag was set without a value (cobra string flag fallback).
		path = DefaultRedactionReportPath
	}

	gen := security.NewReportGenerator()
	report := gen.BuildReport(summary, nil, "", string(security.ConfidenceMedium))
	if err := gen.WriteReport(report, path); err != nil {
		return fmt.Errorf("writing redaction report: %w", err)
	}

	slog.Info("redaction report written", "path", path)
	return nil
}

// buildCompressionConfig derives a compression.CompressionConfig from the
// resolved CLI flags.
func buildCompressionConfig(cfg *config.FlagValues) compression.CompressionConfig {
	cc := compression.DefaultCompressionConfig()
	cc.Enabled = cfg.Compress
	cc.TimeoutPerFile = time.Duration(cfg.CompressTimeout) * time.Millisecond

	// Parse the engine flag. Validation already happened in ValidateFlags,
	// so errors here are unexpected. Fall back to auto on error.
	engine, err := compression.ParseCompressEngine(cfg.CompressEngine)
	if err != nil {
		slog.Warn("invalid compress engine, defaulting to auto",
			"engine", cfg.CompressEngine,
			"error", err,
		)
		engine = compression.EngineAuto
	}
	cc.Engine = engine

	return cc
}

// ErrNoDiscovery is returned when Pipeline.Run is called without a configured
// discovery service and the stage selection requires discovery.
var ErrNoDiscovery = errors.New("pipeline: no discovery service configured")
