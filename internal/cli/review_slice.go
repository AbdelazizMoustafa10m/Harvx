// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx review-slice` subcommand which generates a
// PR-specific context slice with changed files and their bounded neighborhood.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/harvx/harvx/internal/workflows"
)

// reviewSliceJSON is a local flag target for --json on the review-slice command.
var reviewSliceJSON bool

// reviewSliceBase is the --base flag for the base git ref.
var reviewSliceBase string

// reviewSliceHead is the --head flag for the head git ref.
var reviewSliceHead string

// reviewSliceCmd implements `harvx review-slice` which generates a PR-specific
// context slice containing changed files and their bounded neighborhood.
var reviewSliceCmd = &cobra.Command{
	Use:   "review-slice",
	Short: "Generate a PR-specific context slice for AI code review",
	Long: `Generate a Review Slice artifact containing changed files and their bounded
neighborhood (public interfaces, related tests, dependency neighbors) for
PR-specific AI code review.

The review-slice is the dynamic companion to the stable 'brief' artifact,
providing reviewers with the specific code context needed to understand a
change within the broader project architecture.

The output is deterministic and content-addressed via XXH3 hash.

Examples:
  # Generate review slice for a PR
  harvx review-slice --base origin/main --head HEAD --stdout

  # Custom token budget
  harvx review-slice --base main --head feature-branch --max-tokens 30000 --stdout

  # Machine-readable JSON metadata
  harvx review-slice --base main --head HEAD --json

  # XML output for Claude
  harvx review-slice --base main --head HEAD --target claude --stdout

  # Save to file
  harvx review-slice --base main --head HEAD -o review-context.md

  # No neighbors (changed files only)
  harvx review-slice --base main --head HEAD --stdout --profile minimal`,
	RunE: runReviewSlice,
}

func init() {
	reviewSliceCmd.Flags().BoolVar(&reviewSliceJSON, "json", false, "Output machine-readable JSON metadata to stdout")
	reviewSliceCmd.Flags().StringVar(&reviewSliceBase, "base", "", "Base git ref (required, e.g., origin/main, commit SHA)")
	reviewSliceCmd.Flags().StringVar(&reviewSliceHead, "head", "", "Head git ref (required, e.g., HEAD, branch name)")
	_ = reviewSliceCmd.MarkFlagRequired("base")
	_ = reviewSliceCmd.MarkFlagRequired("head")
	rootCmd.AddCommand(reviewSliceCmd)
}

// runReviewSlice executes the review-slice subcommand. It resolves configuration,
// generates the review slice, and routes output to stdout or a file.
func runReviewSlice(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	// Sync the local flag back to FlagValues.
	fv.PreviewJSON = reviewSliceJSON

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("review-slice: resolving directory: %w", err)
	}

	// Resolve slice configuration from profile.
	maxTokens, depth := resolveSliceConfig(fv)

	// Build the token counter.
	tokenCount, err := buildSliceTokenCounter(fv.Tokenizer)
	if err != nil {
		slog.Warn("review-slice: could not create tokenizer, using estimator",
			"tokenizer", fv.Tokenizer,
			"error", err,
		)
		tokenCount = nil
	}

	sliceOpts := workflows.ReviewSliceOptions{
		RootDir:       rootDir,
		BaseRef:       reviewSliceBase,
		HeadRef:       reviewSliceHead,
		MaxTokens:     maxTokens,
		Depth:         depth,
		Target:        fv.Target,
		AssertInclude: fv.AssertIncludes,
		TokenCounter:  tokenCount,
		Compress:      fv.Compress,
	}

	result, err := workflows.GenerateReviewSlice(context.Background(), sliceOpts)
	if err != nil {
		return fmt.Errorf("review-slice: %w", err)
	}

	// Handle --json output.
	if reviewSliceJSON {
		return writeReviewSliceJSON(cmd, result, maxTokens)
	}

	// Route output.
	return writeReviewSliceOutput(cmd, fv, result)
}

// resolveSliceConfig determines the token budget and depth for the review slice.
// It checks the profile configuration first, then falls back to defaults.
func resolveSliceConfig(fv *config.FlagValues) (int, int) {
	maxTokens := workflows.DefaultSliceMaxTokens
	depth := workflows.DefaultSliceDepth

	rc, err := config.Resolve(config.ResolveOptions{
		ProfileName: fv.Profile,
		TargetDir:   fv.Dir,
	})
	if err == nil && rc.Profile != nil {
		if rc.Profile.SliceMaxTokens > 0 {
			maxTokens = rc.Profile.SliceMaxTokens
		}
		if rc.Profile.SliceDepth > 0 {
			depth = rc.Profile.SliceDepth
		}
	}

	// CLI --max-tokens overrides profile.
	if fv.MaxTokens > 0 {
		maxTokens = fv.MaxTokens
	}

	return maxTokens, depth
}

// buildSliceTokenCounter creates a token counter function from the configured
// tokenizer name. Returns nil if the tokenizer cannot be created.
func buildSliceTokenCounter(name string) (func(string) int, error) {
	if name == "none" || name == "" {
		return nil, nil
	}

	tok, err := tokenizer.NewTokenizer(name)
	if err != nil {
		return nil, fmt.Errorf("creating tokenizer %q: %w", name, err)
	}

	return tok.Count, nil
}

// writeReviewSliceJSON writes the review slice metadata as JSON to stdout.
func writeReviewSliceJSON(cmd *cobra.Command, result *workflows.ReviewSliceResult, maxTokens int) error {
	meta := workflows.ReviewSliceJSON{
		TokenCount:    result.TokenCount,
		ContentHash:   result.FormattedHash,
		ChangedFiles:  result.ChangedFiles,
		NeighborFiles: result.NeighborFiles,
		DeletedFiles:  result.DeletedFiles,
		TotalFiles:    result.TotalFiles,
		MaxTokens:     maxTokens,
		BaseRef:       reviewSliceBase,
		HeadRef:       reviewSliceHead,
	}

	// Ensure nil slices serialize as empty arrays.
	if meta.ChangedFiles == nil {
		meta.ChangedFiles = []string{}
	}
	if meta.NeighborFiles == nil {
		meta.NeighborFiles = []string{}
	}
	if meta.DeletedFiles == nil {
		meta.DeletedFiles = []string{}
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("review-slice: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeReviewSliceOutput routes the review slice content to stdout or a file
// based on the flag configuration.
func writeReviewSliceOutput(cmd *cobra.Command, fv *config.FlagValues, result *workflows.ReviewSliceResult) error {
	content := result.Content

	if fv.Stdout {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Determine output path.
	outputPath := fv.Output
	if outputPath == config.DefaultOutput {
		outputPath = "harvx-review-slice.md"
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("review-slice: writing output to %s: %w", outputPath, err)
	}

	// Report to stderr.
	fmt.Fprintf(cmd.ErrOrStderr(),
		"Review slice written to %s (%d tokens, hash: %s, %d changed, %d neighbors)\n",
		outputPath, result.TokenCount, result.FormattedHash,
		len(result.ChangedFiles), len(result.NeighborFiles))
	return nil
}
