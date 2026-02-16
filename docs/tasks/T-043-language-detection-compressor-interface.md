# T-043: Language Detection and LanguageCompressor Interface

**Priority:** Must Have
**Effort:** Small (3-4hrs)
**Dependencies:** None (interface-only; does not depend on WASM runtime)
**Phase:** 3 - Security & Compression

---

## Description

Define the `LanguageCompressor` interface that all language-specific compression implementations must satisfy, along with a file-extension-based language detection system and a compressor registry. This task establishes the contracts and extension points for the entire compression subsystem. The interface defines how source code is parsed and compressed into structural signatures. Language detection maps file extensions to language identifiers, and the registry dispatches compression requests to the appropriate implementation.

## User Story

As a developer extending Harvx with new language support, I want a clean interface and registry pattern so that adding compression for a new language requires only implementing one interface and registering it.

## Acceptance Criteria

- [ ] `LanguageCompressor` interface is defined with `Compress(ctx, source []byte) (CompressedOutput, error)` method
- [ ] `CompressedOutput` struct captures extracted signatures, import statements, type definitions, and metadata
- [ ] `LanguageDetector` maps file extensions to language identifiers (e.g., `.ts` -> `typescript`, `.go` -> `go`)
- [ ] `CompressorRegistry` allows registering and looking up compressors by language identifier
- [ ] Language detection handles ambiguous extensions (e.g., `.h` -> `c` by default)
- [ ] All Tier 1 and Tier 2 file extensions are mapped
- [ ] Unknown extensions return empty string (triggering fallback to uncompressed content)
- [ ] Unit tests cover all extension mappings
- [ ] Unit tests verify registry lookup behavior for registered, unregistered, and fallback cases

## Technical Notes

### Core Interface Design

```go
package compression

import "context"

// SignatureKind classifies the type of extracted code signature.
type SignatureKind int

const (
    KindFunction    SignatureKind = iota // Function or method signature
    KindClass                           // Class declaration
    KindStruct                          // Struct declaration
    KindInterface                       // Interface declaration
    KindType                            // Type alias or enum
    KindImport                          // Import/require statement
    KindExport                          // Export statement
    KindConstant                        // Top-level constant declaration
    KindDocComment                      // Doc comment (not inline)
)

// Signature represents a single extracted code element.
type Signature struct {
    Kind       SignatureKind
    Name       string // Identifier name (empty for imports/exports)
    Source     string // Verbatim source text at AST node boundaries
    StartLine  int    // 1-based line number in original source
    EndLine    int    // 1-based line number in original source
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
func (co *CompressedOutput) Render() string { ... }

// CompressionRatio returns the ratio of output to original (0.0 = fully compressed, 1.0 = no compression).
func (co *CompressedOutput) CompressionRatio() float64 { ... }

// LanguageCompressor compresses source code for a specific language.
type LanguageCompressor interface {
    // Compress parses the source and extracts structural signatures.
    // The returned output contains verbatim source text at AST node boundaries.
    // It must never summarize or rewrite code.
    Compress(ctx context.Context, source []byte) (*CompressedOutput, error)

    // Language returns the language identifier (e.g., "typescript", "go").
    Language() string

    // SupportedNodeTypes returns the AST node types this compressor extracts.
    SupportedNodeTypes() []string
}
```

### Language Detection

```go
// LanguageDetector maps file extensions to language identifiers.
type LanguageDetector struct {
    extMap map[string]string // ".ts" -> "typescript"
}

// DetectLanguage returns the language identifier for a file path.
// Returns empty string if the language is not recognized.
func (d *LanguageDetector) DetectLanguage(filePath string) string { ... }
```

Extension mappings:

| Extension(s) | Language | Tier |
|--------------|----------|------|
| `.ts`, `.tsx`, `.mts`, `.cts` | typescript | 1 |
| `.js`, `.jsx`, `.mjs`, `.cjs` | javascript | 1 |
| `.go` | go | 1 |
| `.py`, `.pyi` | python | 1 |
| `.rs` | rust | 1 |
| `.java` | java | 2 |
| `.c` | c | 2 |
| `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hxx` | cpp | 2 |
| `.h` | c | 2 (default; could be C or C++) |
| `.json` | json | 2 |
| `.yaml`, `.yml` | yaml | 2 |
| `.toml` | toml | 2 |

### Compressor Registry

```go
// CompressorRegistry manages language compressor implementations.
type CompressorRegistry struct {
    compressors map[string]LanguageCompressor
    detector    *LanguageDetector
}

// NewCompressorRegistry creates a registry with all built-in compressors.
func NewCompressorRegistry(detector *LanguageDetector) *CompressorRegistry { ... }

// Register adds a compressor for a language.
func (r *CompressorRegistry) Register(c LanguageCompressor) { ... }

// Get returns the compressor for a given file path, or nil if unsupported.
func (r *CompressorRegistry) Get(filePath string) LanguageCompressor { ... }

// IsSupported checks if compression is available for a file path.
func (r *CompressorRegistry) IsSupported(filePath string) bool { ... }
```

### Key Design Decisions

- **Verbatim extraction**: `Signature.Source` must contain exact text from the original source at AST node boundaries. The compressor must never summarize, rewrite, or reformat code.
- **Source order preservation**: `CompressedOutput.Signatures` must be ordered by `StartLine` to maintain readability.
- **Doc comments**: Attach doc comments to the following declaration, not as standalone signatures. If no declaration follows, include as standalone `KindDocComment`.
- **Stateless interface**: `LanguageCompressor` implementations should be stateless and safe for concurrent use.

## Files to Create/Modify

- `internal/compression/types.go` -- `SignatureKind`, `Signature`, `CompressedOutput` types
- `internal/compression/interface.go` -- `LanguageCompressor` interface definition
- `internal/compression/detector.go` -- `LanguageDetector` with extension mappings
- `internal/compression/registry.go` -- `CompressorRegistry` implementation
- `internal/compression/detector_test.go` -- Tests for language detection
- `internal/compression/registry_test.go` -- Tests for registry lookup

## Testing Requirements

- Unit test: Every extension in the mapping table resolves correctly
- Unit test: Unknown extensions (`.xyz`, `.md`, `.txt`) return empty string
- Unit test: Case sensitivity of extensions (`.Go` vs `.go`)
- Unit test: Registry returns correct compressor for each language
- Unit test: Registry returns nil for unsupported languages
- Unit test: CompressedOutput.Render() produces expected string format
- Unit test: CompressedOutput.CompressionRatio() calculation correctness
- Unit test: Signature ordering in output

## References

- PRD Section 5.6: Tree-Sitter Code Compression (via WASM)
- PRD Section 6.7: Internal API Boundaries (LanguageCompressor interface)
- PRD Section 6.5: Central Data Types (FileDescriptor.Language, FileDescriptor.IsCompressed)
- [Tree-sitter Static Node Types](https://tree-sitter.github.io/tree-sitter/using-parsers/6-static-node-types.html)