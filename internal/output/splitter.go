package output

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

// DefaultOverheadPerFile is the estimated token overhead per file for
// headers, fences, and metadata lines in the rendered output.
const DefaultOverheadPerFile = 200

// DefaultPartHeaderOverhead is the estimated token overhead for each part's
// header and summary sections (part number, hashes, references).
const DefaultPartHeaderOverhead = 300

// SplitOpts configures the output splitter.
type SplitOpts struct {
	// TokensPerPart is the maximum token budget per part. Must be > 0.
	TokensPerPart int

	// Format is the output format: "markdown" or "xml".
	Format string

	// OverheadPerFile is the estimated token overhead per file for headers
	// and fences. When 0, DefaultOverheadPerFile is used.
	OverheadPerFile int
}

// PartData holds the render data for a single split part.
type PartData struct {
	// PartNumber is the 1-based part index.
	PartNumber int

	// TotalParts is the total number of parts in the split.
	TotalParts int

	// RenderData is the per-part render data containing the subset of files
	// assigned to this part, with modified header information.
	RenderData *RenderData

	// GlobalHash is the content hash of ALL files across ALL parts.
	GlobalHash string
}

// PartResult holds metadata about a rendered part file.
type PartResult struct {
	// PartNumber is the 1-based part index.
	PartNumber int

	// Path is the file path of the rendered part.
	Path string

	// TokenCount is the total estimated token count for this part.
	TokenCount int

	// FileCount is the number of files in this part.
	FileCount int

	// Hash is the XXH3 64-bit content hash of the rendered part.
	Hash uint64
}

// Splitter divides a full RenderData into multiple parts, respecting file
// atomicity and tier boundaries where possible. No individual file is ever
// split across parts.
type Splitter struct {
	opts SplitOpts
}

// NewSplitter creates a new Splitter with the given options.
func NewSplitter(opts SplitOpts) *Splitter {
	if opts.OverheadPerFile <= 0 {
		opts.OverheadPerFile = DefaultOverheadPerFile
	}
	return &Splitter{opts: opts}
}

// Split divides the render data into parts based on the token budget. If all
// files fit in a single part, it returns a single PartData with the original
// data unchanged (no part numbering applied). The returned slice is never nil
// and always contains at least one element (even for empty file lists).
//
// Algorithm:
//  1. Calculate overhead for headers/summaries per part.
//  2. Iterate files in their existing order (assumed tier/path sorted).
//  3. When adding the next file would exceed tokensPerPart, start a new part.
//  4. Within a tier, prefer keeping files from the same top-level directory
//     together (soft preference).
//  5. A single file exceeding the budget gets its own part with a warning.
func (s *Splitter) Split(data *RenderData) ([]PartData, error) {
	if data == nil {
		return nil, fmt.Errorf("splitting output: render data is nil")
	}

	if s.opts.TokensPerPart <= 0 {
		return nil, fmt.Errorf("splitting output: tokens per part must be positive, got %d", s.opts.TokensPerPart)
	}

	// Compute the global hash across all files.
	globalHash := data.ContentHash

	// If all files fit in one part, return unchanged.
	totalTokensWithOverhead := s.estimateTotalTokens(data.Files)
	if totalTokensWithOverhead <= s.opts.TokensPerPart {
		return []PartData{
			{
				PartNumber: 1,
				TotalParts: 1,
				RenderData: data,
				GlobalHash: globalHash,
			},
		}, nil
	}

	// Split files into part buckets.
	buckets := s.assignFilesToParts(data.Files)
	totalParts := len(buckets)

	if totalParts == 0 {
		// No files: return one empty part.
		return []PartData{
			{
				PartNumber: 1,
				TotalParts: 1,
				RenderData: s.buildPartRenderData(data, nil, 1, 1, globalHash),
				GlobalHash: globalHash,
			},
		}, nil
	}

	parts := make([]PartData, totalParts)
	for i, bucket := range buckets {
		partNum := i + 1
		parts[i] = PartData{
			PartNumber: partNum,
			TotalParts: totalParts,
			RenderData: s.buildPartRenderData(data, bucket, partNum, totalParts, globalHash),
			GlobalHash: globalHash,
		}
	}

	return parts, nil
}

// assignFilesToParts implements the greedy bin-packing algorithm. Files are
// iterated in order and accumulated into parts. When the next file would
// exceed the per-part budget, a new part is started. Within a tier, files
// from the same top-level directory are kept together when possible.
func (s *Splitter) assignFilesToParts(files []FileRenderEntry) [][]FileRenderEntry {
	if len(files) == 0 {
		return nil
	}

	overhead := s.opts.OverheadPerFile
	budget := s.opts.TokensPerPart

	// Reserve space for part header overhead (Part 1 has more overhead for
	// tree and summary).
	part1HeaderOverhead := DefaultPartHeaderOverhead * 2 // tree + summary + header
	partNHeaderOverhead := DefaultPartHeaderOverhead     // minimal header only

	var buckets [][]FileRenderEntry
	var currentBucket []FileRenderEntry
	currentTokens := 0

	for i, file := range files {
		fileTokens := file.TokenCount + overhead

		// Determine the header overhead for the current part.
		// If no buckets have been flushed yet, we're building Part 1 which
		// includes the tree and summary and thus has higher overhead.
		headerOverhead := partNHeaderOverhead
		if len(buckets) == 0 {
			headerOverhead = part1HeaderOverhead
		}

		effectiveBudget := budget - headerOverhead

		// Handle oversized single file: exceeds any single-part budget.
		if fileTokens > effectiveBudget {
			// Flush current bucket if non-empty.
			if len(currentBucket) > 0 {
				buckets = append(buckets, currentBucket)
				currentBucket = nil
				currentTokens = 0
			}

			slog.Warn("single file exceeds split budget",
				"path", file.Path,
				"file_tokens", file.TokenCount,
				"budget", budget,
			)

			// Give this file its own part.
			buckets = append(buckets, []FileRenderEntry{file})
			continue
		}

		// Check if adding this file would exceed budget.
		if currentTokens+fileTokens > effectiveBudget && len(currentBucket) > 0 {
			// Try directory coherence: keep files from the same tier and
			// top-level directory together if the overflow is within tolerance.
			if s.shouldKeepTogether(files, i, currentTokens, fileTokens, effectiveBudget) {
				// Allow the overflow for coherence -- add without flushing.
				currentBucket = append(currentBucket, file)
				currentTokens += fileTokens
				continue
			}

			// Flush and start a new part.
			buckets = append(buckets, currentBucket)
			currentBucket = nil
			currentTokens = 0
		}

		currentBucket = append(currentBucket, file)
		currentTokens += fileTokens
	}

	// Flush remaining.
	if len(currentBucket) > 0 {
		buckets = append(buckets, currentBucket)
	}

	return buckets
}

// shouldKeepTogether returns true if the current file should be kept in the
// same part as the previous files for directory coherence. This is a soft
// preference: it only applies when the files are in the same tier and
// share the same top-level directory, and the budget overflow is within
// a 15% tolerance.
func (s *Splitter) shouldKeepTogether(files []FileRenderEntry, idx, currentTokens, fileTokens, effectiveBudget int) bool {
	if idx == 0 {
		return false
	}

	prev := files[idx-1]
	curr := files[idx]

	// Must be same tier.
	if prev.Tier != curr.Tier {
		return false
	}

	// Must share top-level directory.
	if topLevelDir(prev.Path) != topLevelDir(curr.Path) {
		return false
	}

	// Allow up to 15% overflow for coherence.
	tolerance := effectiveBudget + effectiveBudget*15/100
	return currentTokens+fileTokens <= tolerance
}

// topLevelDir returns the first path component of a file path. For files at
// the root, it returns an empty string.
func topLevelDir(filePath string) string {
	parts := strings.SplitN(filepath.ToSlash(filePath), "/", 2)
	if len(parts) < 2 {
		return "" // file is at root level
	}
	return parts[0]
}

// estimateTotalTokens calculates the total estimated tokens including
// per-file overhead for all files.
func (s *Splitter) estimateTotalTokens(files []FileRenderEntry) int {
	total := DefaultPartHeaderOverhead * 2 // header + tree + summary overhead
	for _, f := range files {
		total += f.TokenCount + s.opts.OverheadPerFile
	}
	return total
}

// buildPartRenderData creates a RenderData for a specific part. Part 1
// includes the directory tree and file summary. Parts 2+ include only a
// minimal header referencing Part 1.
func (s *Splitter) buildPartRenderData(original *RenderData, files []FileRenderEntry, partNum, totalParts int, globalHash string) *RenderData {
	// Calculate part-specific totals.
	partTokens := 0
	partTierCounts := make(map[int]int)
	for _, f := range files {
		partTokens += f.TokenCount
		partTierCounts[f.Tier]++
	}

	rd := &RenderData{
		ProjectName:     original.ProjectName,
		Timestamp:       original.Timestamp,
		ContentHash:     globalHash,
		ProfileName:     original.ProfileName,
		TokenizerName:   original.TokenizerName,
		TotalTokens:     partTokens,
		TotalFiles:      len(files),
		Files:           files,
		ShowLineNumbers: original.ShowLineNumbers,
		TierCounts:      partTierCounts,
		DiffSummary:     original.DiffSummary,
	}

	if partNum == 1 {
		// Part 1 gets the full tree and summary.
		rd.TreeString = original.TreeString
		rd.TopFilesByTokens = original.TopFilesByTokens
		rd.RedactionSummary = original.RedactionSummary
		rd.TotalRedactions = original.TotalRedactions
	} else {
		// Parts 2+ get a minimal tree placeholder.
		rd.TreeString = fmt.Sprintf("(See Part 1 for full directory tree and file summary)")
		rd.TopFilesByTokens = nil
		rd.RedactionSummary = nil
		rd.TotalRedactions = 0
	}

	return rd
}

// PartPath inserts a part number into a file path. For example,
// "harvx-output.md" becomes "harvx-output.part-001.md". If totalParts is 1,
// the path is returned unchanged (no part suffix needed).
func PartPath(basePath string, partNum, totalParts int) string {
	if totalParts <= 1 {
		return basePath
	}

	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)

	return fmt.Sprintf("%s.part-%03d%s", base, partNum, ext)
}

// SplitOutputOpts extends OutputOpts with split-specific configuration.
type SplitOutputOpts struct {
	OutputOpts

	// SplitTokens is the max tokens per part. 0 means no splitting.
	SplitTokens int
}

// WriteSplit renders the context document as multiple split parts. It returns
// a PartResult for each part written. When SplitTokens is 0 or all files fit
// in one part, the output is written as a single file (no part suffix).
func (ow *OutputWriter) WriteSplit(ctx context.Context, data *RenderData, opts SplitOutputOpts) ([]PartResult, error) {
	if data == nil {
		return nil, fmt.Errorf("writing split output: render data is nil")
	}

	if opts.SplitTokens <= 0 {
		return nil, fmt.Errorf("writing split output: split tokens must be positive, got %d", opts.SplitTokens)
	}

	if opts.Format != FormatMarkdown && opts.Format != FormatXML {
		return nil, fmt.Errorf("writing split output: unsupported format %q", opts.Format)
	}

	splitter := NewSplitter(SplitOpts{
		TokensPerPart: opts.SplitTokens,
		Format:        opts.Format,
	})

	parts, err := splitter.Split(data)
	if err != nil {
		return nil, fmt.Errorf("writing split output: %w", err)
	}

	// Single part, no split suffix.
	if len(parts) == 1 && parts[0].TotalParts == 1 {
		result, err := ow.Write(ctx, parts[0].RenderData, opts.OutputOpts)
		if err != nil {
			return nil, fmt.Errorf("writing split output (single part): %w", err)
		}
		return []PartResult{
			{
				PartNumber: 1,
				Path:       result.Path,
				TokenCount: result.TotalTokens,
				FileCount:  parts[0].RenderData.TotalFiles,
				Hash:       result.Hash,
			},
		}, nil
	}

	// Multiple parts.
	basePath := ResolveOutputPath(opts.OutputPath, opts.ProfileOutput, opts.Format)
	totalParts := parts[0].TotalParts

	results := make([]PartResult, 0, len(parts))

	for _, part := range parts {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		partPath := PartPath(basePath, part.PartNumber, totalParts)

		// Update the part's render data with a part-specific project name
		// to indicate the part number in the header.
		partData := part.RenderData
		partData.ProjectName = fmt.Sprintf("%s (Part %d of %d)",
			data.ProjectName, part.PartNumber, totalParts)
		partData.Timestamp = time.Now()

		partOpts := OutputOpts{
			OutputPath: partPath,
			Format:     opts.Format,
			UseStdout:  opts.UseStdout,
		}

		result, err := ow.Write(ctx, partData, partOpts)
		if err != nil {
			return results, fmt.Errorf("writing part %d of %d: %w", part.PartNumber, totalParts, err)
		}

		results = append(results, PartResult{
			PartNumber: part.PartNumber,
			Path:       result.Path,
			TokenCount: result.TotalTokens,
			FileCount:  partData.TotalFiles,
			Hash:       result.Hash,
		})

		slog.Info("wrote split part",
			"part", part.PartNumber,
			"total_parts", totalParts,
			"path", result.Path,
			"files", partData.TotalFiles,
			"tokens", result.TotalTokens,
		)
	}

	return results, nil
}
