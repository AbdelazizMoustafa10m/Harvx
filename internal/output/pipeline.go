package output

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/harvx/harvx/internal/pipeline"
)

// OutputConfig aggregates all output-related settings needed by the output
// pipeline. It is populated from CLI flags, profile configuration, and pipeline
// defaults before being passed to RenderOutput.
type OutputConfig struct {
	// Format is the output format: "markdown" or "xml".
	Format string

	// Target is the LLM target: "claude", "chatgpt", or "generic".
	Target string

	// OutputPath is the explicit output file path from the --output/-o CLI flag.
	OutputPath string

	// ProfileOutput is the output path from the active TOML profile config.
	ProfileOutput string

	// UseStdout writes to stdout instead of a file when true.
	UseStdout bool

	// SplitTokens is the maximum tokens per part. 0 means no splitting.
	SplitTokens int

	// ShowLineNumbers enables line number prefixes in code blocks.
	ShowLineNumbers bool

	// OutputMetadata enables .meta.json sidecar generation.
	OutputMetadata bool

	// TreeMaxDepth controls tree rendering depth. 0 means unlimited.
	TreeMaxDepth int

	// ShowTreeMetadata shows size and token count annotations in the tree.
	ShowTreeMetadata bool

	// ProjectName is the project name for the output header.
	ProjectName string

	// ProfileName is the config profile name for the output header.
	ProfileName string

	// TokenizerName is the tokenizer encoding name for the output header.
	TokenizerName string

	// Timestamp is the generation timestamp. Use a fixed value for
	// deterministic output. When zero, time.Now() is used.
	Timestamp time.Time

	// MaxTokens is the token budget, used in metadata budget calculation.
	MaxTokens int

	// GenerationTimeMs is the pipeline generation time in milliseconds.
	GenerationTimeMs int64

	// DiffSummary holds change summary data for diff mode rendering.
	// Nil means no diff data is available.
	DiffSummary *DiffSummaryData

	// Writer is an optional custom OutputWriter. When nil, a default
	// OutputWriter writing to os.Stdout/os.Stderr is created.
	Writer *OutputWriter
}

// RenderOutput orchestrates the full output rendering flow. It converts
// pipeline FileDescriptors into rendered output, optionally splitting across
// multiple files and generating metadata sidecars.
//
// The pipeline steps are:
//  1. Convert []pipeline.FileDescriptor to internal types
//  2. Build directory tree via BuildTree + RenderTree
//  3. Compute content hash via ContentHasher
//  4. Assemble RenderData with tier counts, top files, etc.
//  5. Write output via OutputWriter (file or stdout)
//  6. Optionally split into multiple parts
//  7. Optionally generate metadata sidecar
func RenderOutput(ctx context.Context, cfg OutputConfig, files []pipeline.FileDescriptor) (*OutputResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use default timestamp if not set.
	ts := cfg.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	// Step 1: Convert FileDescriptors to internal types.
	renderEntries := toFileRenderEntries(files)
	treeEntries := toFileEntries(files)
	hashEntries := toFileHashEntries(files)

	slog.Debug("output pipeline starting",
		"files", len(files),
		"format", cfg.Format,
		"split_tokens", cfg.SplitTokens,
	)

	// Step 2: Build and render the directory tree.
	tree := BuildTree(treeEntries)
	treeString := RenderTree(tree, TreeRenderOpts{
		MaxDepth:   cfg.TreeMaxDepth,
		ShowSize:   cfg.ShowTreeMetadata,
		ShowTokens: cfg.ShowTreeMetadata,
	})

	// Step 3: Compute content hash.
	hasher := NewContentHasher()
	contentHash, err := hasher.ComputeContentHash(hashEntries)
	if err != nil {
		return nil, fmt.Errorf("computing content hash: %w", err)
	}
	contentHashHex := FormatHash(contentHash)

	// Step 4: Assemble RenderData.
	tierCounts := computeTierCounts(renderEntries)
	totalTokens := computeTotalTokens(renderEntries)
	totalRedactions := computeTotalRedactions(renderEntries)
	topFiles := computeTopFiles(renderEntries, 5)

	data := &RenderData{
		ProjectName:      cfg.ProjectName,
		Timestamp:        ts,
		ContentHash:      contentHashHex,
		ProfileName:      cfg.ProfileName,
		TokenizerName:    cfg.TokenizerName,
		TotalTokens:      totalTokens,
		TotalFiles:       len(renderEntries),
		Files:            renderEntries,
		TreeString:       treeString,
		ShowLineNumbers:  cfg.ShowLineNumbers,
		TierCounts:       tierCounts,
		TopFilesByTokens: topFiles,
		RedactionSummary: map[string]int{},
		TotalRedactions:  totalRedactions,
		DiffSummary:      cfg.DiffSummary,
	}

	// Step 5: Get or create the OutputWriter.
	writer := cfg.Writer
	if writer == nil {
		writer = NewOutputWriter()
	}

	// Step 6: Write output, optionally splitting.
	if cfg.SplitTokens > 0 {
		return renderSplit(ctx, writer, data, cfg)
	}

	return renderSingle(ctx, writer, data, cfg)
}

// renderSingle writes the output as a single file or to stdout.
func renderSingle(ctx context.Context, writer *OutputWriter, data *RenderData, cfg OutputConfig) (*OutputResult, error) {
	opts := OutputOpts{
		OutputPath:       cfg.OutputPath,
		ProfileOutput:    cfg.ProfileOutput,
		Format:           cfg.Format,
		UseStdout:        cfg.UseStdout,
		OutputMetadata:   cfg.OutputMetadata,
		Target:           cfg.Target,
		MaxTokens:        cfg.MaxTokens,
		GenerationTimeMs: cfg.GenerationTimeMs,
	}

	result, err := writer.Write(ctx, data, opts)
	if err != nil {
		return nil, fmt.Errorf("rendering output: %w", err)
	}

	slog.Info("output rendered",
		"path", result.Path,
		"format", cfg.Format,
		"files", data.TotalFiles,
		"tokens", data.TotalTokens,
		"bytes", result.BytesWritten,
		"hash", result.HashHex,
	)

	return result, nil
}

// renderSplit writes the output as multiple split parts.
func renderSplit(ctx context.Context, writer *OutputWriter, data *RenderData, cfg OutputConfig) (*OutputResult, error) {
	splitOpts := SplitOutputOpts{
		OutputOpts: OutputOpts{
			OutputPath:       cfg.OutputPath,
			ProfileOutput:    cfg.ProfileOutput,
			Format:           cfg.Format,
			UseStdout:        cfg.UseStdout,
			OutputMetadata:   cfg.OutputMetadata,
			Target:           cfg.Target,
			MaxTokens:        cfg.MaxTokens,
			GenerationTimeMs: cfg.GenerationTimeMs,
		},
		SplitTokens: cfg.SplitTokens,
	}

	parts, err := writer.WriteSplit(ctx, data, splitOpts)
	if err != nil {
		return nil, fmt.Errorf("rendering split output: %w", err)
	}

	// Aggregate results into a single OutputResult.
	result := &OutputResult{
		TotalTokens: data.TotalTokens,
		Parts:       parts,
	}

	// Use the first part's path and hash as the primary result.
	if len(parts) > 0 {
		result.Path = parts[0].Path
		result.Hash = parts[0].Hash
		result.HashHex = FormatHash(parts[0].Hash)
	}

	// Sum bytes written across all parts.
	// Note: Parts don't track bytes individually in PartResult,
	// so we report the token count instead.
	for _, p := range parts {
		slog.Info("split part written",
			"part", p.PartNumber,
			"path", p.Path,
			"files", p.FileCount,
			"tokens", p.TokenCount,
		)
	}

	return result, nil
}

// toFileRenderEntries converts pipeline FileDescriptors to FileRenderEntries.
func toFileRenderEntries(files []pipeline.FileDescriptor) []FileRenderEntry {
	entries := make([]FileRenderEntry, 0, len(files))
	for _, fd := range files {
		entry := FileRenderEntry{
			Path:         fd.Path,
			Size:         fd.Size,
			TokenCount:   fd.TokenCount,
			Tier:         fd.Tier,
			TierLabel:    tierLabel(fd.Tier),
			Language:     fd.Language,
			Content:      fd.Content,
			IsCompressed: fd.IsCompressed,
			Redactions:   fd.Redactions,
		}
		if fd.Error != nil {
			entry.Error = fd.Error.Error()
		}
		// If no language is set, infer from file extension.
		if entry.Language == "" {
			entry.Language = languageFromExt(fd.Path)
		}
		entries = append(entries, entry)
	}
	return entries
}

// toFileEntries converts pipeline FileDescriptors to tree FileEntries.
func toFileEntries(files []pipeline.FileDescriptor) []FileEntry {
	entries := make([]FileEntry, 0, len(files))
	for _, fd := range files {
		entries = append(entries, FileEntry{
			Path:       fd.Path,
			Size:       fd.Size,
			TokenCount: fd.TokenCount,
			Tier:       fd.Tier,
		})
	}
	return entries
}

// toFileHashEntries converts pipeline FileDescriptors to hash FileHashEntries.
func toFileHashEntries(files []pipeline.FileDescriptor) []FileHashEntry {
	entries := make([]FileHashEntry, 0, len(files))
	for _, fd := range files {
		entries = append(entries, FileHashEntry{
			Path:    fd.Path,
			Content: fd.Content,
		})
	}
	return entries
}

// computeTierCounts returns a map of tier number to file count.
func computeTierCounts(files []FileRenderEntry) map[int]int {
	counts := make(map[int]int)
	for _, f := range files {
		counts[f.Tier]++
	}
	return counts
}

// computeTotalTokens returns the sum of all file token counts.
func computeTotalTokens(files []FileRenderEntry) int {
	total := 0
	for _, f := range files {
		total += f.TokenCount
	}
	return total
}

// computeTotalRedactions returns the sum of all file redaction counts.
func computeTotalRedactions(files []FileRenderEntry) int {
	total := 0
	for _, f := range files {
		total += f.Redactions
	}
	return total
}

// computeTopFiles returns the top N files sorted by token count descending.
// If there are fewer than n files, all files are returned.
func computeTopFiles(files []FileRenderEntry, n int) []FileRenderEntry {
	if len(files) == 0 {
		return nil
	}

	// Copy to avoid mutating the caller's slice.
	sorted := make([]FileRenderEntry, len(files))
	copy(sorted, files)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TokenCount > sorted[j].TokenCount
	})

	if n > len(sorted) {
		n = len(sorted)
	}

	return sorted[:n]
}
