// Package workflows implements high-level workflow commands for harvx.
// This file implements the verification workflow that compares packed output
// to original source files, verifying faithfulness after compression and
// redaction transformations.
package workflows

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/harvx/harvx/internal/output"
)

// VerifyStatus represents the verification result for a single file.
type VerifyStatus string

const (
	// VerifyMatch indicates the packed content exactly matches the original.
	VerifyMatch VerifyStatus = "MATCH"

	// VerifyRedactionDiff indicates differences are explained by redaction
	// placeholders replacing sensitive content.
	VerifyRedactionDiff VerifyStatus = "REDACTION_DIFF"

	// VerifyCompressionDiff indicates the file was compressed (signatures
	// only) and cannot be compared line-for-line.
	VerifyCompressionDiff VerifyStatus = "COMPRESSION_DIFF"

	// VerifyUnexpectedDiff indicates the packed content differs from the
	// original in ways not explained by redaction or compression.
	VerifyUnexpectedDiff VerifyStatus = "UNEXPECTED_DIFF"

	// VerifyFileChanged indicates the source file on disk has changed or
	// been deleted since the output was generated.
	VerifyFileChanged VerifyStatus = "FILE_CHANGED"
)

// FileVerification holds the result of verifying a single file.
type FileVerification struct {
	// Path is the relative file path that was verified.
	Path string `json:"path"`

	// Status is the verification result status.
	Status VerifyStatus `json:"status"`

	// Message is a human-readable description of the verification result.
	Message string `json:"message"`

	// DiffLines holds the first few lines of an unexpected diff for
	// debugging purposes. Only populated for VerifyUnexpectedDiff status.
	DiffLines []string `json:"diff_lines,omitempty"`

	// Redactions is the count of redaction placeholders found in the
	// packed content. Only populated for VerifyRedactionDiff status.
	Redactions int `json:"redactions,omitempty"`

	// Compressed indicates whether the file was marked as compressed
	// in the output.
	Compressed bool `json:"compressed,omitempty"`
}

// BudgetInfo holds token budget reporting data extracted from the metadata
// sidecar file.
type BudgetInfo struct {
	// Tokenizer is the tokenizer encoding used (e.g., "cl100k_base").
	Tokenizer string `json:"tokenizer"`

	// TotalTokens is the total token count of the output.
	TotalTokens int `json:"total_tokens"`

	// MaxTokens is the configured token budget. Zero means no budget.
	MaxTokens int `json:"max_tokens"`

	// BudgetUsedPct is the percentage of the token budget used.
	BudgetUsedPct float64 `json:"budget_used_percent"`

	// CompressedFiles is the number of files that had compression applied.
	CompressedFiles int `json:"compressed_files"`

	// RedactionsTotal is the total number of redactions across all files.
	RedactionsTotal int `json:"redactions_total"`
}

// VerifyResult holds the complete verification output.
type VerifyResult struct {
	// OutputPath is the path to the output file that was verified.
	OutputPath string `json:"output_path"`

	// TotalFiles is the total number of files found in the output.
	TotalFiles int `json:"total_files"`

	// SampledFiles is the number of files that were selected for verification.
	SampledFiles int `json:"sampled_files"`

	// PassedCount is the number of files that passed verification (MATCH,
	// REDACTION_DIFF, or COMPRESSION_DIFF statuses).
	PassedCount int `json:"passed_count"`

	// WarningCount is the number of files with unexpected differences or
	// changes (UNEXPECTED_DIFF or FILE_CHANGED statuses).
	WarningCount int `json:"warning_count"`

	// Files holds per-file verification results.
	Files []FileVerification `json:"files"`

	// Budget holds token budget reporting data, if available from the
	// metadata sidecar. Nil when no sidecar is found.
	Budget *BudgetInfo `json:"budget,omitempty"`
}

// VerifyOptions configures the verification workflow.
type VerifyOptions struct {
	// RootDir is the repository root directory.
	RootDir string

	// OutputPath is the path to the harvx output file to verify.
	OutputPath string

	// SampleSize is the number of files to randomly sample for verification.
	// Zero means verify all files.
	SampleSize int

	// Paths is a list of specific file paths to verify. When non-empty,
	// this overrides SampleSize-based sampling.
	Paths []string

	// Profile is the profile name for resolving settings (currently unused
	// but reserved for future use).
	Profile string
}

// maxDiffLines is the maximum number of diff lines to include in the
// verification result for unexpected differences.
const maxDiffLines = 10

// VerifyOutput runs the verification workflow. It reads the output file,
// parses it to extract file blocks, selects files for verification (via
// explicit paths or random sampling), and compares each packed file to its
// original source on disk.
func VerifyOutput(opts VerifyOptions) (*VerifyResult, error) {
	if opts.OutputPath == "" {
		return nil, fmt.Errorf("verify: output path required")
	}
	if opts.RootDir == "" {
		return nil, fmt.Errorf("verify: root directory required")
	}

	slog.Debug("starting verification",
		"output_path", opts.OutputPath,
		"root_dir", opts.RootDir,
		"sample_size", opts.SampleSize,
		"paths", opts.Paths,
	)

	// Step 1: Read the output file.
	contentBytes, err := os.ReadFile(opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("verify: reading output file %q: %w (did you mean to specify --output?)", opts.OutputPath, err)
	}
	content := string(contentBytes)

	// Step 2: Parse the output to extract file blocks.
	parsed, err := ParseOutput(content)
	if err != nil {
		return nil, fmt.Errorf("verify: parsing output: %w", err)
	}

	if len(parsed.Files) == 0 {
		slog.Info("verify: no files found in output", "path", opts.OutputPath)
		return &VerifyResult{
			OutputPath: opts.OutputPath,
		}, nil
	}

	slog.Debug("verify: parsed output",
		"format", parsed.Format,
		"total_files", len(parsed.Files),
	)

	// Step 3: Read metadata sidecar for budget info.
	budgetInfo := readBudgetInfo(opts.OutputPath)

	// Step 4: Select files to verify.
	selected := selectFilesForVerification(parsed.Files, opts, contentBytes)

	slog.Debug("verify: files selected for verification",
		"total", len(parsed.Files),
		"selected", len(selected),
	)

	// Step 5: Verify each selected file.
	verifications := make([]FileVerification, 0, len(selected))
	passedCount := 0
	warningCount := 0

	for _, pf := range selected {
		v := verifyFile(opts.RootDir, pf)
		verifications = append(verifications, v)

		switch v.Status {
		case VerifyMatch, VerifyRedactionDiff, VerifyCompressionDiff:
			passedCount++
		case VerifyUnexpectedDiff, VerifyFileChanged:
			warningCount++
		}
	}

	result := &VerifyResult{
		OutputPath:   opts.OutputPath,
		TotalFiles:   len(parsed.Files),
		SampledFiles: len(selected),
		PassedCount:  passedCount,
		WarningCount: warningCount,
		Files:        verifications,
		Budget:       budgetInfo,
	}

	slog.Info("verification complete",
		"output_path", opts.OutputPath,
		"total_files", len(parsed.Files),
		"sampled", len(selected),
		"passed", passedCount,
		"warnings", warningCount,
	)

	return result, nil
}

// selectFilesForVerification selects which parsed files should be verified
// based on the options. If explicit Paths are specified, only those files
// are selected. Otherwise, random sampling is applied using a reproducible
// seed derived from the output file content.
func selectFilesForVerification(files []ParsedFile, opts VerifyOptions, contentBytes []byte) []ParsedFile {
	// If explicit paths are specified, filter by those paths.
	if len(opts.Paths) > 0 {
		pathSet := make(map[string]bool, len(opts.Paths))
		for _, p := range opts.Paths {
			pathSet[filepath.ToSlash(p)] = true
		}

		var selected []ParsedFile
		for _, f := range files {
			if pathSet[filepath.ToSlash(f.Path)] {
				selected = append(selected, f)
			}
		}
		return selected
	}

	// If sample size is zero or >= file count, verify all.
	if opts.SampleSize <= 0 || opts.SampleSize >= len(files) {
		result := make([]ParsedFile, len(files))
		copy(result, files)
		return result
	}

	// Reproducible random sampling using content hash as seed.
	h := fnv.New64a()
	h.Write(contentBytes)
	seed := h.Sum64()

	// Copy the file list and shuffle using Fisher-Yates.
	shuffled := make([]ParsedFile, len(files))
	copy(shuffled, files)

	rng := rand.New(rand.NewPCG(seed, seed))
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:opts.SampleSize]
}

// verifyFile compares a single parsed file from the output to its original
// source on disk and returns the verification result.
func verifyFile(rootDir string, pf ParsedFile) FileVerification {
	absPath := filepath.Join(rootDir, filepath.FromSlash(pf.Path))

	originalBytes, err := os.ReadFile(absPath)
	if err != nil {
		slog.Debug("verify: source file not readable",
			"path", pf.Path,
			"error", err,
		)
		return FileVerification{
			Path:    pf.Path,
			Status:  VerifyFileChanged,
			Message: "source file not found (may have been deleted since generation)",
		}
	}

	original := string(originalBytes)

	// If the file was compressed, we cannot re-compress without tree-sitter,
	// so report it as a known compression diff.
	if pf.IsCompressed {
		return FileVerification{
			Path:       pf.Path,
			Status:     VerifyCompressionDiff,
			Message:    "Match (compressed: signatures only)",
			Compressed: true,
		}
	}

	// Normalize trailing newlines for comparison. The Markdown/XML template
	// rendering may add or strip a trailing newline, so we normalize both
	// sides to avoid false positives.
	normalizedOriginal := strings.TrimRight(original, "\n")
	normalizedPacked := strings.TrimRight(pf.Content, "\n")

	// Exact match check (after newline normalization).
	if normalizedOriginal == normalizedPacked {
		return FileVerification{
			Path:    pf.Path,
			Status:  VerifyMatch,
			Message: "Match",
		}
	}

	// Check if differences are explained by redaction.
	if pf.Redactions > 0 && isRedactionDiff(normalizedOriginal, normalizedPacked) {
		return FileVerification{
			Path:       pf.Path,
			Status:     VerifyRedactionDiff,
			Message:    fmt.Sprintf("Match (%d redactions applied)", pf.Redactions),
			Redactions: pf.Redactions,
		}
	}

	// Unexpected difference. Generate a simple diff using normalized content.
	diffLines := simpleDiff(normalizedOriginal, normalizedPacked, maxDiffLines)

	return FileVerification{
		Path:      pf.Path,
		Status:    VerifyUnexpectedDiff,
		Message:   "Unexpected difference detected",
		DiffLines: diffLines,
	}
}

// verifyRedactionRe matches [REDACTED], [REDACTED:type], or [REDACTED:type:detail]
// patterns in content for redaction detection.
var verifyRedactionRe = regexp.MustCompile(`\[REDACTED[^\]]*\]`)

// isRedactionDiff checks whether the differences between original and packed
// content can be explained by redaction patterns. It replaces all redaction
// placeholders in the packed content and compares the remaining structure
// to the original. If the non-redacted portions match closely, the diff is
// considered a redaction diff.
func isRedactionDiff(original, packed string) bool {
	// Split both into lines for comparison.
	origLines := strings.Split(original, "\n")
	packedLines := strings.Split(packed, "\n")

	// If line counts differ significantly, it's not just redaction.
	if abs(len(origLines)-len(packedLines)) > 1 {
		return false
	}

	// Compare line by line. For each line that differs, check if the
	// difference can be explained by a redaction placeholder.
	minLen := len(origLines)
	if len(packedLines) < minLen {
		minLen = len(packedLines)
	}

	unexplainedDiffs := 0
	for i := 0; i < minLen; i++ {
		if origLines[i] == packedLines[i] {
			continue
		}

		// Remove redaction placeholders from the packed line.
		cleaned := verifyRedactionRe.ReplaceAllString(packedLines[i], "")
		// Also remove from original in case the original contained the
		// literal text that was redacted.
		origCleaned := strings.TrimSpace(origLines[i])
		packedCleaned := strings.TrimSpace(cleaned)

		// If the cleaned packed line is a substring of the original line
		// (or vice versa after trimming), the diff is explained by redaction.
		if packedCleaned == "" {
			// The entire line was redacted -- this is explained.
			continue
		}

		// Check if removing the redaction placeholder from the packed line
		// leaves something that's present in the original.
		if strings.Contains(origCleaned, packedCleaned) || strings.Contains(packedCleaned, origCleaned) {
			continue
		}

		unexplainedDiffs++
	}

	// Allow a small tolerance for minor formatting differences around
	// redaction sites.
	return unexplainedDiffs == 0
}

// simpleDiff produces a simple line-based diff between two strings. For each
// differing line, it outputs "- <original line>" and "+ <packed line>". Returns
// at most maxLines diff lines total.
func simpleDiff(original, packed string, maxLines int) []string {
	origLines := strings.Split(original, "\n")
	packedLines := strings.Split(packed, "\n")

	var diffLines []string

	maxLen := len(origLines)
	if len(packedLines) > maxLen {
		maxLen = len(packedLines)
	}

	for i := 0; i < maxLen && len(diffLines) < maxLines; i++ {
		var origLine, packedLine string
		hasOrig := i < len(origLines)
		hasPacked := i < len(packedLines)

		if hasOrig {
			origLine = origLines[i]
		}
		if hasPacked {
			packedLine = packedLines[i]
		}

		if hasOrig && hasPacked && origLine == packedLine {
			continue
		}

		if hasOrig && len(diffLines) < maxLines {
			diffLines = append(diffLines, fmt.Sprintf("- %s", origLine))
		}
		if hasPacked && len(diffLines) < maxLines {
			diffLines = append(diffLines, fmt.Sprintf("+ %s", packedLine))
		}
	}

	return diffLines
}

// readBudgetInfo reads the metadata sidecar file (.meta.json) for the given
// output path and extracts budget information. Returns nil if the sidecar
// cannot be read or parsed.
func readBudgetInfo(outputPath string) *BudgetInfo {
	sidecarPath := output.MetadataSidecarPath(outputPath)

	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		slog.Debug("verify: metadata sidecar not found",
			"path", sidecarPath,
			"error", err,
		)
		return nil
	}

	var meta output.OutputMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		slog.Debug("verify: metadata sidecar parse error",
			"path", sidecarPath,
			"error", err,
		)
		return nil
	}

	budgetUsedPct := 0.0
	if meta.Statistics.BudgetUsedPercent != nil {
		budgetUsedPct = *meta.Statistics.BudgetUsedPercent
	}

	return &BudgetInfo{
		Tokenizer:       meta.Tokenizer,
		TotalTokens:     meta.Statistics.TotalTokens,
		MaxTokens:       meta.Statistics.MaxTokens,
		BudgetUsedPct:   budgetUsedPct,
		CompressedFiles: meta.Statistics.CompressedFiles,
		RedactionsTotal: meta.Statistics.RedactionsTotal,
	}
}

// abs returns the absolute value of an integer.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
