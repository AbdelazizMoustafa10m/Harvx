package compression

import "strings"

// cState tracks the parser state for C/C++ parsers.
type cState int

const (
	cStateTopLevel       cState = iota
	cStateInBlockComment        // Inside /* ... */ block comment
	cStateInFuncBody            // Skipping function body (brace counting)
	cStateInStructBody          // Accumulating struct body
	cStateInEnumBody            // Accumulating enum body
	cStateInPreproc             // Accumulating multi-line preprocessor directive
)

// cParseCtx holds all mutable state for a C/C++ line-by-line parser.
type cParseCtx struct {
	state      cState
	braceDepth int

	// Accumulator for multi-line constructs (struct, enum bodies).
	accum          strings.Builder
	accumStartLine int

	// Preprocessor accumulation.
	preprocAccum strings.Builder

	// Doc comment tracking (C-style /** */ and // before declarations).
	docComment string

	// Collected signatures.
	sigs []Signature
}

// ---------------------------------------------------------------------------
// Shared brace counting for C/C++
// ---------------------------------------------------------------------------

// cCountBraces counts the net brace depth change for a C/C++ source line,
// ignoring braces inside double-quoted strings, single-quoted chars,
// // line comments, and /* */ block comments on the same line.
func cCountBraces(line string) int {
	depth := 0
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

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
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		// Line comment -- ignore rest of line.
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		// Block comment on same line -- skip to closing */.
		if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "*/")
			if end >= 0 {
				i = i + 2 + end + 1 // skip past */
				continue
			}
			// Block comment doesn't close on this line -- ignore rest.
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth
}

// ---------------------------------------------------------------------------
// Shared detection helpers for C/C++
// ---------------------------------------------------------------------------

// cControlKeywords are keywords that introduce blocks but are NOT function definitions.
var cControlKeywords = []string{
	"if", "else", "while", "for", "do", "switch", "case", "return",
	"goto", "break", "continue", "default",
}

// isCFuncDefinition detects function definitions.
// A line is a function definition if it has a return type followed by
// identifier followed by ( params ) followed by { and does NOT start
// with a control keyword.
func isCFuncDefinition(trimmed string) bool {
	// Must contain '(' and '{'.
	if !strings.Contains(trimmed, "(") || !strings.Contains(trimmed, "{") {
		return false
	}
	// Must not start with a control keyword.
	for _, kw := range cControlKeywords {
		if strings.HasPrefix(trimmed, kw+" ") || strings.HasPrefix(trimmed, kw+"(") {
			return false
		}
	}
	// Must not be a union definition or typedef (without struct/enum return type).
	if strings.HasPrefix(trimmed, "union ") || strings.HasPrefix(trimmed, "typedef ") {
		return false
	}
	// The '{' must come after ')' to be a function definition.
	parenClose := strings.LastIndex(trimmed, ")")
	braceOpen := strings.Index(trimmed, "{")
	if parenClose == -1 || braceOpen <= parenClose {
		return false
	}
	// For lines starting with struct/enum, the '(' must come before '{'
	// to distinguish function definitions (with struct/enum return types)
	// from struct/enum declarations.
	if strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "enum ") {
		parenOpen := strings.Index(trimmed, "(")
		return parenOpen != -1 && parenOpen < braceOpen
	}
	return true
}

// isCFuncPrototype detects function prototypes (declarations ending with ;).
// Pattern: return_type name(params);
func isCFuncPrototype(trimmed string) bool {
	if !strings.Contains(trimmed, "(") || !strings.HasSuffix(trimmed, ";") {
		return false
	}
	// Must not start with a control keyword.
	for _, kw := range cControlKeywords {
		if strings.HasPrefix(trimmed, kw+" ") || strings.HasPrefix(trimmed, kw+"(") {
			return false
		}
	}
	// Must not be typedef, union, or preprocessor.
	if strings.HasPrefix(trimmed, "typedef ") || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "union ") {
		return false
	}
	// Must have ')' before the ';'.
	semiIdx := strings.LastIndex(trimmed, ";")
	parenClose := strings.LastIndex(trimmed, ")")
	if parenClose == -1 || parenClose >= semiIdx {
		return false
	}
	// Lines starting with struct/enum are prototypes only if they have
	// the pattern of a function prototype (identifier + parens), not a
	// forward declaration like "struct Foo;".
	if strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "enum ") {
		// Must have parens -- already checked above.
		return true
	}
	return true
}

// isCStructDecl detects struct declarations.
// It returns false for function definitions that have a struct return type,
// such as "struct Config *config_new(const char *host, int port) {".
func isCStructDecl(trimmed string) bool {
	s := trimmed
	// Strip static/extern qualifiers.
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	if !strings.HasPrefix(s, "struct ") &&
		!(strings.HasPrefix(trimmed, "typedef struct") &&
			(strings.Contains(trimmed, "{") || strings.Contains(trimmed, ";"))) {
		return false
	}
	// If the line has '(' before '{', it is a function with a struct return type,
	// not a struct declaration.
	parenIdx := strings.Index(s, "(")
	braceIdx := strings.Index(s, "{")
	if parenIdx != -1 {
		// Has parens. If there is no brace or the paren comes before the brace,
		// this looks like a function, not a struct declaration.
		if braceIdx == -1 || parenIdx < braceIdx {
			// Exception: "typedef struct" with a brace is always a struct.
			if strings.HasPrefix(trimmed, "typedef struct") {
				return true
			}
			return false
		}
	}
	return true
}

// isCEnumDecl detects enum declarations.
// It returns false for function definitions that have an enum return type.
func isCEnumDecl(trimmed string) bool {
	s := trimmed
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	if !strings.HasPrefix(s, "enum ") &&
		!(strings.HasPrefix(trimmed, "typedef enum") &&
			(strings.Contains(trimmed, "{") || strings.Contains(trimmed, ";"))) {
		return false
	}
	// If the line has '(' before '{', it is a function with an enum return type,
	// not an enum declaration.
	parenIdx := strings.Index(s, "(")
	braceIdx := strings.Index(s, "{")
	if parenIdx != -1 {
		if braceIdx == -1 || parenIdx < braceIdx {
			if strings.HasPrefix(trimmed, "typedef enum") {
				return true
			}
			return false
		}
	}
	return true
}

// isCTypedef detects typedef statements.
func isCTypedef(trimmed string) bool {
	return strings.HasPrefix(trimmed, "typedef ")
}

// isCPreprocessorDirective checks if a trimmed line starts with #.
func isCPreprocessorDirective(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#")
}

// isCInclude checks if a trimmed line is a #include directive.
func isCInclude(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#include")
}

// isCDefine checks if a trimmed line is a #define directive.
func isCDefine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#define")
}

// isCForwardDecl detects forward declarations like "struct Foo;" or "enum Bar;".
func isCForwardDecl(trimmed string) bool {
	if !strings.HasSuffix(trimmed, ";") {
		return false
	}
	s := trimmed
	s = strings.TrimPrefix(s, "extern ")
	s = strings.TrimPrefix(s, "static ")
	return (strings.HasPrefix(s, "struct ") || strings.HasPrefix(s, "enum ") ||
		strings.HasPrefix(s, "union ")) && !strings.Contains(s, "{") &&
		!strings.Contains(s, "(")
}

// isCGlobalVarDecl detects global variable declarations.
// These are lines at the top level that contain a type + identifier + optional = + ;
// but are NOT function prototypes, typedefs, or preprocessor directives.
func isCGlobalVarDecl(trimmed string) bool {
	if !strings.HasSuffix(trimmed, ";") {
		return false
	}
	if strings.Contains(trimmed, "(") {
		return false
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "typedef ") {
		return false
	}
	if strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "enum ") ||
		strings.HasPrefix(trimmed, "union ") {
		// Could be a forward declaration -- not a variable.
		return false
	}
	// Must have at least a type and a name (two tokens minimum).
	tokens := strings.Fields(strings.TrimSuffix(trimmed, ";"))
	return len(tokens) >= 2
}

// ---------------------------------------------------------------------------
// Shared extraction helpers for C/C++
// ---------------------------------------------------------------------------

// extractCFuncSignature extracts the function signature up to (not including) the
// opening brace of the function body.
func extractCFuncSignature(line string) string {
	idx := findCBodyBrace(line)
	if idx == -1 {
		return strings.TrimRight(line, " \t;")
	}
	return strings.TrimRight(line[:idx], " \t")
}

// findCBodyBrace finds the index of the { that opens a function body.
// It looks for the first { after all parentheses are balanced.
func findCBodyBrace(line string) int {
	parenDepth := 0
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false
	seenParens := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

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
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		// Line comment -- stop scanning.
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}

		// Block comment on same line.
		if ch == '/' && i+1 < len(line) && line[i+1] == '*' {
			end := strings.Index(line[i+2:], "*/")
			if end >= 0 {
				i = i + 2 + end + 1
				continue
			}
			break
		}

		switch ch {
		case '"':
			inDoubleQuote = true
		case '\'':
			inSingleQuote = true
		case '(':
			parenDepth++
			seenParens = true
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '{':
			if seenParens && parenDepth == 0 {
				return i
			}
		}
	}
	return -1
}

// extractCFuncName extracts the function name from a function declaration line.
// It looks for the identifier immediately before the first '('.
func extractCFuncName(trimmed string) string {
	parenIdx := strings.Index(trimmed, "(")
	if parenIdx <= 0 {
		return ""
	}
	// Walk backwards from '(' to find the identifier.
	before := strings.TrimSpace(trimmed[:parenIdx])
	// The function name is the last token before '('.
	// Handle pointer returns: int *foo(
	// Handle qualifiers: static inline int foo(
	tokens := strings.Fields(before)
	if len(tokens) == 0 {
		return ""
	}
	last := tokens[len(tokens)-1]
	// Strip leading * or & for pointer/ref returns.
	last = strings.TrimLeft(last, "*&")
	return last
}

// extractCStructName extracts the struct name from a struct declaration line.
func extractCStructName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "typedef ")
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	s = strings.TrimPrefix(s, "struct ")
	s = strings.TrimSpace(s)
	return extractCIdentifier(s)
}

// extractCEnumName extracts the enum name from an enum declaration line.
func extractCEnumName(trimmed string) string {
	s := trimmed
	s = strings.TrimPrefix(s, "typedef ")
	s = strings.TrimPrefix(s, "static ")
	s = strings.TrimPrefix(s, "extern ")
	s = strings.TrimPrefix(s, "enum ")
	s = strings.TrimSpace(s)
	return extractCIdentifier(s)
}

// extractCDefineNameLine extracts the #define name and parameters.
// For "#define FOO(x, y) ...", returns "#define FOO(x, y)".
// For "#define BAR 42", returns "#define BAR".
func extractCDefineNameLine(trimmed string) string {
	rest := strings.TrimPrefix(trimmed, "#define ")
	rest = strings.TrimSpace(rest)

	// Find the macro name.
	nameEnd := 0
	for nameEnd < len(rest) {
		ch := rest[nameEnd]
		if ch == '(' || ch == ' ' || ch == '\t' {
			break
		}
		nameEnd++
	}

	if nameEnd >= len(rest) {
		// Just "#define NAME" with nothing else.
		return "#define " + rest
	}

	name := rest[:nameEnd]

	// If it has function-like params: #define FOO(x, y)
	if nameEnd < len(rest) && rest[nameEnd] == '(' {
		closeIdx := strings.Index(rest[nameEnd:], ")")
		if closeIdx != -1 {
			return "#define " + rest[:nameEnd+closeIdx+1]
		}
	}

	return "#define " + name
}

// extractCIdentifier extracts the first C identifier from a string.
func extractCIdentifier(s string) string {
	var b strings.Builder
	for i, ch := range s {
		if i == 0 {
			if !isCIdentStart(ch) {
				break
			}
		} else {
			if !isCIdentContinue(ch) {
				break
			}
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// isCIdentStart checks if a rune can start a C identifier.
func isCIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// isCIdentContinue checks if a rune can continue a C identifier.
func isCIdentContinue(ch rune) bool {
	return isCIdentStart(ch) || (ch >= '0' && ch <= '9')
}

// isCDocCommentStart checks if a trimmed line starts a /** doc comment.
func isCDocCommentStart(trimmed string) bool {
	return strings.HasPrefix(trimmed, "/**")
}

// isCBlockCommentStart checks if a trimmed line starts a /* comment.
func isCBlockCommentStart(trimmed string) bool {
	return strings.HasPrefix(trimmed, "/*")
}

// isCLineComment checks if a trimmed line is a // comment.
func isCLineComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//")
}

// isMultiLineDefine checks if a preprocessor line continues with backslash.
func isMultiLineDefine(trimmed string) bool {
	return strings.HasSuffix(trimmed, "\\")
}