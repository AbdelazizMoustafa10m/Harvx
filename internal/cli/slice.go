// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx slice` subcommand which generates a targeted
// context slice for a specific module or directory and its bounded neighborhood.
package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/workflows"
)

// sliceJSON is a local flag target for --json on the slice command.
var sliceJSON bool

// slicePaths is a local flag target for --path on the slice command (repeatable).
var slicePaths []string

// sliceCmd implements `harvx slice` which generates a targeted context slice
// for a specific module or directory and its bounded neighborhood.
var sliceCmd = &cobra.Command{
	Use:   "slice",
	Short: "Generate a targeted context slice for a specific module or directory",
	Long: `Generate a Module Slice artifact containing all files within the specified
path(s) and their bounded neighborhood (imports, tests, dependency neighbors)
for focused AI context about a specific area of the codebase.

The module slice is ideal for coding agents that need deep context about a
specific module without consuming the entire context window on unrelated code.

The output is deterministic and content-addressed via XXH3 hash.

Examples:
  # Slice a single module
  harvx slice --path internal/auth --stdout

  # Slice multiple modules
  harvx slice --path internal/auth --path internal/middleware --stdout

  # Custom token budget
  harvx slice --path src/components --max-tokens 30000 --stdout

  # Machine-readable JSON metadata
  harvx slice --path internal/auth --json

  # XML output for Claude
  harvx slice --path lib/services --target claude --stdout

  # Save to file
  harvx slice --path internal/auth -o auth-context.md`,
	RunE: runSlice,
}

func init() {
	sliceCmd.Flags().BoolVar(&sliceJSON, "json", false, "Output machine-readable JSON metadata to stdout")
	sliceCmd.Flags().StringArrayVar(&slicePaths, "path", nil, "Relative path to slice (repeatable)")
	_ = sliceCmd.MarkFlagRequired("path")
	rootCmd.AddCommand(sliceCmd)
}

// runSlice executes the slice subcommand. It resolves configuration, generates
// the module slice, and routes output to stdout or a file.
func runSlice(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	// Sync the local flag back to FlagValues.
	fv.PreviewJSON = sliceJSON

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("slice: resolving directory: %w", err)
	}

	// Resolve slice configuration from profile.
	maxTokens, depth := resolveSliceConfig(fv)

	// Build the token counter.
	tokenCount, err := buildSliceTokenCounter(fv.Tokenizer)
	if err != nil {
		slog.Warn("slice: could not create tokenizer, using estimator",
			"tokenizer", fv.Tokenizer,
			"error", err,
		)
		tokenCount = nil
	}

	sliceOpts := workflows.ModuleSliceOptions{
		RootDir:       rootDir,
		Paths:         slicePaths,
		MaxTokens:     maxTokens,
		Depth:         depth,
		Target:        fv.Target,
		AssertInclude: fv.AssertIncludes,
		TokenCounter:  tokenCount,
		Compress:      fv.Compress,
	}

	result, err := workflows.GenerateModuleSlice(sliceOpts)
	if err != nil {
		return fmt.Errorf("slice: %w", err)
	}

	// Handle --json output.
	if sliceJSON {
		return writeSliceJSON(cmd, result, maxTokens)
	}

	// Route output.
	return writeSliceOutput(cmd, fv, result)
}

// writeSliceJSON writes the module slice metadata as JSON to stdout.
func writeSliceJSON(cmd *cobra.Command, result *workflows.ModuleSliceResult, maxTokens int) error {
	meta := workflows.ModuleSliceJSON{
		TokenCount:    result.TokenCount,
		ContentHash:   result.FormattedHash,
		ModuleFiles:   result.ModuleFiles,
		NeighborFiles: result.NeighborFiles,
		TotalFiles:    result.TotalFiles,
		MaxTokens:     maxTokens,
		Paths:         slicePaths,
	}

	// Ensure nil slices serialize as empty arrays.
	if meta.ModuleFiles == nil {
		meta.ModuleFiles = []string{}
	}
	if meta.NeighborFiles == nil {
		meta.NeighborFiles = []string{}
	}
	if meta.Paths == nil {
		meta.Paths = []string{}
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("slice: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeSliceOutput routes the module slice content to stdout or a file
// based on the flag configuration.
func writeSliceOutput(cmd *cobra.Command, fv *config.FlagValues, result *workflows.ModuleSliceResult) error {
	content := result.Content

	if fv.Stdout {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Determine output path.
	outputPath := fv.Output
	if outputPath == config.DefaultOutput {
		outputPath = "harvx-slice.md"
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("slice: writing output to %s: %w", outputPath, err)
	}

	// Report to stderr.
	fmt.Fprintf(cmd.ErrOrStderr(),
		"Module slice written to %s (%d tokens, hash: %s, %d module files, %d neighbors)\n",
		outputPath, result.TokenCount, result.FormattedHash,
		len(result.ModuleFiles), len(result.NeighborFiles))
	return nil
}
