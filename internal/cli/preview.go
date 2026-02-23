// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx preview` subcommand which shows file selection
// and token statistics without generating an output file.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/tokenizer"
)

// previewHeatmap is a local flag target for --heatmap on the preview command.
// It is a file-level variable (not inside init) to avoid dereferencing the
// flagValues pointer before root.go's init() has populated it.
var previewHeatmap bool

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

  # Preview with a specific tokenizer
  harvx preview --tokenizer o200k_base

  # Show the top 20 largest files
  harvx preview --top-files 20`,
	RunE: runPreview,
}

func init() {
	previewCmd.Flags().BoolVar(&previewHeatmap, "heatmap", false, "Show token density heatmap (tokens per line)")
	rootCmd.AddCommand(previewCmd)
}

// runPreview executes the preview subcommand. The pipeline is a stub; this
// will be wired to full discovery and token counting in a later task. For now
// it shows an empty report or heatmap with the configured flags, exercising
// the CLI flag wiring end-to-end without producing output files.
func runPreview(cmd *cobra.Command, args []string) error {
	fv := GlobalFlags()

	// Sync the local heatmap flag back to the shared FlagValues so that
	// downstream callers (e.g. pipeline) can read it from a single place.
	fv.Heatmap = previewHeatmap

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
