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
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/workflows"
)

// qualityJSON is a local flag target for --json on the quality command.
var qualityJSON bool

// qualityQuestionsPath is a local flag target for --questions.
var qualityQuestionsPath string

// qualityYes is a local flag target for --yes on the quality init subcommand.
var qualityYes bool

// qualityCmd implements `harvx quality` (alias: `qa`) which evaluates
// golden questions coverage.
var qualityCmd = &cobra.Command{
	Use:     "quality",
	Aliases: []string{"qa"},
	Short:   "Evaluate golden questions coverage",
	Long: `Evaluate coverage of golden questions against the current repository.

For each golden question, the command checks whether the critical files are
present in the repository. Coverage reports show which questions have all
their critical files included and which are missing files.

Golden questions are loaded from .harvx/golden-questions.toml by default.
Use --questions to specify a custom path.

Examples:
  # Run quality check
  harvx quality

  # Use a custom questions file
  harvx quality --questions path/to/questions.toml

  # JSON output for CI
  harvx quality --json

  # Initialize a starter golden questions file
  harvx quality init`,
	RunE: runQuality,
}

// qualityInitCmd implements `harvx quality init` which generates a starter
// golden questions file.
var qualityInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter golden questions file",
	Long: `Generate a starter golden questions TOML file with example questions.

The file is written to .harvx/golden-questions.toml by default. If the file
already exists, the command exits with an error unless --yes is specified.`,
	RunE: runQualityInit,
}

func init() {
	qualityCmd.Flags().BoolVar(&qualityJSON, "json", false, "Output results as structured JSON")
	qualityCmd.Flags().StringVar(&qualityQuestionsPath, "questions", "", "Path to golden questions TOML file")
	qualityInitCmd.Flags().BoolVar(&qualityYes, "yes", false, "Overwrite existing golden questions file without prompting")
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

	slog.Debug("quality: starting evaluation",
		"root_dir", rootDir,
		"questions_path", qualityQuestionsPath,
		"profile", fv.Profile,
	)

	opts := workflows.QualityOptions{
		RootDir:       rootDir,
		QuestionsPath: qualityQuestionsPath,
		Profile:       fv.Profile,
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

// writeQualityReport renders a human-readable quality coverage report to stdout.
func writeQualityReport(cmd *cobra.Command, result *workflows.QualityResult) error {
	w := cmd.OutOrStdout()

	// Header line.
	questionsBase := filepath.Base(result.QuestionsPath)
	fmt.Fprintf(w, "Golden Questions Coverage (%d questions from %s)\n\n",
		result.TotalQuestions, questionsBase)

	// Per-question results.
	for _, qr := range result.Questions {
		label := qualityStatusLabel(qr.Covered)
		fmt.Fprintf(w, "%s %-20s %s\n", label, qr.ID, qr.Question)

		if qr.Covered {
			if len(qr.FoundFiles) > 0 {
				fmt.Fprintf(w, "      found: %s\n", strings.Join(qr.FoundFiles, ", "))
			}
		} else {
			if len(qr.FoundFiles) > 0 {
				fmt.Fprintf(w, "      found:   %s\n", strings.Join(qr.FoundFiles, ", "))
			}
			if len(qr.MissingFiles) > 0 {
				fmt.Fprintf(w, "      missing: %s\n", strings.Join(qr.MissingFiles, ", "))
			}
		}
	}

	// Per-category summary table.
	if len(result.ByCategory) > 0 {
		fmt.Fprintf(w, "\nBy Category:\n")

		// Sort category names for deterministic output.
		categories := make([]string, 0, len(result.ByCategory))
		for cat := range result.ByCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		for _, cat := range categories {
			stats := result.ByCategory[cat]
			fmt.Fprintf(w, "  %-20s %d/%d (%.0f%%)\n", cat, stats.Covered, stats.Total, stats.Percent)
		}
	}

	// Overall coverage line.
	fmt.Fprintf(w, "\nCoverage: %d/%d (%.0f%%)\n",
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
// golden questions TOML file at .harvx/golden-questions.toml.
func runQualityInit(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("quality init: resolving directory: %w", err)
	}

	// Build the output path.
	harvxDir := filepath.Join(rootDir, ".harvx")
	outputPath := filepath.Join(harvxDir, "golden-questions.toml")

	// Check if the file already exists.
	if _, statErr := os.Stat(outputPath); statErr == nil {
		// File exists. Check --yes flags (local and global).
		if !qualityYes && !fv.Yes {
			return fmt.Errorf("quality init: %s already exists; use --yes to overwrite", outputPath)
		}
		slog.Debug("quality init: overwriting existing file",
			"path", outputPath,
		)
	}

	// Create the .harvx directory if needed.
	if err := os.MkdirAll(harvxDir, 0o755); err != nil {
		return fmt.Errorf("quality init: creating directory %s: %w", harvxDir, err)
	}

	// Generate and write the starter content.
	content := config.GenerateGoldenQuestionsInit()
	if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("quality init: writing %s: %w", outputPath, err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Created %s with 3 example golden questions\n", outputPath)
	fmt.Fprintf(cmd.ErrOrStderr(), "Edit the file to add questions specific to your project.\n")

	return nil
}
