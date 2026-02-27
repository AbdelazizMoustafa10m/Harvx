package output

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// MetadataVersion is the schema version for the metadata sidecar JSON.
const MetadataVersion = "1.0.0"

// OutputMetadata is the top-level structure written to the .meta.json sidecar
// file. It provides machine-readable metadata about a generated context
// document for downstream pipeline consumption.
type OutputMetadata struct {
	// Version is the metadata schema version (currently "1.0.0").
	Version string `json:"version"`

	// GeneratedAt is the RFC 3339 timestamp of when the context was generated.
	GeneratedAt string `json:"generated_at"`

	// Profile is the config profile name used for generation.
	Profile string `json:"profile"`

	// Tokenizer is the tokenizer encoding used (e.g., "cl100k_base").
	Tokenizer string `json:"tokenizer"`

	// Format is the output format: "markdown" or "xml".
	Format string `json:"format"`

	// Target is the target LLM (e.g., "claude").
	Target string `json:"target"`

	// ContentHash is the hex-encoded XXH3 content hash of the rendered output.
	ContentHash string `json:"content_hash"`

	// Statistics holds aggregate statistics about the generated output.
	Statistics Statistics `json:"statistics"`

	// Files holds per-file statistics, sorted by path.
	Files []FileStats `json:"files"`
}

// Statistics holds aggregate statistics about the generated context document.
type Statistics struct {
	// TotalFiles is the number of files included in the output.
	TotalFiles int `json:"total_files"`

	// TotalTokens is the sum of all file token counts.
	TotalTokens int `json:"total_tokens"`

	// TotalBytes is the total size of all files in bytes.
	TotalBytes int64 `json:"total_bytes"`

	// BudgetUsedPercent is the percentage of the token budget used.
	// Nil when no budget (MaxTokens == 0) is set, serialized as null.
	BudgetUsedPercent *float64 `json:"budget_used_percent"`

	// MaxTokens is the maximum token budget. Zero means no budget was set.
	MaxTokens int `json:"max_tokens"`

	// FilesByTier maps tier number (as string key) to the count of files
	// in that tier. Always a non-nil map (empty {} when no tiers).
	FilesByTier map[string]int `json:"files_by_tier"`

	// RedactionsTotal is the total number of redactions across all files.
	RedactionsTotal int `json:"redactions_total"`

	// RedactionsByType maps redaction type labels to their counts.
	// Always a non-nil map (empty {} when no redactions).
	RedactionsByType map[string]int `json:"redactions_by_type"`

	// CompressedFiles is the number of files that had compression applied.
	CompressedFiles int `json:"compressed_files"`

	// GenerationTimeMs is the pipeline generation time in milliseconds.
	GenerationTimeMs int64 `json:"generation_time_ms"`
}

// FileStats holds per-file statistics in the metadata sidecar.
type FileStats struct {
	// Path is the file's relative path.
	Path string `json:"path"`

	// Tier is the relevance tier (0-5).
	Tier int `json:"tier"`

	// Tokens is the token count after processing.
	Tokens int `json:"tokens"`

	// Bytes is the file size in bytes.
	Bytes int64 `json:"bytes"`

	// Redactions is the number of secrets redacted from this file.
	Redactions int `json:"redactions"`

	// Compressed indicates whether compression was applied to this file.
	Compressed bool `json:"compressed"`

	// Language is the detected programming language.
	Language string `json:"language"`
}

// MetadataOpts holds the inputs needed to generate an OutputMetadata.
type MetadataOpts struct {
	// RenderData is the render data used to produce the context document.
	RenderData *RenderData

	// Result is the output result from writing the context document.
	Result *OutputResult

	// Format is the output format ("markdown" or "xml").
	Format string

	// Target is the target LLM identifier (e.g., "claude").
	Target string

	// MaxTokens is the token budget. Zero means no budget was set.
	MaxTokens int

	// GenerationTimeMs is the pipeline generation time in milliseconds.
	GenerationTimeMs int64
}

// GenerateMetadata assembles an OutputMetadata from the provided options.
// The returned metadata has all maps initialized (never nil) and the Files
// slice sorted by path.
func GenerateMetadata(opts MetadataOpts) *OutputMetadata {
	meta := &OutputMetadata{
		Version:     MetadataVersion,
		GeneratedAt: opts.RenderData.Timestamp.Format(time.RFC3339),
		Profile:     opts.RenderData.ProfileName,
		Tokenizer:   opts.RenderData.TokenizerName,
		Format:      opts.Format,
		Target:      opts.Target,
		ContentHash: opts.Result.HashHex,
	}

	// Build per-file stats.
	files := make([]FileStats, 0, len(opts.RenderData.Files))
	var totalBytes int64
	compressedCount := 0

	for _, f := range opts.RenderData.Files {
		files = append(files, FileStats{
			Path:       f.Path,
			Tier:       f.Tier,
			Tokens:     f.TokenCount,
			Bytes:      f.Size,
			Redactions: f.Redactions,
			Compressed: f.IsCompressed,
			Language:   f.Language,
		})
		totalBytes += f.Size
		if f.IsCompressed {
			compressedCount++
		}
	}

	// Sort files by path for deterministic output.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	meta.Files = files

	// Build files_by_tier map with string keys.
	filesByTier := make(map[string]int, len(opts.RenderData.TierCounts))
	for tier, count := range opts.RenderData.TierCounts {
		filesByTier[strconv.Itoa(tier)] = count
	}

	// Build redactions_by_type map, ensuring non-nil.
	redactionsByType := make(map[string]int, len(opts.RenderData.RedactionSummary))
	for typ, count := range opts.RenderData.RedactionSummary {
		redactionsByType[typ] = count
	}

	// Compute budget used percent.
	var budgetUsedPercent *float64
	if opts.MaxTokens > 0 {
		pct := (float64(opts.RenderData.TotalTokens) / float64(opts.MaxTokens)) * 100
		budgetUsedPercent = &pct
	}

	meta.Statistics = Statistics{
		TotalFiles:        opts.RenderData.TotalFiles,
		TotalTokens:       opts.RenderData.TotalTokens,
		TotalBytes:        totalBytes,
		BudgetUsedPercent: budgetUsedPercent,
		MaxTokens:         opts.MaxTokens,
		FilesByTier:       filesByTier,
		RedactionsTotal:   opts.RenderData.TotalRedactions,
		RedactionsByType:  redactionsByType,
		CompressedFiles:   compressedCount,
		GenerationTimeMs:  opts.GenerationTimeMs,
	}

	return meta
}

// WriteMetadata marshals the metadata to pretty-printed JSON and writes it
// atomically to the sidecar path (outputPath + ".meta.json"). The write
// uses a temporary file, sync, close, and rename pattern identical to the
// main output writer.
func WriteMetadata(meta *OutputMetadata, outputPath string) (retErr error) {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	// Append a trailing newline for POSIX compliance.
	data = append(data, '\n')

	sidecarPath := MetadataSidecarPath(outputPath)
	dir := filepath.Dir(sidecarPath)

	tmpFile, err := os.CreateTemp(dir, ".harvx-meta-*.tmp")
	if err != nil {
		return fmt.Errorf("writing metadata: creating temp file in %q: %w", dir, err)
	}
	tmpPath := tmpFile.Name()

	// Clean up the temp file on any error.
	defer func() {
		if retErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("writing metadata: syncing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("writing metadata: closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, sidecarPath); err != nil {
		return fmt.Errorf("writing metadata: renaming %q to %q: %w", tmpPath, sidecarPath, err)
	}

	slog.Debug("wrote metadata sidecar",
		"path", sidecarPath,
		"bytes", len(data),
	)

	return nil
}

// MetadataSidecarPath returns the sidecar file path for a given output path.
// The sidecar is always the output path with ".meta.json" appended.
func MetadataSidecarPath(outputPath string) string {
	return outputPath + ".meta.json"
}
