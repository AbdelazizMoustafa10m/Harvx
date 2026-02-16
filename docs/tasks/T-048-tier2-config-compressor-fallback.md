# T-048: Tier 2 Config Compressors (JSON/YAML/TOML) and Unsupported Language Fallback

**Priority:** Must Have
**Effort:** Small (4-6hrs)
**Dependencies:** T-043
**Phase:** 3 - Security & Compression

---

## Description

Implement basic compressors for structured configuration file formats (JSON, YAML, TOML) and the fallback behavior for unsupported languages. Config file "compression" extracts top-level keys and structure without deep values, which is often sufficient for an LLM to understand a project's configuration shape. The unsupported language fallback simply returns the full file content unchanged, ensuring the compression pipeline never drops files silently.

JSON/YAML/TOML compressors do NOT require tree-sitter WASM grammars. They use lightweight, Go-native approaches: JSON uses `encoding/json`, YAML uses structure-aware line parsing, and TOML uses `BurntSushi/toml`. This keeps these compressors zero-dependency relative to the WASM subsystem.

## User Story

As a developer using Harvx with `--compress`, I want configuration files (package.json, tsconfig.json, docker-compose.yml, Cargo.toml) to be compressed to their structural skeleton so that token budget is preserved for actual source code, while unsupported file types are included verbatim.

## Acceptance Criteria

### JSON Compressor
- [ ] Extracts top-level keys and their value types (string, number, boolean, array, object)
- [ ] For nested objects, extracts keys up to depth 2 (configurable)
- [ ] Arrays show element count and type of first element (e.g., `"dependencies": [/* 45 items */]`)
- [ ] String values are truncated to 50 chars with `...` if longer
- [ ] Large arrays (> 5 elements) are collapsed to count + sample
- [ ] Output is valid JSON (preserves parseable structure)
- [ ] Handles `package.json`, `tsconfig.json`, `*.json` configuration files

### YAML Compressor
- [ ] Extracts top-level keys and structure up to depth 2
- [ ] Preserves comments (YAML comments are often semantically important)
- [ ] Lists are collapsed to count when > 5 items
- [ ] Multi-line string values are truncated with `...`
- [ ] Handles `docker-compose.yml`, `.github/workflows/*.yml`, etc.

### TOML Compressor
- [ ] Extracts section headers `[section]` and `[[array-of-tables]]`
- [ ] Extracts key names with value types at depth 1 within each section
- [ ] Preserves comments
- [ ] Handles `Cargo.toml`, `pyproject.toml`, `harvx.toml`

### Fallback
- [ ] Unsupported languages return full file content unchanged
- [ ] Fallback output is NOT marked as compressed (`IsCompressed = false` on FileDescriptor)
- [ ] No error is raised for unsupported languages -- fallback is silent
- [ ] Compressed output for config files IS marked compressed with the header marker

### All
- [ ] Config compressors achieve 30-60% reduction on typical config files
- [ ] Unit tests for each format with at least 3 fixture files

## Technical Notes

### JSON Compression Strategy

Use `encoding/json` to decode into `map[string]interface{}`, then walk the structure:

```go
func (c *JSONCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
    var data interface{}
    if err := json.Unmarshal(source, &data); err != nil {
        // Invalid JSON: fall back to full content
        return fullContentFallback(source, "json"), nil
    }
    
    compressed := compressJSON(data, 0, c.maxDepth)
    output, _ := json.MarshalIndent(compressed, "", "  ")
    // ... build CompressedOutput
}

// compressJSON recursively compresses JSON values.
// - Objects: keep keys, compress values recursively
// - Arrays: show count + type + first element sample
// - Strings: truncate at maxLen
// - Numbers/booleans: keep as-is
func compressJSON(v interface{}, depth, maxDepth int) interface{} { ... }
```

Example:
```json
// Input (package.json):
{
  "name": "my-app",
  "version": "1.0.0",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "test": "jest",
    "test:watch": "jest --watch"
  },
  "dependencies": {
    "next": "^15.0.0",
    "react": "^19.0.0",
    ... (40 more entries)
  }
}

// Output:
{
  "name": "my-app",
  "version": "1.0.0",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "test": "jest",
    "test:watch": "jest --watch"
  },
  "dependencies": "/* 42 entries */"
}
```

### YAML Compression Strategy

Line-based approach (avoid full YAML parsing to preserve comments):

1. Track indentation level to determine depth
2. At depth > maxDepth, collapse children to `# ... (N items)`
3. Preserve comment lines that appear at depth <= maxDepth
4. Truncate long string values

### TOML Compression Strategy

Use `BurntSushi/toml` to decode, then re-encode with structure only:

1. Preserve section headers `[section]` and `[[array]]`
2. Within each section, show key names and value types
3. Collapse large arrays to count
4. Preserve comments by working with raw lines alongside parsed structure

### Fallback Implementation

```go
// FallbackCompressor returns the full file content for unsupported languages.
type FallbackCompressor struct{}

func (f *FallbackCompressor) Compress(ctx context.Context, source []byte) (*CompressedOutput, error) {
    return &CompressedOutput{
        Signatures: []Signature{{
            Kind:   KindDocComment, // Using DocComment kind for raw content
            Source: string(source),
        }},
        Language:     "unknown",
        OriginalSize: len(source),
        OutputSize:   len(source),
    }, nil
}
```

The compression orchestrator (T-049) checks if the output is a fallback and sets `FileDescriptor.IsCompressed = false` accordingly.

### Key Design Decisions

- JSON/YAML/TOML compressors do NOT use tree-sitter WASM -- they use native Go parsing
- This keeps the dependency footprint minimal for these simple formats
- Config file structure (keys and nesting) is more valuable than values for LLM understanding
- Comments in YAML and TOML are preserved because they often contain important context
- Invalid/malformed config files fall back to full content (no error)

## Files to Create/Modify

- `internal/compression/json_compressor.go` -- JSON compressor
- `internal/compression/yaml_compressor.go` -- YAML compressor
- `internal/compression/toml_compressor.go` -- TOML compressor
- `internal/compression/fallback.go` -- Unsupported language fallback
- `internal/compression/json_compressor_test.go` -- JSON tests
- `internal/compression/yaml_compressor_test.go` -- YAML tests
- `internal/compression/toml_compressor_test.go` -- TOML tests
- `internal/compression/fallback_test.go` -- Fallback tests
- `testdata/compression/json/` -- JSON fixtures (package.json, tsconfig.json, etc.)
- `testdata/compression/yaml/` -- YAML fixtures (docker-compose.yml, CI config, etc.)
- `testdata/compression/toml/` -- TOML fixtures (Cargo.toml, pyproject.toml, etc.)

## Testing Requirements

### JSON
- Golden test: Compress a `package.json` with many dependencies
- Golden test: Compress a `tsconfig.json` with nested compiler options
- Unit test: Large array collapsing
- Unit test: Deeply nested object truncation
- Unit test: Invalid JSON falls back to full content

### YAML
- Golden test: Compress a `docker-compose.yml`
- Golden test: Compress a GitHub Actions workflow file
- Unit test: Comment preservation
- Unit test: Multi-line string truncation

### TOML
- Golden test: Compress a `Cargo.toml` with many dependencies
- Golden test: Compress a `pyproject.toml`
- Unit test: Section header extraction
- Unit test: Array of tables handling

### Fallback
- Unit test: Unknown file extension returns full content
- Unit test: IsCompressed is false for fallback output
- Unit test: Empty file returns empty output

## References

- [Go encoding/json](https://pkg.go.dev/encoding/json)
- [BurntSushi/toml](https://github.com/BurntSushi/toml)
- PRD Section 5.6: "For unsupported languages, falls back to full file content (no compression)"
- PRD Section 5.6: Tier 2 basic support for JSON, YAML, TOML