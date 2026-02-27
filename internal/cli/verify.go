// Package cli implements the Cobra command hierarchy for the harvx CLI tool.
// This file implements the `harvx verify` subcommand which verifies that harvx
// output faithfully represents the original source files by comparing packed
// content against the source on disk, accounting for expected transformations
// such as compression and redaction.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/workflows"
)

// verifySampleSize is a local flag target for --sample on the verify command.
// It controls how many files are randomly sampled for verification.
var verifySampleSize int

// verifyPaths is a local flag target for --path on the verify command (repeatable).
// When specified, only these files are verified instead of random sampling.
var verifyPaths []string

// verifyJSON is a local flag target for --json on the verify command.
// When true, verification results are output as structured JSON.
var verifyJSON bool

// verifyCmd implements `harvx verify` which verifies faithfulness of harvx
// output to the original source files.
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify faithfulness of harvx output to source files",
	Long: `Verify that harvx output faithfully represents your source files.

For each sampled file, the verify command reads the original source from disk
and compares it against the packed version in the output document. Expected
transformations (compression, redaction) are identified; any unexpected
differences are flagged as warnings.

Examples:
  # Verify default output file (sample 10 files)
  harvx verify

  # Verify all files
  harvx verify --sample 0

  # Verify specific files
  harvx verify --path src/main.go --path src/config.go

  # JSON output for CI pipelines
  harvx verify --json

  # Verify output from a specific profile
  harvx verify --profile security-review`,
	RunE: runVerify,
}

func init() {
	verifyCmd.Flags().IntVar(&verifySampleSize, "sample", 10, "Number of files to randomly sample for verification (0 = all)")
	verifyCmd.Flags().StringArrayVar(&verifyPaths, "path", nil, "Specific file paths to verify (repeatable)")
	verifyCmd.Flags().BoolVar(&verifyJSON, "json", false, "Output verification results as structured JSON")
	rootCmd.AddCommand(verifyCmd)
}

// runVerify executes the verify subcommand. It resolves the root directory and
// output path, invokes the verification workflow, and renders either a
// human-readable report or structured JSON to stdout.
func runVerify(cmd *cobra.Command, _ []string) error {
	fv := GlobalFlags()

	rootDir, err := filepath.Abs(fv.Dir)
	if err != nil {
		return fmt.Errorf("verify: resolving directory: %w", err)
	}

	// Resolve the output path to verify. If the user did not specify --output,
	// use the default output file name.
	outputPath := fv.Output
	if outputPath == "" {
		outputPath = config.DefaultOutput
	}

	// Make the output path absolute relative to the root directory so the
	// workflow can find it regardless of the working directory.
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(rootDir, outputPath)
	}

	slog.Debug("verify: resolved paths",
		"root_dir", rootDir,
		"output_path", outputPath,
		"sample_size", verifySampleSize,
		"paths", verifyPaths,
	)

	opts := workflows.VerifyOptions{
		RootDir:    rootDir,
		OutputPath: outputPath,
		SampleSize: verifySampleSize,
		Paths:      verifyPaths,
		Profile:    fv.Profile,
	}

	result, err := workflows.VerifyOutput(opts)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	// Handle --json output.
	if verifyJSON {
		return writeVerifyJSON(cmd, result)
	}

	// Render human-readable report and return appropriate exit code.
	return writeVerifyReport(cmd, result)
}

// writeVerifyJSON marshals the verification result as indented JSON to stdout.
func writeVerifyJSON(cmd *cobra.Command, result *workflows.VerifyResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("verify: marshaling JSON: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))

	// Return partial error if there were warnings, so the exit code is 2.
	if result.WarningCount > 0 {
		return pipeline.NewPartialError(
			fmt.Sprintf("verify: %d/%d files had warnings", result.WarningCount, result.SampledFiles),
			nil,
		)
	}

	return nil
}

// writeVerifyReport renders a human-readable verification report to stdout.
// It returns a partial error (exit code 2) if any files had warnings.
func writeVerifyReport(cmd *cobra.Command, result *workflows.VerifyResult) error {
	w := cmd.OutOrStdout()

	// Header line.
	outputBase := filepath.Base(result.OutputPath)
	fmt.Fprintf(w, "Verifying %s (%d sampled files)\n\n", outputBase, result.SampledFiles)

	// Per-file results.
	for _, fv := range result.Files {
		label := verifyStatusLabel(fv.Status)
		fmt.Fprintf(w, "%s %-40s - %s\n", label, fv.Path, fv.Message)

		// For unexpected diffs, print the diff lines indented.
		if fv.Status == workflows.VerifyUnexpectedDiff && len(fv.DiffLines) > 0 {
			for _, line := range fv.DiffLines {
				fmt.Fprintf(w, "      %s\n", line)
			}
		}
	}

	// Summary line.
	fmt.Fprintf(w, "\nResult: %d/%d passed, %d warning%s\n",
		result.PassedCount, result.SampledFiles,
		result.WarningCount, pluralS(result.WarningCount),
	)

	// Budget info section.
	if result.Budget != nil {
		writeVerifyBudgetLine(w, result.Budget)
	}

	// Return partial error if there were warnings, so the exit code is 2.
	if result.WarningCount > 0 {
		return pipeline.NewPartialError(
			fmt.Sprintf("verify: %d/%d files had warnings", result.WarningCount, result.SampledFiles),
			nil,
		)
	}

	return nil
}

// verifyStatusLabel returns the display label for a verification status.
// MATCH, REDACTION_DIFF, and COMPRESSION_DIFF are considered passing.
// UNEXPECTED_DIFF and FILE_CHANGED are considered warnings.
func verifyStatusLabel(status workflows.VerifyStatus) string {
	switch status {
	case workflows.VerifyMatch, workflows.VerifyRedactionDiff, workflows.VerifyCompressionDiff:
		return "[PASS]"
	case workflows.VerifyUnexpectedDiff, workflows.VerifyFileChanged:
		return "[WARN]"
	default:
		return "[????]"
	}
}

// writeVerifyBudgetLine writes a single-line budget summary to the writer.
func writeVerifyBudgetLine(w io.Writer, b *workflows.BudgetInfo) {
	var parts []string

	if b.Tokenizer != "" {
		parts = append(parts, b.Tokenizer)
	}

	if b.MaxTokens > 0 {
		parts = append(parts, fmt.Sprintf("%s / %s tokens (%.1f%%)",
			formatNumber(b.TotalTokens), formatNumber(b.MaxTokens), b.BudgetUsedPct))
	} else {
		parts = append(parts, fmt.Sprintf("%s tokens", formatNumber(b.TotalTokens)))
	}

	if b.CompressedFiles > 0 {
		parts = append(parts, fmt.Sprintf("%d compressed", b.CompressedFiles))
	}

	if b.RedactionsTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d redactions", b.RedactionsTotal))
	}

	fmt.Fprintf(w, "\nBudget: %s\n", strings.Join(parts, " | "))
}

// formatNumber formats an integer with comma separators for readability.
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas from right to left.
	var result []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(ch))
	}
	return string(result)
}

// pluralS returns "s" when count != 1, for simple English pluralization.
func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
