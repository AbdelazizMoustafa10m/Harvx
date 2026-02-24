package compression

import "context"

// Compile-time interface compliance check.
var _ LanguageCompressor = (*TypeScriptCompressor)(nil)

// TypeScriptCompressor implements LanguageCompressor for TypeScript.
// It uses a line-by-line state machine parser to extract structural signatures
// from TypeScript source code. The compressor is stateless and safe for
// concurrent use.
type TypeScriptCompressor struct {
	parser *jsParser
}

// NewTypeScriptCompressor creates a TypeScript compressor.
func NewTypeScriptCompressor() *TypeScriptCompressor {
	return &TypeScriptCompressor{
		parser: newJSParser(jsParserConfig{
			extractInterfaces:      true,
			extractTypeAliases:     true,
			extractEnums:           true,
			extractTypeAnnotations: true,
			language:               "typescript",
		}),
	}
}

// Compress parses TypeScript source and extracts structural signatures.
// The returned output contains verbatim source text; it never summarizes
// or rewrites code.
func (c *TypeScriptCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	return c.parser.parse(ctx, source)
}

// Language returns "typescript".
func (c *TypeScriptCompressor) Language() string {
	return "typescript"
}

// SupportedNodeTypes returns the AST node types this compressor extracts.
func (c *TypeScriptCompressor) SupportedNodeTypes() []string {
	return []string{
		"function_declaration",
		"method_definition",
		"arrow_function",
		"class_declaration",
		"interface_declaration",
		"type_alias_declaration",
		"enum_declaration",
		"import_statement",
		"export_statement",
		"lexical_declaration",
	}
}
