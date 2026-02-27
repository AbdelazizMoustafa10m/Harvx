// Package compression implements tree-sitter based code compression for
// reducing token usage in LLM context documents. It uses the wazero WASM
// runtime to execute tree-sitter grammar parsers compiled to WebAssembly,
// enabling language-aware structural extraction without CGO dependencies.
//
// The GrammarRegistry manages lazy compilation and caching of WASM grammar
// modules. All methods are safe for concurrent use.
package compression

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/harvx/harvx/grammars"
	"github.com/tetratelabs/wazero"
)

// ErrUnknownLanguage is returned when a grammar is not available for the
// requested language. Use errors.Is to check for this sentinel.
var ErrUnknownLanguage = errors.New("unknown language")

// GrammarRegistry manages wazero compiled modules for tree-sitter grammars.
// It provides lazy compilation and caching of WASM grammar modules.
// All methods are safe for concurrent use.
type GrammarRegistry struct {
	runtime  wazero.Runtime
	compiled map[string]wazero.CompiledModule
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewGrammarRegistry creates a new GrammarRegistry with an initialized wazero
// runtime. The runtime uses compiler mode (default) on amd64/arm64 for
// near-native performance, and falls back to interpreter mode on other
// architectures.
//
// Call Close when done to release all WASM resources and avoid memory leaks.
func NewGrammarRegistry(ctx context.Context, logger *slog.Logger) (*GrammarRegistry, error) {
	if logger == nil {
		logger = slog.Default()
	}

	config := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true)

	rt := wazero.NewRuntimeWithConfig(ctx, config)

	logger.Debug("wazero runtime initialized",
		"languages", len(grammars.GrammarFiles),
	)

	return &GrammarRegistry{
		runtime:  rt,
		compiled: make(map[string]wazero.CompiledModule),
		logger:   logger,
	}, nil
}

// GetCompiledModule returns a compiled WASM module for the given language.
// Modules are lazily compiled on first access and cached for subsequent calls.
// Returns ErrUnknownLanguage if the language has no embedded grammar.
//
// The method uses a double-checked locking pattern for efficient concurrent
// access: a read lock for cache hits, upgrading to a write lock only on cache
// misses.
func (r *GrammarRegistry) GetCompiledModule(ctx context.Context, lang string) (wazero.CompiledModule, error) {
	// Fast path: read lock check cache.
	r.mu.RLock()
	if mod, ok := r.compiled[lang]; ok {
		r.mu.RUnlock()
		return mod, nil
	}
	r.mu.RUnlock()

	// Check if language is supported before acquiring write lock.
	filename, ok := grammars.GrammarFiles[lang]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownLanguage, lang)
	}

	// Slow path: write lock, double-check, compile.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock to handle concurrent compilation.
	if mod, ok := r.compiled[lang]; ok {
		return mod, nil
	}

	// Read embedded WASM bytes.
	wasmBytes, err := grammars.FS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading grammar %s: %w", lang, err)
	}

	r.logger.Debug("compiling grammar",
		"language", lang,
		"size_bytes", len(wasmBytes),
	)

	// Compile the WASM module.
	compiled, err := r.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compiling grammar %s: %w", lang, err)
	}

	r.compiled[lang] = compiled
	r.logger.Debug("grammar compiled", "language", lang)

	return compiled, nil
}

// SupportedLanguages returns a sorted list of languages with embedded grammars.
func (r *GrammarRegistry) SupportedLanguages() []string {
	langs := make([]string, 0, len(grammars.GrammarFiles))
	for lang := range grammars.GrammarFiles {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// HasLanguage checks if a grammar is available for the given language.
func (r *GrammarRegistry) HasLanguage(lang string) bool {
	_, ok := grammars.GrammarFiles[lang]
	return ok
}

// Close releases all compiled WASM modules and the wazero runtime.
// It should be called when the registry is no longer needed to avoid memory leaks.
// If multiple errors occur during cleanup, only the first is returned.
func (r *GrammarRegistry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for lang, mod := range r.compiled {
		if err := mod.Close(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("closing grammar %s: %w", lang, err)
		}
		delete(r.compiled, lang)
	}

	if err := r.runtime.Close(ctx); err != nil && firstErr == nil {
		firstErr = fmt.Errorf("closing wazero runtime: %w", err)
	}

	return firstErr
}

// Runtime returns the underlying wazero runtime for advanced use cases.
// This is needed by the parser layer to instantiate modules with host functions
// for tree-sitter parsing operations.
func (r *GrammarRegistry) Runtime() wazero.Runtime {
	return r.runtime
}