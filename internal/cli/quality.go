// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx quality` (alias: `qa`) subcommand which
// evaluates golden questions coverage, and the `harvx quality init` subcommand
// which generates a starter golden questions TOML file.
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

// qualityJSON is a local flag target for --json on the quality command.
// When true, quality results are output as structured JSON.
var qualityJSON bool

// qualityQuestionsPath is a local flag target for --questions on the quality
// command. When set, it overrides auto-discovery of the golden questions file.
var qualityQuestionsPath string

// qualityInitOutput is a local flag target for --output on the quality init
// subcommand. It specifies the output path for the generated golden questions
// file (default: .harvx/golden-questions.toml).
var qualityInitOutput string

// qualityInitYes is a local flag target for --yes on the quality init
// subcommand. When true, existing files are overwritten without prompting.
var qualityInitYes bool

// qualityCmd implements `harvx quality` (alias: `qa`) which evaluates
// golden questions coverage against the repository.
var qualityCmd = &cobra.Command{
	Use:     "quality",
	Aliases: []string{"qa"},
	Short:   "Evaluate golden questions coverage",
	Long: `Evaluate how well your Harvx context covers a set of golden questions.

Each golden question pairs a natural-language query with the critical files
an LLM needs to answer it. The quality command checks whether those files
exist in the repository and reports per-question coverage.

Examples:
  # Evaluate coverage using auto-discovered golden questions
  harvx quality

  # Output results as structured JSON
  harvx quality --json

  # Use a custom golden questions file
  harvx quality --questions path/to/questions.toml

  # Generate a starter golden questions file
  harvx quality init`,
	RunE: runQuality,
}

// qualityInitCmd implements `harvx quality init` which generates a starter
// golden questions TOML file.
var qualityInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter golden questions file",
	RunE:  runQualityInit,
}

func init() {
	qualityCmd.Flags().BoolVar(&qualityJSON, "json", false, "Output structured JSON")
	qualityCmd.Flags().StringVar(&qualityQuestionsPath, "questions", "", "Path to golden questions TOML (overrides auto-discovery)")

	qualityInitCmd.Flags().StringVar(&qualityInitOutput, "output", ".harvx/golden-questions.toml", "Output path for the generated file")
	qualityInitCmd.Flags().BoolVar(&qualityInitYes, "yes", false, "Overwrite without prompting")

	qualityCmd.AddCommand(qualityInitCmd)
	rootCmd.AddCommand(qualityCmd)
}

// runQuality executes the quality subcommand. It resolves the root directory,
// invokes the quality evaluation workflow, and renders either a human-readable
// report or structured JSON to stdout.
func runQuality(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("quality: resolving directory: %w", err)
	}

	slog.Debug("quality: resolved paths",
		"root_dir", rootDir,
		"questions_path", qualityQuestionsPath,
		"json", qualityJSON,
		"profile", fv.Profile,
	)

	opts := workflows.QualityOptions{
		RootDir:       rootDir,
		QuestionsPath: qualityQuestionsPath,
		ProfileName:   fv.Profile,
	}

	result, err := workflows.EvaluateQuality(opts)
	if err != nil {
		return fmt.Errorf("quality: %w", err)
	}

	// Handle --json output.
	if qualityJSON {
		return writeQualityJSON(cmd, result)
	}

	// Render human-readable report.
	return writeQualityReport(cmd, result)
}

// writeQualityJSON marshals the quality result as indented JSON to stdout.
func writeQualityJSON(cmd *cobra.Command, result *workflows.QualityResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("quality: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeQualityReport renders a human-readable quality coverage report to
// stdout. Each question is shown with a [PASS] or [MISS] label, and missed
// questions list their missing files indented below.
func writeQualityReport(cmd *cobra.Command, result *workflows.QualityResult) error {
	w := cmd.OutOrStdout()

	// Header line.
	fmt.Fprintf(w, "Golden Questions Coverage\n\n")

	// Per-question results.
	for _, qr := range result.Questions {
		label := qualityStatusLabel(qr.Covered)
		detail := "all critical files found"
		if !qr.Covered {
			detail = fmt.Sprintf("%d/%d critical files missing", len(qr.MissingFiles), len(qr.CriticalFiles))
		}

		fmt.Fprintf(w, "%s %s - %s\n", label, qr.ID, detail)

		// For missed questions, list missing files indented.
		if !qr.Covered {
			for _, f := range qr.MissingFiles {
				fmt.Fprintf(w, "      %s\n", f)
			}
		}
	}

	// Summary line.
	fmt.Fprintf(w, "\nCoverage: %d/%d questions covered (%.1f%%)\n",
		result.CoveredCount, result.TotalQuestions, result.CoveragePercent)

	return nil
}

// qualityStatusLabel returns the display label for a question coverage status.
func qualityStatusLabel(covered bool) string {
	if covered {
		return "[PASS]"
	}
	return "[MISS]"
}

// runQualityInit executes the quality init subcommand. It generates a starter
// golden questions TOML file at the specified path, creating parent
// directories as needed.
func runQualityInit(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("quality init: resolving directory: %w", err)
	}

	// Resolve output path relative to the root directory.
	outputPath := qualityInitOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(rootDir, outputPath)
	}

	slog.Debug("quality init: resolved output path",
		"output", outputPath,
		"yes", qualityInitYes,
	)

	// Check if the file already exists.
	if _, statErr := os.Stat(outputPath); statErr == nil && !qualityInitYes {
		return fmt.Errorf("quality init: file already exists, use --yes to overwrite")
	}

	// Create parent directories.
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("quality init: creating directory %s: %w", dir, err)
	}

	// Generate and write the starter content.
	content := config.GenerateGoldenQuestionsInit()
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("quality init: writing file %s: %w", outputPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created golden questions file: %s\n", outputPath)
	return nil
}
