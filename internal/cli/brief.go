// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx brief` subcommand which generates a stable,
// small Repo Brief artifact containing project-wide invariants.
package cli

import (
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

// briefJSON is a local flag target for --json on the brief command.
// When true, brief outputs machine-readable JSON metadata to stdout.
var briefJSON bool

// briefCmd implements `harvx brief` which generates a stable Repo Brief
// artifact containing project-wide invariants.
var briefCmd = &cobra.Command{
	Use:   "brief",
	Short: "Generate a stable Repo Brief with project-wide invariants",
	Long: `Generate a small, stable Repo Brief artifact (~1-4K tokens) containing
project-wide invariants suitable for LLM context injection.

The brief includes:
  - README / high-level product overview
  - Architecture docs and ADRs
  - Build/test commands (Makefile targets, package.json scripts)
  - Key invariants (CLAUDE.md, CONVENTIONS.md)
  - High-level module map

The output is deterministic and content-addressed via XXH3 hash,
enabling prompt caching across commits.

Examples:
  # Generate brief to stdout
  harvx brief --stdout

  # Generate brief with custom token budget
  harvx brief --max-tokens 8000 --stdout

  # Machine-readable JSON metadata
  harvx brief --json

  # XML output for Claude
  harvx brief --target claude --stdout

  # Save to file
  harvx brief -o project-brief.md

  # Verify coverage
  harvx brief --assert-include "README.md" --stdout`,
	RunE: runBrief,
}

func init() {
	briefCmd.Flags().BoolVar(&briefJSON, "json", false, "Output machine-readable JSON metadata to stdout")
	rootCmd.AddCommand(briefCmd)
}

// runBrief executes the brief subcommand. It resolves configuration, generates
// the brief, and routes output to stdout or a file.
func runBrief(cmd *cobra.Command, args []string) error {
	fv := GlobalFlags()

	// Sync the local flag back to FlagValues.
	fv.PreviewJSON = briefJSON

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("brief: resolving directory: %w", err)
	}

	// Resolve the brief max tokens from profile configuration.
	maxTokens := resolveBriefMaxTokens(fv)

	// Build the token counter.
	tokenCount, err := buildBriefTokenCounter(fv.Tokenizer)
	if err != nil {
		slog.Warn("brief: could not create tokenizer, using estimator",
			"tokenizer", fv.Tokenizer,
			"error", err,
		)
		tokenCount = nil // Will use the built-in estimator.
	}

	briefOpts := workflows.BriefOptions{
		RootDir:       rootDir,
		MaxTokens:     maxTokens,
		Target:        fv.Target,
		AssertInclude: fv.AssertIncludes,
		TokenCounter:  tokenCount,
	}

	result, err := workflows.GenerateBrief(briefOpts)
	if err != nil {
		return fmt.Errorf("brief: %w", err)
	}

	// Handle --json output.
	if briefJSON {
		return writeBriefJSON(cmd, result, maxTokens)
	}

	// Route output.
	return writeBriefOutput(cmd, fv, result)
}

// resolveBriefMaxTokens determines the token budget for the brief. It checks
// the profile configuration first, then falls back to the default.
func resolveBriefMaxTokens(fv *config.FlagValues) int {
	// Try to resolve from profile config.
	rc, err := config.Resolve(config.ResolveOptions{
		ProfileName: fv.Profile,
		TargetDir:   fv.Dir,
	})
	if err == nil && rc.Profile != nil && rc.Profile.BriefMaxTokens > 0 {
		return rc.Profile.BriefMaxTokens
	}

	// If --max-tokens was explicitly set and is reasonable for brief, use it.
	if fv.MaxTokens > 0 && fv.MaxTokens <= 32000 {
		return fv.MaxTokens
	}

	return workflows.DefaultBriefMaxTokens
}

// buildBriefTokenCounter creates a token counter function from the configured
// tokenizer name. Returns nil if the tokenizer cannot be created.
func buildBriefTokenCounter(name string) (func(string) int, error) {
	if name == "none" || name == "" {
		return nil, nil
	}

	tok, err := tokenizer.NewTokenizer(name)
	if err != nil {
		return nil, fmt.Errorf("creating tokenizer %q: %w", name, err)
	}

	return tok.Count, nil
}

// writeBriefJSON writes the brief metadata as JSON to stdout.
func writeBriefJSON(cmd *cobra.Command, result *workflows.BriefResult, maxTokens int) error {
	sectionNames := make([]string, 0, len(result.Sections))
	for _, sec := range result.Sections {
		sectionNames = append(sectionNames, sec.Name)
	}

	briefMeta := workflows.BriefJSON{
		TokenCount:    result.TokenCount,
		ContentHash:   result.FormattedHash,
		FilesIncluded: result.FilesIncluded,
		Sections:      sectionNames,
		MaxTokens:     maxTokens,
	}

	data, err := json.MarshalIndent(briefMeta, "", "  ")
	if err != nil {
		return fmt.Errorf("brief: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeBriefOutput routes the brief content to stdout or a file based on
// the flag configuration.
func writeBriefOutput(cmd *cobra.Command, fv *config.FlagValues, result *workflows.BriefResult) error {
	content := result.Content

	if fv.Stdout {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Determine output path.
	outputPath := fv.Output
	if outputPath == config.DefaultOutput {
		// Use a brief-specific default output name.
		outputPath = "harvx-brief.md"
	}

	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("brief: writing output to %s: %w", outputPath, err)
	}

	// Report to stderr.
	fmt.Fprintf(cmd.ErrOrStderr(), "Brief written to %s (%d tokens, hash: %s)\n",
		outputPath, result.TokenCount, result.FormattedHash)
	return nil
}
