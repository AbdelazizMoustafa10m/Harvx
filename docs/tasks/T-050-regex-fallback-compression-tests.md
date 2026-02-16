# T-050: Regex Heuristic Fallback and End-to-End Compression Tests

**Priority:** Must Have
**Effort:** Medium (10-14hrs)
**Dependencies:** T-043, T-049
**Phase:** 3 - Security & Compression

---

## Description

Implement the regex-based heuristic fallback compressor as a safety net for when WASM tree-sitter parsing is inadequate or unavailable, and build the comprehensive end-to-end test suite that validates the entire compression subsystem. The regex fallback uses language-specific regular expressions to extract function signatures, class/struct declarations, and import statements -- less precise than AST parsing but zero-dependency and fast. The E2E test suite validates compression correctness, token reduction targets, faithfulness (verbatim extraction), and performance benchmarks across all supported languages.

## User Story

As a developer relying on Harvx compression, I want a reliable fallback mechanism so that compression works even when WASM parsing encounters edge cases, and I want comprehensive tests to ensure compression never corrupts or misrepresents my source code.

## Acceptance Criteria

### Regex Fallback
- [ ] `RegexCompressor` implements the `LanguageCompressor` interface
- [ ] Per-language regex patterns extract:
  - Function/method signatures (line starting with `func`, `def`, `fn`, `function`, etc.)
  - Class/struct/interface declarations
  - Import/include/use statements
  - Type definitions and enum declarations
  - Export statements
  - Top-level constant declarations
- [ ] Regex patterns for at minimum: TypeScript, JavaScript, Go, Python, Rust, Java, C/C++
- [ ] Regex compressor can be used as a standalone alternative to WASM (not just a fallback)
- [ ] Configurable via a build tag or runtime flag: `--compress-engine=regex` vs `--compress-engine=wasm` (default: wasm)
- [ ] Regex fallback achieves 30-50% token reduction (lower than WASM but still useful)
- [ ] Regex patterns handle multi-line signatures (continuation lines)

### End-to-End Tests
- [ ] Golden test suite with real-world code samples for all Tier 1 languages
- [ ] Each golden test validates: correct signatures extracted, correct order, verbatim text, no body leakage
- [ ] Compression ratio benchmarks for each language (target ranges documented)
- [ ] Faithfulness tests: compressed output contains ONLY verbatim source text (never summarized/rewritten)
- [ ] Performance benchmarks: per-file and batch compression timing
- [ ] Regression test: known edge cases that previously failed
- [ ] Cross-engine comparison: WASM vs regex output for same input (WASM should be strictly better)

## Technical Notes

### Regex Fallback Architecture

```go
package compression

import "regexp"

// RegexCompressor uses regular expressions for heuristic signature extraction.
// It is less precise than AST-based compression but works without WASM.
type RegexCompressor struct {
    patterns map[string][]*RegexPattern
}

// RegexPattern defines a single extraction pattern for a language.
type RegexPattern struct {
    Kind    SignatureKind
    Pattern *regexp.Regexp
    // MultiLine indicates this pattern may span multiple lines.
    // When true, the extractor looks for continuation (open parens, trailing comma, etc.)
    MultiLine bool
}

func NewRegexCompressor() *RegexCompressor {
    c := &RegexCompressor{
        patterns: make(map[string][]*RegexPattern),
    }
    c.registerGoPatterns()
    c.registerTypeScriptPatterns()
    c.registerJavaScriptPatterns()
    c.registerPythonPatterns()
    c.registerRustPatterns()
    c.registerJavaPatterns()
    c.registerCPatterns()
    return c
}
```

### Regex Patterns by Language

**Go:**
```go
func (c *RegexCompressor) registerGoPatterns() {
    c.patterns["go"] = []*RegexPattern{
        {Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+(\([\s\S]*?\)|"[^"]+")`)},
        {Kind: KindFunction, Pattern: regexp.MustCompile(`^func\s+(\([^)]*\)\s+)?\w+\s*(\[[^\]]*\])?\s*\([^)]*\)(\s*\([^)]*\)|\s*\w+[\w.*\[\]]*)?`), MultiLine: true},
        {Kind: KindStruct, Pattern: regexp.MustCompile(`^type\s+\w+\s+struct\s*\{[\s\S]*?\n\}`)},
        {Kind: KindInterface, Pattern: regexp.MustCompile(`^type\s+\w+\s+interface\s*\{[\s\S]*?\n\}`)},
        {Kind: KindType, Pattern: regexp.MustCompile(`^type\s+\w+\s+\w+`)},
        {Kind: KindConstant, Pattern: regexp.MustCompile(`^(const|var)\s+(\([\s\S]*?\)|[^\n]+)`)},
    }
}
```

**TypeScript/JavaScript:**
```go
func (c *RegexCompressor) registerTypeScriptPatterns() {
    c.patterns["typescript"] = []*RegexPattern{
        {Kind: KindImport, Pattern: regexp.MustCompile(`^import\s+.*$`)},
        {Kind: KindExport, Pattern: regexp.MustCompile(`^export\s+(default\s+)?(type\s+|interface\s+|enum\s+|const\s+|function\s+|class\s+|abstract\s+)`)},
        {Kind: KindFunction, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?(async\s+)?function\s+\w+`), MultiLine: true},
        {Kind: KindClass, Pattern: regexp.MustCompile(`^(export\s+)?(default\s+)?(abstract\s+)?class\s+\w+`)},
        {Kind: KindInterface, Pattern: regexp.MustCompile(`^(export\s+)?interface\s+\w+`)},
        {Kind: KindType, Pattern: regexp.MustCompile(`^(export\s+)?type\s+\w+\s*(<[^>]*>)?\s*=`)},
        {Kind: KindType, Pattern: regexp.MustCompile(`^(export\s+)?enum\s+\w+`)},
        {Kind: KindConstant, Pattern: regexp.MustCompile(`^(export\s+)?const\s+\w+\s*:`)},
    }
}
```

**Python:**
```go
func (c *RegexCompressor) registerPythonPatterns() {
    c.patterns["python"] = []*RegexPattern{
        {Kind: KindImport, Pattern: regexp.MustCompile(`^(import\s+|from\s+\S+\s+import\s+).*$`)},
        {Kind: KindFunction, Pattern: regexp.MustCompile(`^(\s*@\w+.*\n)*\s*(async\s+)?def\s+\w+`), MultiLine: true},
        {Kind: KindClass, Pattern: regexp.MustCompile(`^(\s*@\w+.*\n)*\s*class\s+\w+`)},
        {Kind: KindConstant, Pattern: regexp.MustCompile(`^[A-Z_][A-Z0-9_]*\s*[:=]`)},
    }
}
```

**Rust:**
```go
func (c *RegexCompressor) registerRustPatterns() {
    c.patterns["rust"] = []*RegexPattern{
        {Kind: KindImport, Pattern: regexp.MustCompile(`^use\s+.*;\s*$`)},
        {Kind: KindFunction, Pattern: regexp.MustCompile(`^(\s*#\[.*\]\n)*\s*(pub(\([^)]*\))?\s+)?(async\s+)?(unsafe\s+)?fn\s+\w+`), MultiLine: true},
        {Kind: KindStruct, Pattern: regexp.MustCompile(`^(\s*#\[.*\]\n)*\s*(pub(\([^)]*\))?\s+)?struct\s+\w+`)},
        {Kind: KindType, Pattern: regexp.MustCompile(`^(\s*#\[.*\]\n)*\s*(pub(\([^)]*\))?\s+)?enum\s+\w+`)},
        {Kind: KindInterface, Pattern: regexp.MustCompile(`^(\s*#\[.*\]\n)*\s*(pub(\([^)]*\))?\s+)?(unsafe\s+)?trait\s+\w+`)},
        {Kind: KindType, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?type\s+\w+`)},
        {Kind: KindConstant, Pattern: regexp.MustCompile(`^(pub(\([^)]*\))?\s+)?const\s+\w+`)},
    }
}
```

### Multi-Line Handling

For patterns marked `MultiLine: true`, the extractor reads continuation lines until:
- Balanced parentheses are closed
- A line ending with `{` is found (start of body -- exclude this line's `{`)
- Two consecutive non-continuation lines (no trailing comma, no open bracket)

```go
func extractMultiLineSignature(lines []string, startIdx int) (string, int) {
    result := lines[startIdx]
    openParens := strings.Count(result, "(") - strings.Count(result, ")")
    openBrackets := strings.Count(result, "{") - strings.Count(result, "}")
    
    for i := startIdx + 1; i < len(lines) && openParens > 0; i++ {
        line := lines[i]
        openParens += strings.Count(line, "(") - strings.Count(line, ")")
        openBrackets += strings.Count(line, "{") - strings.Count(line, "}")
        
        if openBrackets > 0 {
            // Body started -- trim the opening brace and stop
            trimmed := strings.TrimRight(line, " \t{")
            if trimmed != "" {
                result += "\n" + trimmed
            }
            return result, i
        }
        result += "\n" + line
    }
    return result, startIdx
}
```

### Engine Selection

```go
// CompressEngine determines which compression implementation to use.
type CompressEngine string

const (
    EngineWASM  CompressEngine = "wasm"  // Default: tree-sitter via WASM
    EngineRegex CompressEngine = "regex" // Fallback: regex heuristics
    EngineAuto  CompressEngine = "auto"  // Try WASM, fall back to regex on failure
)
```

In `auto` mode (which should be the default internally), the orchestrator:
1. Attempts WASM compression first
2. On any WASM failure (parse error, timeout, runtime error), retries with regex
3. If regex also fails, falls back to full content

### E2E Test Suite Structure

```
testdata/
  compression/
    e2e/
      typescript/
        api-route.ts          # Next.js API route
        api-route.expected     # Expected compressed output
        react-component.tsx
        react-component.expected
      go/
        http-handler.go
        http-handler.expected
        service.go
        service.expected
      python/
        django-model.py
        django-model.expected
        fastapi-router.py
        fastapi-router.expected
      rust/
        struct-impl.rs
        struct-impl.expected
        trait-def.rs
        trait-def.expected
      mixed/
        sample-repo/           # Small multi-language project
          package.json
          src/app.ts
          server/main.go
          scripts/deploy.py
```

### Faithfulness Verification

Every compressed output must satisfy:
1. Every line in the compressed output appears verbatim in the original source
2. No lines are added that do not exist in the source (except the `<!-- Compressed -->` marker)
3. Lines appear in the same relative order as in the source

```go
func verifyFaithfulness(t *testing.T, original, compressed string) {
    t.Helper()
    compressedLines := strings.Split(compressed, "\n")
    for i, line := range compressedLines {
        if line == "<!-- Compressed: signatures only -->" {
            continue
        }
        if !strings.Contains(original, line) {
            t.Errorf("line %d in compressed output not found in original: %q", i+1, line)
        }
    }
}
```

## Files to Create/Modify

- `internal/compression/regex.go` -- RegexCompressor implementation
- `internal/compression/regex_patterns.go` -- Per-language regex pattern definitions
- `internal/compression/regex_test.go` -- Regex compressor unit tests
- `internal/compression/e2e_test.go` -- End-to-end compression test suite
- `internal/compression/faithfulness_test.go` -- Verbatim extraction verification
- `internal/compression/benchmark_test.go` -- Performance and compression ratio benchmarks
- `testdata/compression/e2e/` -- Full E2E test fixtures (input + expected per language)
- `internal/compression/orchestrator.go` -- Add `EngineAuto` logic (modify from T-049)

## Testing Requirements

### Regex Unit Tests
- Per-language: verify each regex pattern extracts the correct signatures
- Multi-line continuation: function with parameters spanning 3+ lines
- False positive check: ensure regex does not match string literals or comments
- Edge case: function keyword inside a string literal is NOT extracted

### E2E Tests
- Golden test per Tier 1 language (at least 2 files each) -- both WASM and regex
- Golden test for Tier 2 languages (at least 1 file each)
- Golden test for config files (JSON, YAML, TOML)
- Mixed-language project test: compress all files in a mini sample repo
- Verify `<!-- Compressed: signatures only -->` marker on every compressed file
- Verify `IsCompressed = false` on unsupported files

### Faithfulness Tests
- For every golden test file, verify that all compressed lines exist verbatim in original
- Verify no body code leaks into compressed output
- Verify doc comments are correctly attached (not duplicated)

### Benchmark Tests
- Compression ratio per language (document actual vs target)
- Per-file compression time (target: < 50ms for WASM, < 5ms for regex)
- Batch compression time for 100 files (target: < 3s)
- Compare WASM vs regex ratio and timing

### Regression Tests
- Known edge cases that previously caused issues (tracked in test file)
- Arrow function edge cases (TS/JS)
- Generic functions with complex type parameters (Go, Rust, TS)
- Large files (> 10KB) compression correctness and timing

## References

- PRD Section 5.6: "If WASM approach proves inadequate, fallback plan: regex-based heuristic signature extraction per language"
- PRD Section 5.6: "50-70% token reduction target"
- PRD Section 5.6: "Compression never alters semantics -- it extracts verbatim source text at AST node boundaries"
- PRD Section 9.1: "compression: per-language extraction correctness (compare extracted signatures to expected output)"
- PRD Section 9.3: "Performance Benchmarks"