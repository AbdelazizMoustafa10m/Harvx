package compression

import "context"

// LanguageCompressor compresses source code for a specific language by parsing
// it and extracting structural signatures. Implementations must be stateless
// and safe for concurrent use.
type LanguageCompressor interface {
	// Compress parses the source and extracts structural signatures.
	// The returned output contains verbatim source text at AST node boundaries.
	// It must never summarize or rewrite code.
	Compress(ctx context.Context, source []byte) (*CompressedOutput, error)

	// Language returns the language identifier (e.g., "typescript", "go").
	Language() string

	// SupportedNodeTypes returns the AST node types this compressor extracts.
	SupportedNodeTypes() []string
}
