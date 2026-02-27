// Package pipeline defines the stage service interfaces for the Harvx
// processing pipeline. Each interface represents a single pipeline stage
// and is designed to be small, composable, and independently testable.
//
// These interfaces are consumed by the Pipeline struct via functional options
// (WithDiscovery, WithRelevance, etc.) and enable workflow commands (brief,
// slice, review-slice) and the MCP server to share the same processing engine.
//
// This file has zero external dependencies -- only stdlib types and types
// defined in this package.
package pipeline

import (
	"context"
	"io"
)

// DiscoveryService discovers files in a target directory, applying ignore
// patterns, binary detection, size limits, and glob filters. Returns files
// ready for subsequent pipeline stages.
type DiscoveryService interface {
	Discover(ctx context.Context, opts DiscoveryOptions) (*DiscoveryResult, error)
}

// DiscoveryOptions encapsulates all parameters for the discovery stage.
type DiscoveryOptions struct {
	// RootDir is the target directory to scan.
	RootDir string

	// GitTrackedOnly restricts results to files tracked by git.
	GitTrackedOnly bool

	// SkipLargeFiles is the size threshold in bytes (0 = no limit).
	SkipLargeFiles int64

	// Includes are doublestar glob patterns; files matching any are included.
	Includes []string

	// Excludes are doublestar glob patterns; matched files are excluded.
	Excludes []string

	// Extensions are file extensions to filter by (no leading dot).
	Extensions []string
}

// RelevanceService assigns relevance tiers to files and sorts them by
// priority. Lower tier numbers indicate higher relevance.
type RelevanceService interface {
	Classify(files []*FileDescriptor) []*FileDescriptor
}

// TokenizerService counts tokens in file content using a specific encoding
// (e.g., cl100k_base, o200k_base).
type TokenizerService interface {
	// Count returns the number of tokens in the given text.
	Count(text string) int

	// Name returns the tokenizer encoding name (e.g., "cl100k_base").
	Name() string
}

// BudgetService enforces token budgets across a set of files, deciding which
// files to include, truncate, or skip to stay within the limit.
type BudgetService interface {
	// Enforce applies the token budget, potentially skipping or truncating
	// files. MaxTokens of 0 means unlimited (no enforcement).
	Enforce(files []*FileDescriptor, maxTokens int) (*BudgetResult, error)
}

// BudgetResult holds the outcome of budget enforcement.
type BudgetResult struct {
	// Included contains files that fit within the budget.
	Included []*FileDescriptor

	// Skipped contains files that were dropped to stay within budget.
	Skipped []*FileDescriptor

	// TotalTokens is the sum of tokens across all included files.
	TotalTokens int

	// BudgetUsed is the percentage of the budget consumed (0-100).
	// A value of -1 indicates no budget was set.
	BudgetUsed float64
}

// RedactionService scans file content for secrets (API keys, tokens,
// connection strings) and replaces them with placeholders.
type RedactionService interface {
	// Redact scans content for secrets and returns sanitized content.
	// The returned int is the number of redactions applied.
	Redact(ctx context.Context, content string, filePath string) (string, int, error)
}

// CompressionService compresses file content by extracting structural
// signatures using tree-sitter grammars. This reduces token usage while
// preserving the essential structure of source files.
type CompressionService interface {
	// Compress processes files, replacing content with compressed structural
	// signatures where applicable. Modifies files in place.
	Compress(ctx context.Context, files []*FileDescriptor) error
}

// RenderService renders processed files into the final output document
// (Markdown or XML format).
type RenderService interface {
	// Render writes the output document to the given writer.
	Render(ctx context.Context, w io.Writer, files []FileDescriptor, opts RenderOptions) error
}

// RenderOptions holds rendering configuration for the output stage.
type RenderOptions struct {
	// Format is the output format ("markdown" or "xml").
	Format string

	// ProjectName is the project name for the output header.
	ProjectName string

	// ProfileName is the profile name for the output header.
	ProfileName string

	// TokenizerName is the tokenizer encoding name for the output header.
	TokenizerName string

	// ShowLineNumbers enables line number prefixes in code blocks.
	ShowLineNumbers bool

	// DiffSummary holds change summary data for diff mode rendering.
	// Nil when not in diff mode.
	DiffSummary *DiffSummaryEntry
}

// DiffSummaryEntry holds diff-mode change data for rendering. It captures
// which files were added, modified, or deleted between two snapshots.
type DiffSummaryEntry struct {
	// AddedFiles lists relative paths of files added since the baseline.
	AddedFiles []string

	// ModifiedFiles lists relative paths of files modified since the baseline.
	ModifiedFiles []string

	// DeletedFiles lists relative paths of files deleted since the baseline.
	DeletedFiles []string

	// Unchanged is the count of files that did not change.
	Unchanged int
}
