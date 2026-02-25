// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx preview` subcommand which shows file selection
// and token statistics without generating an output file.
package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// previewHeatmap is a local flag target for --heatmap on the preview command.
// It is a file-level variable (not inside init) to avoid dereferencing the
// flagValues pointer before root.go's init() has populated it.
var previewHeatmap bool

// previewJSON is a local flag target for --json on the preview command.
// When true, preview outputs machine-readable JSON to stdout instead of
// human-readable text to stderr.
var previewJSON bool

// previewCmd implements `harvx preview` which shows file selection and token
// distribution without generating an output file.
var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview file selection and token statistics without generating output",
	Long: `Preview runs the file discovery and token counting stages without writing
an output context file. Use this to inspect which files would be included,
their token counts, and how they relate to your token budget.

Examples:
  # Preview the current directory
  harvx preview

  # Show token density heatmap to find context-bloat files
  harvx preview --heatmap

  # Machine-readable JSON output for scripts
  harvx preview --json

  # Preview with a specific tokenizer
  harvx preview --tokenizer o200k_base

  # Show the top 20 largest files
  harvx preview --top-files 20`,
	RunE: runPreview,
}

func init() {
	previewCmd.Flags().BoolVar(&previewHeatmap, "heatmap", false, "Show token density heatmap (tokens per line)")
	previewCmd.Flags().BoolVar(&previewJSON, "json", false, "Output machine-readable JSON to stdout")
	rootCmd.AddCommand(previewCmd)
}

// runPreview executes the preview subcommand. When --json is set, it runs
// discovery+relevance+tokenization and outputs a PreviewResult JSON to stdout.
// Without --json, it produces human-readable text to stderr.
func runPreview(cmd *cobra.Command, args []string) error {
	fv := GlobalFlags()

	// Sync the local flags back to the shared FlagValues so that
	// downstream callers (e.g. pipeline) can read them from a single place.
	fv.Heatmap = previewHeatmap
	fv.PreviewJSON = previewJSON

	if fv.PreviewJSON {
		return runPreviewJSON(cmd, fv)
	}

	if fv.Heatmap {
		// Pipeline is a stub: show an empty heatmap.
		report := tokenizer.NewHeatmapReport(nil, nil)
		fmt.Fprint(os.Stderr, report.Format())
		return nil
	}

	// Default preview: show an empty token report with configured settings.
	report := tokenizer.NewTokenReport(nil, fv.Tokenizer, fv.MaxTokens)
	fmt.Fprint(os.Stderr, report.Format())
	return nil
}

// runPreviewJSON executes the preview pipeline in JSON mode. It runs
// discovery, relevance, and tokenization stages (skipping budget, redaction,
// compression, and rendering for speed), then outputs a PreviewResult JSON
// to stdout. Diagnostics and warnings go to stderr via slog.
//
// This function always exits 0 on success, even if the repository has issues.
// An empty or problematic directory produces valid JSON with zero counts.
func runPreviewJSON(cmd *cobra.Command, fv *config.FlagValues) error {
	ctx := cmd.Context()

	slog.Debug("preview --json: building pipeline",
		"dir", fv.Dir,
		"tokenizer", fv.Tokenizer,
		"profile", fv.Profile,
	)

	// Build a pipeline with available services.
	// For preview mode, we only need discovery+relevance+tokenization.
	pipeOpts, err := buildPreviewPipelineOptions(fv)
	if err != nil {
		slog.Warn("preview --json: could not build full pipeline, using empty result",
			"error", err,
		)
		// Produce a valid JSON result with zero counts.
		emptyResult := &pipeline.RunResult{
			Stats: pipeline.RunStats{
				TierBreakdown: make(map[int]int),
				TokenizerName: fv.Tokenizer,
			},
		}
		return writePreviewJSON(cmd, emptyResult, fv.Profile, fv.MaxTokens)
	}

	pipe := pipeline.NewPipeline(pipeOpts...)

	// Run with preview stages only (discovery + relevance + tokenize).
	runOpts := pipeline.RunOptions{
		Dir:       fv.Dir,
		MaxTokens: fv.MaxTokens,
		Stages:    pipeline.PreviewStages(),
	}

	result, err := pipe.Run(ctx, runOpts)
	if err != nil {
		slog.Warn("preview --json: pipeline error, producing partial result",
			"error", err,
		)
		// On pipeline error, produce a valid JSON result with zero counts.
		emptyResult := &pipeline.RunResult{
			Stats: pipeline.RunStats{
				TierBreakdown: make(map[int]int),
				TokenizerName: fv.Tokenizer,
			},
		}
		return writePreviewJSON(cmd, emptyResult, fv.Profile, fv.MaxTokens)
	}

	return writePreviewJSON(cmd, result, fv.Profile, fv.MaxTokens)
}

// writePreviewJSON converts a RunResult to a PreviewResult and writes it as
// indented JSON to stdout. Returns an error only if JSON marshaling fails.
func writePreviewJSON(cmd *cobra.Command, result *pipeline.RunResult, profile string, maxTokens int) error {
	preview := pipeline.BuildPreviewResult(result, profile, maxTokens)

	data, err := json.MarshalIndent(preview, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling preview JSON: %w", err)
	}

	// Write JSON to stdout (not stderr). Use cmd.OutOrStdout() for testability.
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// buildPreviewPipelineOptions constructs pipeline options for preview mode.
// It creates a discovery service from the current flag values. Relevance and
// tokenizer services are created when available. Returns an error if the
// discovery service cannot be created.
func buildPreviewPipelineOptions(fv *config.FlagValues) ([]pipeline.PipelineOption, error) {
	// For now, return nil options since the full service wiring (discovery
	// adapter, relevance classifier, tokenizer adapter) happens in later
	// workflow tasks. The pipeline handles nil services gracefully by
	// skipping the corresponding stage.
	//
	// This produces a valid but empty result -- exactly what we want for
	// the initial --json implementation. When services are wired in later
	// tasks (e.g., T-069+), this function will be extended.
	_ = fv
	return nil, nil
}
