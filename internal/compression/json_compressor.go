package compression

import (
	"context"
	"encoding/json"
	"fmt"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*JSONCompressor)(nil)

// JSONCompressor implements LanguageCompressor for JSON configuration files.
// It extracts the structural skeleton by keeping keys, truncating long strings,
// collapsing large arrays, and limiting nesting depth. It uses encoding/json
// from the standard library and does not require tree-sitter WASM grammars.
// The compressor is stateless and safe for concurrent use.
type JSONCompressor struct {
	maxDepth      int // maximum nesting depth to preserve (default 2)
	maxStringLen  int // maximum string value length before truncation (default 50)
	maxArrayDisplay int // maximum array elements to display before collapsing (default 5)
}

// NewJSONCompressor creates a JSON compressor with sensible defaults.
func NewJSONCompressor() *JSONCompressor {
	return &JSONCompressor{
		maxDepth:      2,
		maxStringLen:  50,
		maxArrayDisplay: 5,
	}
}

// Compress parses JSON source and extracts a structural skeleton.
// If the input is invalid JSON, it falls back to returning the full content
// unchanged (no error). The output is a single Signature with Kind=KindDocComment
// containing the compressed JSON text.
func (c *JSONCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "json",
			OriginalSize: 0,
		}, nil
	}

	var data interface{}
	if err := json.Unmarshal(source, &data); err != nil {
		// Invalid JSON: fall back to full content unchanged.
		return c.fullContentFallback(source), nil
	}

	compressed := c.compressValue(data, 0)

	output, err := json.MarshalIndent(compressed, "", "  ")
	if err != nil {
		// Should not happen with our own structures, but fall back gracefully.
		return c.fullContentFallback(source), nil
	}

	rendered := string(output)

	return &CompressedOutput{
		Signatures: []Signature{{
			Kind:      KindDocComment,
			Name:      "json-skeleton",
			Source:    rendered,
			StartLine: 1,
			EndLine:   countLines(rendered),
		}},
		Language:     "json",
		OriginalSize: len(source),
		OutputSize:   len(rendered),
		NodeCount:    1,
	}, nil
}

// Language returns "json".
func (c *JSONCompressor) Language() string {
	return "json"
}

// SupportedNodeTypes returns the JSON node types this compressor handles.
func (c *JSONCompressor) SupportedNodeTypes() []string {
	return []string{"object", "array", "key_value"}
}

// compressValue recursively compresses a JSON value.
// Objects: keep all keys, recursively compress values up to maxDepth.
// Arrays: if > maxArrayDisplay elements, collapse to a summary string.
// Strings: truncate to maxStringLen with "..." suffix if longer.
// Numbers/booleans/null: keep as-is.
// Beyond maxDepth, objects become "/* object with N keys */" and arrays become "/* N items */".
func (c *JSONCompressor) compressValue(v interface{}, depth int) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return c.compressObject(val, depth)
	case []interface{}:
		return c.compressArray(val, depth)
	case string:
		return c.truncateString(val)
	default:
		// Numbers (float64), booleans, nil -- keep as-is.
		return v
	}
}

// compressObject compresses a JSON object. At depths beyond maxDepth, the entire
// object is replaced with a summary string.
func (c *JSONCompressor) compressObject(obj map[string]interface{}, depth int) interface{} {
	if depth >= c.maxDepth {
		return fmt.Sprintf("/* object with %d keys */", len(obj))
	}

	result := make(map[string]interface{}, len(obj))
	for key, val := range obj {
		result[key] = c.compressValue(val, depth+1)
	}
	return result
}

// compressArray compresses a JSON array. Large arrays (> maxArrayDisplay elements)
// are collapsed to a summary string. At depths beyond maxDepth, arrays are always
// collapsed.
func (c *JSONCompressor) compressArray(arr []interface{}, depth int) interface{} {
	if depth >= c.maxDepth {
		return fmt.Sprintf("/* %d items */", len(arr))
	}

	if len(arr) > c.maxArrayDisplay {
		return fmt.Sprintf("/* %d items */", len(arr))
	}

	result := make([]interface{}, len(arr))
	for i, val := range arr {
		result[i] = c.compressValue(val, depth+1)
	}
	return result
}

// truncateString truncates a string to maxStringLen characters, appending "..."
// if the string was longer than the limit.
func (c *JSONCompressor) truncateString(s string) string {
	runes := []rune(s)
	if len(runes) <= c.maxStringLen {
		return s
	}
	return string(runes[:c.maxStringLen]) + "..."
}

// fullContentFallback returns the full source content as a CompressedOutput
// when parsing fails.
func (c *JSONCompressor) fullContentFallback(source []byte) *CompressedOutput {
	content := string(source)
	return &CompressedOutput{
		Signatures: []Signature{{
			Kind:      KindDocComment,
			Name:      "json-raw",
			Source:    content,
			StartLine: 1,
			EndLine:   countLines(content),
		}},
		Language:     "json",
		OriginalSize: len(source),
		OutputSize:   len(source),
		NodeCount:    1,
	}
}

// countLines counts the number of lines in a string (1-based).
func countLines(s string) int {
	if len(s) == 0 {
		return 0
	}
	n := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}
