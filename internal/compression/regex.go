package compression

import (
	"context"
	"strings"
)

// Compile-time interface compliance check.
var _ LanguageCompressor = (*RegexCompressor)(nil)

// RegexCompressor uses regular expressions for heuristic signature extraction.
// It is less precise than the AST-based state-machine compressors but works
// without complex parsing logic and is faster for simple extractions.
//
// Each RegexCompressor is parameterized by a single language. Use
// NewRegexCompressor to create an instance for a specific language.
//
// RegexCompressor is stateless and safe for concurrent use.
type RegexCompressor struct {
	language string
	patterns []*RegexPattern
}

// NewRegexCompressor creates a regex-based compressor for the given language.
// If the language has no registered patterns, the compressor will produce
// no signatures (effectively a no-op).
func NewRegexCompressor(language string) *RegexCompressor {
	return &RegexCompressor{
		language: language,
		patterns: getRegexPatterns(language),
	}
}

// Compress parses the source using language-specific regular expressions and
// extracts structural signatures. The returned output contains verbatim source
// text; it never summarizes or rewrites code.
func (c *RegexCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
	if len(source) == 0 {
		return &CompressedOutput{
			Language:     c.language,
			OriginalSize: 0,
		}, nil
	}

	if len(c.patterns) == 0 {
		// No patterns for this language -- return empty output.
		return &CompressedOutput{
			Language:     c.language,
			OriginalSize: len(source),
		}, nil
	}

	lines := strings.Split(string(source), "\n")
	var sigs []Signature

	for i := 0; i < len(lines); i++ {
		// Check context cancellation every 1000 lines.
		if i%1000 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		lineNum := i + 1 // 1-based
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comment-only lines.
		if trimmed == "" {
			continue
		}

		// For most languages, only match non-indented lines to avoid
		// capturing function body code. Python is an exception since
		// indentation is part of its syntax (class methods are indented).
		if c.language != "python" && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}

		// Try each pattern against the trimmed line (first match wins).
		for _, pat := range c.patterns {
			if !pat.Pattern.MatchString(trimmed) {
				continue
			}

			var sigText string
			endLine := lineNum

			if pat.MultiLine {
				// Extract a potentially multi-line signature.
				sigText, endLine = extractMultiLineSignature(lines, i)
			} else {
				sigText = lines[i]
			}

			name := extractRegexName(trimmed, pat.Kind, c.language)

			sigs = append(sigs, Signature{
				Kind:      pat.Kind,
				Name:      name,
				Source:    sigText,
				StartLine: lineNum,
				EndLine:   endLine,
			})

			// Advance past multi-line signatures.
			if endLine > lineNum {
				i = endLine - 1 // -1 because the for loop will increment
			}

			break // First match wins.
		}
	}

	output := &CompressedOutput{
		Signatures:   sigs,
		Language:     c.language,
		OriginalSize: len(source),
		NodeCount:    len(sigs),
	}
	rendered := output.Render()
	output.OutputSize = len(rendered)

	return output, nil
}

// Language returns the language identifier for this compressor.
func (c *RegexCompressor) Language() string {
	return c.language
}

// SupportedNodeTypes returns the node types this compressor can extract.
func (c *RegexCompressor) SupportedNodeTypes() []string {
	seen := make(map[string]bool)
	var types []string
	for _, pat := range c.patterns {
		name := pat.Kind.String()
		if !seen[name] {
			seen[name] = true
			types = append(types, "regex_"+name)
		}
	}
	return types
}

// extractMultiLineSignature reads continuation lines starting from startIdx
// until parentheses are balanced. It stops when:
//   - All open parentheses are closed
//   - A line ending with '{' is found (start of body -- the '{' is excluded)
//   - The end of the file is reached
//
// Returns the accumulated signature text and the 1-based end line number.
func extractMultiLineSignature(lines []string, startIdx int) (string, int) {
	result := lines[startIdx]
	openParens := strings.Count(result, "(") - strings.Count(result, ")")
	openBraces := strings.Count(result, "{") - strings.Count(result, "}")

	// If parens are already balanced on the first line, check for body brace.
	if openParens <= 0 {
		// Trim any trailing body brace from the signature.
		sig := trimTrailingBodyBrace(result)
		return sig, startIdx + 1
	}

	endIdx := startIdx
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		openParens += strings.Count(trimmed, "(") - strings.Count(trimmed, ")")
		openBraces += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

		if openBraces > 0 {
			// Body started -- trim the opening brace and stop.
			trimmedBrace := strings.TrimRight(line, " \t")
			if strings.HasSuffix(trimmedBrace, "{") {
				trimmedBrace = strings.TrimRight(trimmedBrace[:len(trimmedBrace)-1], " \t")
			}
			if trimmedBrace != "" {
				result += "\n" + trimmedBrace
			}
			endIdx = i
			return result, endIdx + 1
		}

		result += "\n" + line
		endIdx = i

		if openParens <= 0 {
			// Parens balanced -- check if this line or remaining text ends the signature.
			return trimTrailingBodyBrace(result), endIdx + 1
		}
	}

	return result, endIdx + 1
}

// trimTrailingBodyBrace removes a trailing opening brace from a signature
// that indicates the start of a function/block body.
func trimTrailingBodyBrace(sig string) string {
	trimmed := strings.TrimRight(sig, " \t\n")
	if strings.HasSuffix(trimmed, "{") {
		trimmed = strings.TrimRight(trimmed[:len(trimmed)-1], " \t")
	}
	return trimmed
}

// extractRegexName attempts to extract a meaningful identifier name from the
// matched line. This is a best-effort extraction based on the signature kind
// and language.
func extractRegexName(trimmed string, kind SignatureKind, language string) string {
	switch kind {
	case KindFunction:
		return extractRegexFuncName(trimmed, language)
	case KindClass:
		return extractRegexClassName(trimmed)
	case KindStruct:
		return extractRegexStructName(trimmed)
	case KindInterface:
		return extractRegexInterfaceName(trimmed, language)
	case KindType:
		return extractRegexTypeName(trimmed, language)
	case KindConstant:
		return extractRegexConstName(trimmed, language)
	case KindImport, KindExport, KindDocComment:
		return "" // These typically don't have a single name.
	default:
		return ""
	}
}

// extractRegexFuncName extracts a function name from a declaration line.
func extractRegexFuncName(trimmed, language string) string {
	switch language {
	case "go":
		// func Name( or func (recv) Name(
		s := strings.TrimPrefix(trimmed, "func ")
		if strings.HasPrefix(s, "(") {
			// Method with receiver -- skip to closing paren.
			closeIdx := strings.Index(s, ")")
			if closeIdx >= 0 && closeIdx+1 < len(s) {
				s = strings.TrimSpace(s[closeIdx+1:])
			}
		}
		return extractWordBefore(s, "(")

	case "python":
		s := trimmed
		s = strings.TrimPrefix(s, "async ")
		s = strings.TrimPrefix(s, "def ")
		return extractWordBefore(s, "(")

	case "rust":
		// Find "fn " and take the word after it.
		idx := strings.Index(trimmed, "fn ")
		if idx >= 0 {
			rest := trimmed[idx+3:]
			return extractWordBefore(rest, "(")
		}
		return ""

	case "java":
		// Last identifier before the first '('.
		parenIdx := strings.Index(trimmed, "(")
		if parenIdx <= 0 {
			return ""
		}
		before := strings.TrimSpace(trimmed[:parenIdx])
		tokens := strings.Fields(before)
		if len(tokens) == 0 {
			return ""
		}
		return tokens[len(tokens)-1]

	case "typescript", "javascript":
		s := trimmed
		for _, prefix := range []string{"export ", "default ", "async ", "function ", "function*"} {
			s = strings.TrimPrefix(s, prefix)
		}
		s = strings.TrimSpace(s)
		return extractWordBefore(s, "(")

	case "c", "cpp":
		// Last identifier before '(' (stripping pointer indicators).
		parenIdx := strings.Index(trimmed, "(")
		if parenIdx <= 0 {
			return ""
		}
		before := strings.TrimSpace(trimmed[:parenIdx])
		tokens := strings.Fields(before)
		if len(tokens) == 0 {
			return ""
		}
		last := tokens[len(tokens)-1]
		last = strings.TrimLeft(last, "*&")
		// Strip any class/namespace prefix (e.g., "MyClass::method").
		if colIdx := strings.LastIndex(last, "::"); colIdx >= 0 {
			last = last[colIdx+2:]
		}
		return last

	default:
		return ""
	}
}

// extractRegexClassName extracts a class name from a declaration line.
func extractRegexClassName(trimmed string) string {
	// Strip common prefixes to find "class Name".
	s := trimmed
	for _, prefix := range []string{"export ", "default ", "abstract ", "declare ", "public ", "private ", "protected ", "static ", "final "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "class ")
	s = strings.TrimSpace(s)
	return extractWordBefore(s, " {(<:")
}

// extractRegexStructName extracts a struct name from a declaration line.
func extractRegexStructName(trimmed string) string {
	s := trimmed
	for _, prefix := range []string{"pub(crate) ", "pub(super) ", "pub "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimPrefix(s, "typedef ")
	s = strings.TrimPrefix(s, "struct ")
	s = strings.TrimSpace(s)
	return extractWordBefore(s, " {(<;")
}

// extractRegexInterfaceName extracts an interface/trait name from a declaration line.
func extractRegexInterfaceName(trimmed, language string) string {
	s := trimmed
	for _, prefix := range []string{"export ", "declare ", "pub(crate) ", "pub(super) ", "pub ", "unsafe "} {
		s = strings.TrimPrefix(s, prefix)
	}
	switch language {
	case "rust":
		s = strings.TrimPrefix(s, "trait ")
	default:
		s = strings.TrimPrefix(s, "interface ")
	}
	s = strings.TrimSpace(s)
	return extractWordBefore(s, " {(<")
}

// extractRegexTypeName extracts a type/enum name from a declaration line.
func extractRegexTypeName(trimmed, language string) string {
	s := trimmed
	for _, prefix := range []string{"export ", "declare ", "pub(crate) ", "pub(super) ", "pub ", "const "} {
		s = strings.TrimPrefix(s, prefix)
	}
	switch language {
	case "go":
		s = strings.TrimPrefix(s, "type ")
	case "rust":
		if strings.HasPrefix(s, "enum ") {
			s = strings.TrimPrefix(s, "enum ")
		} else {
			s = strings.TrimPrefix(s, "type ")
		}
	case "typescript":
		if strings.HasPrefix(s, "enum ") {
			s = strings.TrimPrefix(s, "enum ")
		} else {
			s = strings.TrimPrefix(s, "type ")
		}
	default:
		s = strings.TrimPrefix(s, "typedef ")
		s = strings.TrimPrefix(s, "enum ")
		s = strings.TrimPrefix(s, "type ")
		s = strings.TrimPrefix(s, "namespace ")
	}
	s = strings.TrimSpace(s)
	return extractWordBefore(s, " {(<;=")
}

// extractRegexConstName extracts a constant/variable name from a declaration line.
func extractRegexConstName(trimmed, language string) string {
	s := trimmed
	switch language {
	case "go":
		s = strings.TrimPrefix(s, "const ")
		s = strings.TrimPrefix(s, "var ")
	case "python":
		// Pattern: NAME = or NAME:
		return extractWordBefore(s, " :=")
	case "rust":
		for _, prefix := range []string{"pub(crate) ", "pub(super) ", "pub "} {
			s = strings.TrimPrefix(s, prefix)
		}
		s = strings.TrimPrefix(s, "const ")
	case "c", "cpp":
		if strings.HasPrefix(s, "#define ") {
			s = strings.TrimPrefix(s, "#define ")
			return extractWordBefore(s, " (")
		}
	default:
		s = strings.TrimPrefix(s, "export ")
		s = strings.TrimPrefix(s, "const ")
	}
	s = strings.TrimSpace(s)
	// Handle grouped declarations: "const (" -> no name.
	if strings.HasPrefix(s, "(") {
		return ""
	}
	return extractWordBefore(s, " :=(<")
}

// extractWordBefore extracts the first identifier-like word from s,
// stopping at any character in stopChars or at the end of s.
func extractWordBefore(s string, stopChars string) string {
	var b strings.Builder
	for _, ch := range s {
		if strings.ContainsRune(stopChars, ch) {
			break
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			break
		}
		b.WriteRune(ch)
	}
	result := b.String()
	// Strip any leading/trailing non-identifier characters.
	result = strings.Trim(result, "*&")
	return result
}
