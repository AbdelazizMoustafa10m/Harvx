package compression

import "context"

// Compile-time interface compliance check.
var _ LanguageCompressor = (*FallbackCompressor)(nil)

// FallbackCompressor returns file content unchanged for unsupported languages.
// It ensures the compression pipeline never drops files silently. The output
// is intentionally NOT marked as compressed so callers can distinguish between
// genuinely compressed files and passthrough content.
//
// FallbackCompressor is stateless and safe for concurrent use.
type FallbackCompressor struct{}

// NewFallbackCompressor creates a fallback compressor.
func NewFallbackCompressor() *FallbackCompressor {
	return &FallbackCompressor{}
}

// Compress returns the full file content as a single signature. Empty input
// yields an empty CompressedOutput with no signatures. This method never
// returns an error except when the context is already cancelled.
func (c *FallbackCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "unknown",
			OriginalSize: 0,
			OutputSize:   0,
			NodeCount:    0,
		}, nil
	}

	size := len(source)
	return &CompressedOutput{
		Signatures: []Signature{{
			Kind:      KindDocComment,
			Name:      "",
			Source:    string(source),
			StartLine: 1,
			EndLine:   countLines(string(source)),
		}},
		Language:     "unknown",
		OriginalSize: size,
		OutputSize:   size,
		NodeCount:    1,
	}, nil
}

// Language returns "fallback".
func (c *FallbackCompressor) Language() string {
	return "fallback"
}

// SupportedNodeTypes returns the node types this compressor produces.
func (c *FallbackCompressor) SupportedNodeTypes() []string {
	return []string{"raw_content"}
}

// IsFallback reports whether the compressed output was produced by the fallback
// compressor (i.e., the file was not actually compressed).
func IsFallback(co *CompressedOutput) bool {
	return co != nil && co.Language == "unknown"
}