package compression

import "fmt"

// CompressEngine determines which compression implementation to use.
type CompressEngine string

const (
	// EngineAST uses the built-in state-machine parsers (default).
	// These are line-by-line parsers that extract structural signatures
	// with high accuracy for supported languages.
	EngineAST CompressEngine = "ast"

	// EngineRegex uses regular expression heuristics for signature extraction.
	// Less precise than AST-based compression but faster and simpler.
	// Works for any language with registered patterns.
	EngineRegex CompressEngine = "regex"

	// EngineAuto tries the AST engine first, falling back to regex on failure.
	// This is the recommended default for production use.
	EngineAuto CompressEngine = "auto"
)

// ValidEngines lists all valid compression engine values.
var ValidEngines = []CompressEngine{EngineAST, EngineRegex, EngineAuto}

// ParseCompressEngine parses a string into a CompressEngine.
// It accepts "ast", "regex", "auto", and "wasm" (alias for "ast").
// Returns an error for unrecognized values.
func ParseCompressEngine(s string) (CompressEngine, error) {
	switch s {
	case "ast":
		return EngineAST, nil
	case "wasm":
		// Accept "wasm" as an alias for "ast" since the original spec
		// used WASM terminology, but our AST compressors are state-machine based.
		return EngineAST, nil
	case "regex":
		return EngineRegex, nil
	case "auto":
		return EngineAuto, nil
	default:
		return "", fmt.Errorf("unknown compression engine %q (allowed: ast, regex, auto, wasm)", s)
	}
}

// String returns the string representation of the engine.
func (e CompressEngine) String() string {
	return string(e)
}
