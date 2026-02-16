// Package pipeline defines the central data types shared across all pipeline
// stages in Harvx. These types serve as the data backbone: discovery, filtering,
// relevance sorting, content loading, tokenization, and rendering all operate
// on the same DTOs defined here.
//
// This package has zero external dependencies -- only stdlib types.
// It contains only data types and lightweight validation helpers; no business logic.
package pipeline

// ExitCode represents the process exit code returned by the harvx CLI.
type ExitCode int

const (
	// ExitSuccess indicates the pipeline completed successfully.
	ExitSuccess ExitCode = 0

	// ExitError indicates a fatal error occurred, or --fail-on-redaction was
	// triggered by detected secrets.
	ExitError ExitCode = 1

	// ExitPartial indicates partial success: some files failed processing but
	// output was still generated for the rest.
	ExitPartial ExitCode = 2
)

// OutputFormat specifies the format of the rendered context document.
type OutputFormat string

const (
	// FormatMarkdown renders the context document as Markdown with fenced code blocks.
	FormatMarkdown OutputFormat = "markdown"

	// FormatXML renders the context document as XML, optimized for Claude's
	// XML-native parsing capabilities.
	FormatXML OutputFormat = "xml"
)

// LLMTarget identifies the target LLM platform, allowing format and token
// defaults to be tuned per model family.
type LLMTarget string

const (
	// TargetClaude targets Anthropic Claude models. Defaults to XML output
	// format and cl100k_base tokenizer.
	TargetClaude LLMTarget = "claude"

	// TargetChatGPT targets OpenAI ChatGPT/GPT-4 models. Defaults to Markdown
	// output format.
	TargetChatGPT LLMTarget = "chatgpt"

	// TargetGeneric is a generic target with no model-specific optimizations.
	// Uses Markdown output format and cl100k_base tokenizer.
	TargetGeneric LLMTarget = "generic"
)

// DefaultTier is the relevance tier assigned to files that do not match any
// explicit tier pattern. Per the PRD (Section 5.3), unmatched files default to
// tier 2 (source code) to avoid excluding unexpected but important files.
const DefaultTier = 2

// FileDescriptor is the central DTO passed between all pipeline stages. Each
// stage enriches or mutates the descriptor as the file flows through the
// pipeline:
//
//   - Discovery: sets Path, AbsPath, Size, IsSymlink, IsBinary
//   - Relevance: sets Tier
//   - Content loading: sets Content, ContentHash, Language
//   - Security: updates Content (redacted), sets Redactions count
//   - Compression: updates Content (compressed), sets IsCompressed
//   - Tokenization: sets TokenCount
//
// The Content field stores processed content only. The original file content is
// never retained in memory -- files are processed one at a time to keep memory
// usage bounded.
type FileDescriptor struct {
	// Path is the file path relative to the repository root. Used for display,
	// tier matching, and deterministic output ordering.
	Path string `json:"path"`

	// AbsPath is the absolute filesystem path. Used for reading file content.
	AbsPath string `json:"abs_path"`

	// Size is the file size in bytes as reported by the filesystem.
	Size int64 `json:"size"`

	// Tier is the relevance tier (0-5). Lower tiers are higher priority and
	// included first when enforcing token budgets. Defaults to DefaultTier (2)
	// for unmatched files.
	Tier int `json:"tier"`

	// TokenCount is the number of tokens in the processed content, counted
	// after redaction and optional compression.
	TokenCount int `json:"token_count"`

	// ContentHash is the XXH3 hash of the processed content, used for change
	// detection and deterministic output verification. The hash is computed
	// externally; this field only stores the result.
	ContentHash uint64 `json:"content_hash"`

	// Content is the processed file content after redaction and optional
	// tree-sitter compression. The original content is never stored.
	Content string `json:"content"`

	// IsCompressed indicates whether tree-sitter compression was applied to
	// this file's content.
	IsCompressed bool `json:"is_compressed"`

	// Redactions is the number of secrets that were redacted from this file's
	// content during the security scanning stage.
	Redactions int `json:"redactions"`

	// Language is the detected programming language, used by the compression
	// stage to select the appropriate tree-sitter grammar.
	Language string `json:"language"`

	// IsSymlink indicates whether the file is a symbolic link. Symlinks may be
	// followed or skipped depending on configuration.
	IsSymlink bool `json:"is_symlink"`

	// IsBinary indicates whether binary content was detected. Binary files are
	// typically skipped during content loading.
	IsBinary bool `json:"is_binary"`

	// Error tracks per-file processing failures. When set, the file may still
	// appear in output with an error annotation rather than content. This field
	// does not serialize to JSON since the error interface cannot be marshaled
	// cleanly.
	Error error `json:"-"`
}

// IsValid reports whether the FileDescriptor has the minimum required fields
// for a valid pipeline entry. A descriptor is valid if it has a non-empty
// relative path.
func (fd *FileDescriptor) IsValid() bool {
	return fd.Path != ""
}

// DiscoveryResult holds the aggregate output of the file discovery phase,
// including the discovered files and summary statistics about what was found
// and what was skipped.
type DiscoveryResult struct {
	// Files is the slice of discovered file descriptors that passed all
	// filtering criteria (ignore patterns, binary detection, size limits).
	Files []FileDescriptor `json:"files"`

	// TotalFound is the total number of files encountered during directory
	// traversal, before any filtering was applied.
	TotalFound int `json:"total_found"`

	// TotalSkipped is the total number of files that were skipped due to
	// ignore patterns, binary detection, size limits, or other filters.
	TotalSkipped int `json:"total_skipped"`

	// SkipReasons maps each skip reason (e.g., "binary", "gitignore",
	// "size_limit") to the count of files skipped for that reason.
	SkipReasons map[string]int `json:"skip_reasons"`
}
