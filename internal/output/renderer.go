package output

import (
	"context"
	"io"
	"time"
)

// Renderer is the interface for output format renderers. Each output format
// (Markdown, XML) implements this interface to produce the final context document.
type Renderer interface {
	// Render writes the complete context document to w using the provided data.
	// The renderer must not buffer the entire output in memory; it should stream
	// directly to w for large codebases.
	Render(ctx context.Context, w io.Writer, data *RenderData) error
}

// RenderData holds all data needed to render a context document. It is assembled
// by the pipeline after discovery, relevance sorting, tokenization, redaction,
// and compression stages.
type RenderData struct {
	// ProjectName is the name of the project (typically the directory name).
	ProjectName string

	// Timestamp is the generation timestamp, formatted as RFC 3339 by the
	// renderer. The caller sets this to a fixed value for deterministic output.
	Timestamp time.Time

	// ContentHash is the hex-encoded hash of all processed content, used for
	// change detection.
	ContentHash string

	// ProfileName is the name of the config profile used for generation.
	ProfileName string

	// TokenizerName is the tokenizer encoding used (e.g., "cl100k_base").
	TokenizerName string

	// TotalTokens is the sum of all file token counts.
	TotalTokens int

	// TotalFiles is the total number of files included in the output.
	TotalFiles int

	// Files is the sorted list of files to render, with per-file metadata
	// and content.
	Files []FileRenderEntry

	// TreeString is the pre-rendered directory tree from T-051's RenderTree.
	TreeString string

	// ShowLineNumbers enables line number prefixes inside code blocks.
	ShowLineNumbers bool

	// TierCounts maps tier number (0-5) to the count of files in that tier.
	TierCounts map[int]int

	// TopFilesByTokens is a list of top N files by token count, for the summary.
	TopFilesByTokens []FileRenderEntry

	// RedactionSummary maps redaction type labels to their counts.
	RedactionSummary map[string]int

	// TotalRedactions is the total number of redactions across all files.
	TotalRedactions int

	// DiffSummary holds change summary data when diff mode is active.
	// Nil means no diff data is available.
	DiffSummary *DiffSummaryData
}

// FileRenderEntry holds per-file data needed for rendering.
type FileRenderEntry struct {
	// Path is the file's relative path.
	Path string

	// Size is the file size in bytes.
	Size int64

	// TokenCount is the token count after processing.
	TokenCount int

	// Tier is the relevance tier (0-5).
	Tier int

	// TierLabel is the human-readable tier name (e.g., "critical", "primary").
	TierLabel string

	// Language is the detected programming language for code fence identifiers.
	Language string

	// Content is the processed file content (after redaction/compression).
	Content string

	// IsCompressed indicates whether compression was applied.
	IsCompressed bool

	// Redactions is the number of secrets redacted from this file.
	Redactions int

	// Error is set when the file had a processing error. When non-empty, the
	// renderer displays the error message instead of file content.
	Error string
}

// DiffSummaryData holds change summary information for diff mode rendering.
type DiffSummaryData struct {
	// AddedFiles is the list of newly added file paths.
	AddedFiles []string

	// ModifiedFiles is the list of modified file paths.
	ModifiedFiles []string

	// DeletedFiles is the list of deleted file paths.
	DeletedFiles []string

	// Unchanged is the count of files that have not changed since the last run.
	Unchanged int
}
