package compression

import "context"

// Compile-time interface compliance check.
var _ LanguageCompressor = (*JavaScriptCompressor)(nil)

// JavaScriptCompressor implements LanguageCompressor for JavaScript.
// It uses a line-by-line state machine parser to extract structural signatures
// from JavaScript source code. The compressor is stateless and safe for
// concurrent use.
type JavaScriptCompressor struct {
	parser *jsParser
}

// NewJavaScriptCompressor creates a JavaScript compressor.
func NewJavaScriptCompressor() *JavaScriptCompressor {
	return &JavaScriptCompressor{
		parser: newJSParser(jsParserConfig{
			extractInterfaces:      false,
			extractTypeAliases:     false,
			extractEnums:           false,
			extractTypeAnnotations: false,
			language:               "javascript",
		}),
	}
}

// Compress parses JavaScript source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *JavaScriptCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	return c.parser.parse(ctx, source)
}

// Language returns "javascript".
func (c *JavaScriptCompressor) Language() string {
	return "javascript"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *JavaScriptCompressor) SupportedNodeTypes() []string {
	return []string{
		"function_declaration",
		"method_definition",
		"arrow_function",
		"class_declaration",
		"import_statement",
		"export_statement",
		"lexical_declaration",
	}
}
