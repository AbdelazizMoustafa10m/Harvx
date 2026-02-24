package compression

import "regexp"

// RegexPattern defines a single extraction pattern for a language.
// Each pattern matches a specific structural element (function, class, etc.)
// and maps it to a SignatureKind.
type RegexPattern struct {
	Kind      SignatureKind
	Pattern   *regexp.Regexp
	MultiLine bool // When true, the extractor reads continuation lines until parentheses are balanced.
}

// regexPatternRegistry maps language identifiers to their pattern slices.
// Patterns are tried in order; first match wins per line.
var regexPatternRegistry = map[string][]*RegexPattern{
	"go":         goRegexPatterns(),
	"typescript": typescriptRegexPatterns(),
	"javascript": javascriptRegexPatterns(),
	"python":     pythonRegexPatterns(),
	"rust":       rustRegexPatterns(),
	"java":       javaRegexPatterns(),
	"c":          cRegexPatterns(),
	"cpp":        cppRegexPatterns(),
}

// goRegexPatterns returns regex patterns for Go source extraction.
func goRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^func\s+`), MultiLine: true},
		{Kind: KindType, Pattern: regexp.MustCompile(`^type\s+\w+\s+(struct|interface)\b`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^type\s+\w+`)},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^(const|var)\s+`)},
	}
}

// typescriptRegexPatterns returns regex patterns for TypeScript source extraction.
// Patterns are ordered so that more specific matches (interface, type, class,
// function, const) are tried before the generic export catch-all.
func typescriptRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+`)},
		{Kind: KindInterface, Pattern: regexp.MustCompile(`^(export\s+)?interface\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(export\s+)?type\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(export\s+)?(const\s+)?enum\s+\w+`)},
		{Kind: KindClass, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?(abstract\s+)?class\s+\w+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?(async\s+)?function\s+`), MultiLine: true},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^(export\s+)?const\s+\w+`)},
		{Kind: KindExport, Pattern: regexp.MustCompile(`^export\s+(\*|{)`)},
		{Kind: KindExport, Pattern: regexp.MustCompile(`^export\s+default\s+`)},
	}
}

// javascriptRegexPatterns returns regex patterns for JavaScript source extraction.
// Similar to TypeScript but without interface, type, and enum patterns.
func javascriptRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+`)},
		{Kind: KindClass, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?class\s+\w+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?(async\s+)?function\s+`), MultiLine: true},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^(export\s+)?const\s+\w+`)},
		{Kind: KindExport, Pattern: regexp.MustCompile(`^export\s+(\*|{)`)},
		{Kind: KindExport, Pattern: regexp.MustCompile(`^export\s+default\s+`)},
	}
}

// pythonRegexPatterns returns regex patterns for Python source extraction.
func pythonRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^(import\s+|from\s+\S+\s+import\s+)`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(async\s+)?def\s+\w+`), MultiLine: true},
		{Kind: KindClass, Pattern: regexp.MustCompile(`^class\s+\w+`)},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^[A-Z_][A-Z0-9_]*\s*[:=]`)},
	}
}

// rustRegexPatterns returns regex patterns for Rust source extraction.
func rustRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^use\s+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?(async\s+)?(unsafe\s+)?fn\s+\w+`), MultiLine: true},
		{Kind: KindStruct, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?struct\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?enum\s+\w+`)},
		{Kind: KindInterface, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?(unsafe\s+)?trait\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?type\s+\w+`)},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?const\s+\w+`)},
	}
}

// javaRegexPatterns returns regex patterns for Java source extraction.
func javaRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+`)},
		{Kind: KindImport, Pattern: regexp.MustCompile(`^package\s+`)},
		{Kind: KindClass, Pattern: regexp.MustCompile(`^(public\s+|private\s+|protected\s+)?(abstract\s+|static\s+|final\s+)*(class|interface|enum|@interface|record)\s+\w+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(public|private|protected)\s+.+\s+\w+\s*\(`), MultiLine: true},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^(static\s+|abstract\s+|final\s+|synchronized\s+|native\s+)+(public|private|protected)?\s*.*\s+\w+\s*\(`), MultiLine: true},
	}
}

// cRegexPatterns returns regex patterns for C source extraction.
func cRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^#include\s+`)},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^#define\s+`)},
		{Kind: KindStruct, Pattern: regexp.MustCompile(`^(typedef\s+)?struct\s+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(typedef\s+)?enum\s+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^typedef\s+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^\w[\w\s*]+\w+\s*\(`), MultiLine: true},
	}
}

// cppRegexPatterns returns regex patterns for C++ source extraction.
// Extends C patterns with class, namespace, and template support.
func cppRegexPatterns() []*RegexPattern {
	return []*RegexPattern{
		{Kind: KindImport, Pattern: regexp.MustCompile(`^#include\s+`)},
		{Kind: KindConstant, Pattern: regexp.MustCompile(`^#define\s+`)},
		{Kind: KindClass, Pattern: regexp.MustCompile(`^(template\s*<[^>]*>\s*)?(class|struct)\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^(typedef\s+)?enum\s+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^namespace\s+\w+`)},
		{Kind: KindType, Pattern: regexp.MustCompile(`^typedef\s+`)},
		{Kind: KindFunction, Pattern: regexp.MustCompile(`^\w[\w\s*:&<>,]+\w+\s*\(`), MultiLine: true},
	}
}

// getRegexPatterns returns the regex patterns for a language.
// Returns nil if no patterns are registered for the language.
func getRegexPatterns(language string) []*RegexPattern {
	return regexPatternRegistry[language]
}
