package pipeline

import (
	"context"
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

// Run executes the harvx context generation pipeline. It is the central
// orchestrator that coordinates discovery, filtering, relevance sorting,
// content loading, tokenization, redaction, compression, and rendering.
func Run(ctx context.Context, cfg *config.FlagValues) error {
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
	return cc
}
