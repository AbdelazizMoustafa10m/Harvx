package compression

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*TOMLCompressor)(nil)

// TOMLCompressor implements LanguageCompressor for TOML configuration files.
// It uses line-based parsing to preserve comments and extract the structural
// skeleton. Section headers, array-of-tables markers, comments, and blank lines
// are always preserved. Key-value pairs are kept with truncated or collapsed
// values depending on type and size. It does NOT use BurntSushi/toml or any
// external TOML library -- pure line-based approach. The compressor is stateless
// and safe for concurrent use.
type TOMLCompressor struct {
	maxArrayItems int // inline array items to display before collapsing (default 5)
	maxStringLen  int // max string value length before truncation (default 80)
}

// NewTOMLCompressor creates a TOML compressor with sensible defaults.
func NewTOMLCompressor() *TOMLCompressor {
	return &TOMLCompressor{
		maxArrayItems: 5,
		maxStringLen:  80,
	}
}

// Compress parses TOML source line-by-line and produces a structural skeleton.
// Comments, section headers, blank lines, and key names are preserved. Long
// strings are truncated, large arrays are collapsed to item counts, and
// multi-line arrays/strings are handled via state tracking. The output is a
// single Signature with Kind=KindDocComment containing the compressed TOML text.
func (c *TOMLCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(source) == 0 {
		return &CompressedOutput{
			Language:     "toml",
			OriginalSize: 0,
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	compressed := c.compressLines(ctx, lines)
	rendered := strings.Join(compressed, "\n")

	return &CompressedOutput{
		Signatures: []Signature{{
			Kind:      KindDocComment,
			Name:      "toml-skeleton",
			Source:    rendered,
			StartLine: 1,
			EndLine:   countLines(rendered),
		}},
		Language:     "toml",
		OriginalSize: len(source),
		OutputSize:   len(rendered),
		NodeCount:    1,
	}, nil
}

// Language returns "toml".
func (c *TOMLCompressor) Language() string {
	return "toml"
}

// SupportedNodeTypes returns the TOML structural elements this compressor handles.
func (c *TOMLCompressor) SupportedNodeTypes() []string {
	return []string{"table", "array_of_tables", "key_value", "comment"}
}

// tomlParserState tracks multi-line context during line-by-line parsing.
type tomlParserState int

const (
	tomlStateNormal         tomlParserState = iota // Processing normal lines
	tomlStateMultiLineArray                        // Inside a multi-line array value
	tomlStateMultiLineStr                          // Inside a triple-quoted string (""" or ''')
)

// compressLines processes TOML lines and returns the compressed output lines.
func (c *TOMLCompressor) compressLines(ctx context.Context, lines []string) []string {
	var out []string
	state := tomlStateNormal

	// Multi-line array tracking.
	var arrayKey string
	var arrayItemCount int
	var arrayIndent string

	// Multi-line string tracking.
	var mlStringKey string
	var mlStringQuote string // `"""` or `'''`
	var mlStringLines []string
	var mlStringIndent string

	for i, line := range lines {
		// Check context cancellation every 1000 lines.
		if i%1000 == 0 {
			if ctx.Err() != nil {
				break
			}
		}

		switch state {
		case tomlStateMultiLineArray:
			trimmed := strings.TrimSpace(line)

			// Count items: non-empty, non-comment lines that aren't just the closing bracket.
			if trimmed == "]" || trimmed == "]," {
				// Closing bracket -- emit the collapsed array.
				out = append(out, fmt.Sprintf("%s%s = [/* %d items */]", arrayIndent, arrayKey, arrayItemCount))
				state = tomlStateNormal
				continue
			}
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				arrayItemCount++
			}
			continue

		case tomlStateMultiLineStr:
			// Check if this line closes the multi-line string.
			if strings.Contains(line, mlStringQuote) {
				// Emit a truncated version.
				combined := strings.Join(mlStringLines, " ")
				truncated := c.truncateString(combined)
				quote := `"`
				if strings.HasPrefix(mlStringQuote, "'") {
					quote = "'"
				}
				out = append(out, fmt.Sprintf("%s%s = %s%s...%s", mlStringIndent, mlStringKey, quote, truncated, quote))
				state = tomlStateNormal
				mlStringLines = nil
				continue
			}
			mlStringLines = append(mlStringLines, strings.TrimSpace(line))
			continue

		case tomlStateNormal:
			// Fall through to normal processing below.
		}

		trimmed := strings.TrimSpace(line)

		// Blank lines: preserve as-is.
		if trimmed == "" {
			out = append(out, line)
			continue
		}

		// Comment lines: always preserve.
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}

		// Section header: [section] or [section.subsection]
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[[") {
			out = append(out, line)
			continue
		}

		// Array of tables: [[array.of.tables]]
		if strings.HasPrefix(trimmed, "[[") {
			out = append(out, line)
			continue
		}

		// Key-value pair: key = value
		key, value, indent, ok := c.parseKeyValue(line)
		if !ok {
			// Not recognized as key-value; preserve as-is (e.g., continuation).
			out = append(out, line)
			continue
		}

		valueTrimmed := strings.TrimSpace(value)

		// Check for multi-line basic string: """
		if strings.HasPrefix(valueTrimmed, `"""`) {
			// Check if it also closes on this line (after the opening).
			rest := valueTrimmed[3:]
			if strings.Contains(rest, `"""`) {
				// Single-line triple-quoted string -- truncate normally.
				inner := rest[:strings.Index(rest, `"""`)]
				truncated := c.truncateString(inner)
				out = append(out, fmt.Sprintf(`%s%s = """%s"""`, indent, key, truncated))
				continue
			}
			state = tomlStateMultiLineStr
			mlStringKey = key
			mlStringQuote = `"""`
			mlStringIndent = indent
			mlStringLines = nil
			if len(rest) > 0 {
				mlStringLines = append(mlStringLines, strings.TrimSpace(rest))
			}
			continue
		}

		// Check for multi-line literal string: '''
		if strings.HasPrefix(valueTrimmed, `'''`) {
			rest := valueTrimmed[3:]
			if strings.Contains(rest, `'''`) {
				inner := rest[:strings.Index(rest, `'''`)]
				truncated := c.truncateString(inner)
				out = append(out, fmt.Sprintf(`%s%s = '''%s'''`, indent, key, truncated))
				continue
			}
			state = tomlStateMultiLineStr
			mlStringKey = key
			mlStringQuote = `'''`
			mlStringIndent = indent
			mlStringLines = nil
			if len(rest) > 0 {
				mlStringLines = append(mlStringLines, strings.TrimSpace(rest))
			}
			continue
		}

		// Check for multi-line array: value starts with [ but doesn't end with ]
		if strings.HasPrefix(valueTrimmed, "[") && !c.isClosedInlineArray(valueTrimmed) {
			state = tomlStateMultiLineArray
			arrayKey = key
			arrayIndent = indent
			// Count items on the opening line (after the [).
			arrayItemCount = c.countInlineItems(valueTrimmed[1:])
			continue
		}

		// Inline array: [item1, item2, ...]
		if strings.HasPrefix(valueTrimmed, "[") {
			compressed := c.compressInlineArray(valueTrimmed)
			out = append(out, fmt.Sprintf("%s%s = %s", indent, key, compressed))
			continue
		}

		// Inline table: { key = val, ... }
		if strings.HasPrefix(valueTrimmed, "{") {
			// Preserve inline tables as-is (they're typically short).
			out = append(out, line)
			continue
		}

		// Basic string: "..."
		if strings.HasPrefix(valueTrimmed, `"`) {
			compressed := c.compressStringValue(valueTrimmed, `"`)
			out = append(out, fmt.Sprintf("%s%s = %s", indent, key, compressed))
			continue
		}

		// Literal string: '...'
		if strings.HasPrefix(valueTrimmed, `'`) {
			compressed := c.compressStringValue(valueTrimmed, `'`)
			out = append(out, fmt.Sprintf("%s%s = %s", indent, key, compressed))
			continue
		}

		// Numbers, booleans, dates: keep as-is.
		out = append(out, line)
	}

	// Handle unterminated multi-line state at EOF.
	switch state {
	case tomlStateMultiLineArray:
		out = append(out, fmt.Sprintf("%s%s = [/* %d items */]", arrayIndent, arrayKey, arrayItemCount))
	case tomlStateMultiLineStr:
		combined := strings.Join(mlStringLines, " ")
		truncated := c.truncateString(combined)
		quote := `"`
		if strings.HasPrefix(mlStringQuote, "'") {
			quote = "'"
		}
		out = append(out, fmt.Sprintf("%s%s = %s%s...%s", mlStringIndent, mlStringKey, quote, truncated, quote))
	}

	return out
}

// parseKeyValue splits a TOML line into key, value, and leading indent.
// Returns ok=false if the line is not a key-value pair.
// Handles bare keys, quoted keys, and dotted keys.
func (c *TOMLCompressor) parseKeyValue(line string) (key, value, indent string, ok bool) {
	// Determine leading whitespace (indent).
	trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
	indent = line[:len(line)-len(trimmed)]

	// Skip non-key-value lines.
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
		return "", "", "", false
	}

	// Find the = separator, respecting quoted keys.
	eqIdx := c.findEquals(trimmed)
	if eqIdx < 0 {
		return "", "", "", false
	}

	key = strings.TrimSpace(trimmed[:eqIdx])
	value = strings.TrimSpace(trimmed[eqIdx+1:])

	if key == "" {
		return "", "", "", false
	}

	return key, value, indent, true
}

// findEquals finds the index of the first unquoted '=' in a trimmed TOML line.
// Returns -1 if not found.
func (c *TOMLCompressor) findEquals(trimmed string) int {
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if escaped {
			escaped = false
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '=':
			return i
		}
	}
	return -1
}

// isClosedInlineArray checks if an array value opens and closes on the same line.
// Handles nested brackets properly.
func (c *TOMLCompressor) isClosedInlineArray(value string) bool {
	depth := 0
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(value); i++ {
		ch := value[i]

		if escaped {
			escaped = false
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return true
			}
		case '#':
			// Unquoted # starts a comment; stop scanning.
			return false
		}
	}
	return false
}

// countInlineItems counts comma-separated items in a partial array string.
// Used for the first line of a multi-line array after the opening bracket.
func (c *TOMLCompressor) countInlineItems(s string) int {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return 0
	}

	count := 0
	depth := 0
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false
	hasContent := false

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if escaped {
			escaped = false
			hasContent = true
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			hasContent = true
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			hasContent = true
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
			hasContent = true
		case '\'':
			inSingleQuote = true
			hasContent = true
		case '[', '{':
			depth++
			hasContent = true
		case ']', '}':
			depth--
			hasContent = true
		case ',':
			if depth == 0 && hasContent {
				count++
				hasContent = false
			}
		case '#':
			if depth == 0 {
				// Comment; stop scanning.
				if hasContent {
					count++
				}
				return count
			}
			hasContent = true
		case ' ', '\t':
			// Whitespace doesn't count as content.
		default:
			hasContent = true
		}
	}

	if hasContent {
		count++
	}
	return count
}

// compressInlineArray compresses an inline TOML array value.
// If the array has more items than maxArrayItems, it is collapsed to [/* N items */].
func (c *TOMLCompressor) compressInlineArray(value string) string {
	count := c.countArrayItems(value)
	if count > c.maxArrayItems {
		// Strip trailing comment if present.
		return fmt.Sprintf("[/* %d items */]", count)
	}
	return value
}

// countArrayItems counts items in a complete inline array string like "[1, 2, 3]".
func (c *TOMLCompressor) countArrayItems(value string) int {
	// Remove outer brackets.
	inner := strings.TrimSpace(value)
	if len(inner) < 2 {
		return 0
	}

	// Find the matching closing bracket.
	closingIdx := c.findMatchingBracket(inner)
	if closingIdx < 0 {
		closingIdx = len(inner) - 1
	}

	inner = strings.TrimSpace(inner[1:closingIdx])
	if inner == "" {
		return 0
	}

	return c.countInlineItems(inner)
}

// findMatchingBracket finds the index of the ] that matches the opening [.
func (c *TOMLCompressor) findMatchingBracket(s string) int {
	depth := 0
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escaped {
			escaped = false
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// compressStringValue truncates a quoted TOML string value if it exceeds maxStringLen.
// The quote parameter is either `"` or `'`.
func (c *TOMLCompressor) compressStringValue(value string, quote string) string {
	// Handle trailing inline comment.
	rawValue, comment := c.splitTrailingComment(value)
	rawValue = strings.TrimSpace(rawValue)

	// Extract the inner string content.
	if len(rawValue) < 2 || !strings.HasPrefix(rawValue, quote) {
		return value
	}

	// Find the closing quote.
	closeIdx := strings.LastIndex(rawValue[1:], quote)
	if closeIdx < 0 {
		return value
	}
	closeIdx++ // Adjust for the offset of 1.

	inner := rawValue[1:closeIdx]
	runes := []rune(inner)

	if len(runes) <= c.maxStringLen {
		if comment != "" {
			return rawValue + " " + comment
		}
		return rawValue
	}

	truncated := string(runes[:c.maxStringLen]) + "..."
	result := quote + truncated + quote

	if comment != "" {
		return result + " " + comment
	}
	return result
}

// splitTrailingComment splits a TOML value from its trailing inline comment.
// Returns the value portion and the comment portion (including the #).
func (c *TOMLCompressor) splitTrailingComment(value string) (string, string) {
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(value); i++ {
		ch := value[i]

		if escaped {
			escaped = false
			continue
		}

		if inDoubleQuote {
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if inSingleQuote {
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '#':
			return strings.TrimSpace(value[:i]), strings.TrimSpace(value[i:])
		}
	}

	return value, ""
}

// truncateString truncates a plain string to maxStringLen runes.
func (c *TOMLCompressor) truncateString(s string) string {
	runes := []rune(s)
	if len(runes) <= c.maxStringLen {
		return s
	}
	return string(runes[:c.maxStringLen])
}
