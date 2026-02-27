package compression

import "strings"

// SignatureKind classifies the type of extracted code signature.
type SignatureKind int

const (
	KindFunction  SignatureKind = iota // Function or method signature
	KindClass                         // Class declaration
	KindStruct                        // Struct declaration
	KindInterface                     // Interface declaration
	KindType                          // Type alias or enum
	KindImport                        // Import/require statement
	KindExport                        // Export statement
	KindConstant                      // Top-level constant declaration
	KindDocComment                    // Doc comment (not inline)
)

// String returns the human-readable name of a SignatureKind.
func (k SignatureKind) String() string {
	switch k {
	case KindFunction:
		return "function"
	case KindClass:
		return "class"
	case KindStruct:
		return "struct"
	case KindInterface:
		return "interface"
	case KindType:
		return "type"
	case KindImport:
		return "import"
	case KindExport:
		return "export"
	case KindConstant:
		return "constant"
	case KindDocComment:
		return "doc_comment"
	default:
		return "unknown"
	}
}

// Signature represents a single extracted code element.
type Signature struct {
	Kind      SignatureKind
	Name      string // Identifier name (empty for imports/exports)
	Source    string // Verbatim source text at AST node boundaries
	StartLine int    // 1-based line number in original source
	EndLine   int    // 1-based line number in original source
}

// CompressedOutput is the result of compressing a single source file.
type CompressedOutput struct {
	Signatures   []Signature // Extracted structural elements in source order
	Language     string      // Language identifier
	OriginalSize int         // Original source size in bytes
	OutputSize   int         // Compressed output size in bytes
	NodeCount    int         // Number of AST nodes processed
}

// Render produces the compressed output as a string, preserving source order.
// Each signature's Source is included separated by blank lines.
func (co *CompressedOutput) Render() string {
	if len(co.Signatures) == 0 {
		return ""
	}
	var b strings.Builder
	for i, sig := range co.Signatures {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(sig.Source)
	}
	result := b.String()
	co.OutputSize = len(result)
	return result
}

// CompressionRatio returns the ratio of output to original size.
// 0.0 means fully compressed (no output), 1.0 means no compression.
// Returns 0.0 if OriginalSize is zero to avoid division by zero.
func (co *CompressedOutput) CompressionRatio() float64 {
	if co.OriginalSize == 0 {
		return 0.0
	}
	return float64(co.OutputSize) / float64(co.OriginalSize)
}