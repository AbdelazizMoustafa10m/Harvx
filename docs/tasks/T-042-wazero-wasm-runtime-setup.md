# T-042: Wazero WASM Runtime Setup and Grammar Embedding

**Priority:** Must Have
**Effort:** Medium (8-12hrs)
**Dependencies:** None (standalone foundation for compression subsystem)
**Phase:** 3 - Security & Compression

---

## Description

Set up the wazero WebAssembly runtime as the execution engine for tree-sitter grammars. This task establishes the foundational WASM infrastructure: initializing the wazero runtime with appropriate configuration, embedding prebuilt tree-sitter grammar `.wasm` files using `//go:embed` directives, and providing a grammar registry that lazily instantiates WASM modules on demand. This is the lowest layer of the compression subsystem -- everything else builds on top of it.

## User Story

As a developer building Harvx, I want a zero-CGO WASM runtime integrated into the binary so that tree-sitter grammars can execute on any platform without external dependencies or C compilation toolchains.

## Acceptance Criteria

- [ ] Wazero runtime initializes successfully with compiler mode (default) on amd64/arm64
- [ ] Grammar `.wasm` files for Tier 1 languages (TypeScript, JavaScript, Go, Python, Rust) are embedded via `//go:embed`
- [ ] Grammar `.wasm` files for Tier 2 languages (Java, C, C++) are embedded via `//go:embed`
- [ ] `GrammarRegistry` provides lazy-loaded, cached WASM module instances per language
- [ ] Runtime supports `context.Context` for cancellation and timeout propagation
- [ ] Runtime cleanup (via `Close()`) is properly handled to avoid memory leaks
- [ ] Unit tests verify grammar loading for all embedded languages
- [ ] Unit tests verify that unknown languages return an appropriate error
- [ ] Binary size impact of embedded grammars is documented (expected: 3-8MB total for all grammars)
- [ ] Benchmark test measures grammar instantiation time (target: < 100ms per grammar, < 500ms total cold start)

## Technical Notes

### WASM Runtime Selection

Use `github.com/tetratelabs/wazero` -- the only zero-dependency WebAssembly runtime for Go. Key properties:
- Pure Go, zero CGO, no system dependencies
- WebAssembly Core Spec 1.0 and 2.0 compliant
- Compiler mode (default on supported platforms) for near-native performance
- Interpreter mode as fallback on unsupported architectures
- Stable API with semantic versioning since v1.0 (March 2023)

### Grammar WASM Files

Obtain prebuilt `.wasm` grammar files from one of these sources:
1. **tree-sitter official releases** -- individual grammar repos publish `.wasm` in GitHub Releases
2. **sourcegraph/tree-sitter-wasms** -- prebuilt WASM binaries for many languages
3. **Menci/tree-sitter-wasm-prebuilt** -- alternative prebuilt source
4. **Build from source** -- `npx tree-sitter build --wasm` for each grammar

Each `.wasm` file contains the full tree-sitter runtime + grammar-specific parser tables. Typical sizes:
- JavaScript: ~600KB
- TypeScript: ~1.2MB (larger due to complex grammar)
- Go: ~400KB
- Python: ~500KB
- Rust: ~800KB
- Java/C/C++: ~300-600KB each

### Implementation Approach

```go
package compression

import (
    "context"
    "embed"
    "fmt"
    "sync"

    "github.com/tetratelabs/wazero"
    "github.com/tetratelabs/wazero/api"
)

//go:embed grammars/tree-sitter-typescript.wasm
var typescriptGrammar []byte

//go:embed grammars/tree-sitter-javascript.wasm
var javascriptGrammar []byte

//go:embed grammars/tree-sitter-go.wasm
var goGrammar []byte

//go:embed grammars/tree-sitter-python.wasm
var pythonGrammar []byte

//go:embed grammars/tree-sitter-rust.wasm
var rustGrammar []byte

//go:embed grammars/tree-sitter-java.wasm
var javaGrammar []byte

//go:embed grammars/tree-sitter-c.wasm
var cGrammar []byte

//go:embed grammars/tree-sitter-cpp.wasm
var cppGrammar []byte

// GrammarRegistry manages WASM module instances for tree-sitter grammars.
type GrammarRegistry struct {
    runtime  wazero.Runtime
    modules  map[string]api.Module
    grammars map[string][]byte
    mu       sync.RWMutex
}

// NewGrammarRegistry creates a new registry with the given wazero runtime.
func NewGrammarRegistry(ctx context.Context) (*GrammarRegistry, error) {
    rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
        WithCloseOnContextDone(true))

    reg := &GrammarRegistry{
        runtime:  rt,
        modules:  make(map[string]api.Module),
        grammars: make(map[string][]byte),
    }

    // Register embedded grammars
    reg.grammars["typescript"] = typescriptGrammar
    reg.grammars["javascript"] = javascriptGrammar
    reg.grammars["go"] = goGrammar
    reg.grammars["python"] = pythonGrammar
    reg.grammars["rust"] = rustGrammar
    reg.grammars["java"] = javaGrammar
    reg.grammars["c"] = cGrammar
    reg.grammars["cpp"] = cppGrammar

    return reg, nil
}

// GetModule returns a compiled WASM module for the given language.
// Modules are lazily instantiated and cached.
func (r *GrammarRegistry) GetModule(ctx context.Context, lang string) (api.Module, error) {
    // ... lazy instantiation with double-checked locking
}

// Close releases all WASM resources.
func (r *GrammarRegistry) Close(ctx context.Context) error {
    return r.runtime.Close(ctx)
}

// SupportedLanguages returns the list of languages with embedded grammars.
func (r *GrammarRegistry) SupportedLanguages() []string { ... }

// HasLanguage checks if a grammar is available for the given language.
func (r *GrammarRegistry) HasLanguage(lang string) bool { ... }
```

### Alternative Approach: malivvan/tree-sitter

Consider `github.com/malivvan/tree-sitter` -- a cgo-free tree-sitter wrapper that already uses wazero internally. This may simplify the integration significantly since it provides a higher-level API for parsing and querying. However, it is pre-release software (published January 2025), so evaluate stability carefully. If adopted, the `GrammarRegistry` becomes a thin wrapper around `malivvan/tree-sitter`'s language loading.

**Decision point:** During implementation, evaluate both approaches:
1. **Direct wazero** -- more control, more code to write, stable foundation
2. **malivvan/tree-sitter** -- less code, higher-level API, pre-release risk

Document the decision and rationale in a short ADR in `docs/adr/`.

### Key Considerations

- Use `wazero.NewRuntimeConfig().WithCloseOnContextDone(true)` to respect context cancellation
- Module compilation is expensive (~50-100ms per grammar); cache aggressively
- Consider compiling grammars ahead-of-time via `wazero.CompilationCache` for faster subsequent loads
- The registry must be safe for concurrent access (multiple goroutines compressing files in parallel)
- Each WASM file embeds the full tree-sitter runtime (~250KB overhead per grammar); this is expected

### Dependencies & Versions

| Package/Library | Version | Purpose |
|-----------------|---------|---------|
| github.com/tetratelabs/wazero | ^1.8.0 | Pure Go WASM runtime, zero CGO |
| (or) github.com/malivvan/tree-sitter | latest | Higher-level tree-sitter + wazero wrapper |

## Files to Create/Modify

- `internal/compression/wasm.go` -- GrammarRegistry, runtime initialization, module caching
- `internal/compression/wasm_test.go` -- Unit tests for grammar loading and caching
- `internal/compression/wasm_bench_test.go` -- Benchmark tests for instantiation time
- `grammars/` -- Directory containing embedded `.wasm` grammar files
- `grammars/README.md` -- Documentation of grammar sources, versions, and build instructions
- `scripts/fetch-grammars.sh` -- Script to download/build grammar `.wasm` files

## Testing Requirements

- Unit test: Initialize runtime, load each grammar, verify non-nil module
- Unit test: Request unknown language, verify error returned
- Unit test: Concurrent access to GetModule from multiple goroutines
- Unit test: Verify Close() releases resources properly
- Unit test: Verify SupportedLanguages() returns expected list
- Benchmark: Cold start time for first grammar load
- Benchmark: Warm cache time for subsequent grammar loads
- Integration test: Load grammar and parse a trivial source string (smoke test)

## References

- [Wazero Documentation](https://wazero.io/)
- [Wazero GitHub](https://github.com/tetratelabs/wazero)
- [Wazero Go Package](https://pkg.go.dev/github.com/tetratelabs/wazero)
- [malivvan/tree-sitter (CGO-free)](https://github.com/malivvan/tree-sitter)
- [Sourcegraph Prebuilt WASM Grammars](https://github.com/sourcegraph/tree-sitter-wasms)
- [Menci Prebuilt WASM Grammars](https://github.com/Menci/tree-sitter-wasm-prebuilt)
- [Tree-sitter WASM Size Discussion](https://github.com/tree-sitter/tree-sitter/issues/410)
- PRD Section 5.6: Tree-Sitter Code Compression (via WASM)
- PRD Section 6.1: Recommended Stack (tetratelabs/wazero)