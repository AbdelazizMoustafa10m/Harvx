package compression

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a silent logger for tests to avoid noisy output.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(testLogWriter{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// testLogWriter discards all log output.
type testLogWriter struct{}

func (testLogWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestNewGrammarRegistry(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err, "NewGrammarRegistry should succeed")
	require.NotNil(t, reg, "registry should not be nil")

	assert.NotNil(t, reg.runtime, "runtime should be initialized")
	assert.NotNil(t, reg.compiled, "compiled map should be initialized")
	assert.Empty(t, reg.compiled, "compiled map should be empty initially")

	require.NoError(t, reg.Close(ctx))
}

func TestNewGrammarRegistry_NilLogger(t *testing.T) {
	ctx := context.Background()

	reg, err := NewGrammarRegistry(ctx, nil)
	require.NoError(t, err, "NewGrammarRegistry with nil logger should succeed")
	require.NotNil(t, reg)
	assert.NotNil(t, reg.logger, "should fall back to slog.Default()")

	require.NoError(t, reg.Close(ctx))
}

func TestGetCompiledModule_AllLanguages(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	languages := []string{
		"typescript",
		"javascript",
		"go",
		"python",
		"rust",
		"java",
		"c",
		"cpp",
	}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			mod, err := reg.GetCompiledModule(ctx, lang)
			require.NoError(t, err, "GetCompiledModule should succeed for %s", lang)
			assert.NotNil(t, mod, "compiled module should not be nil for %s", lang)
		})
	}
}

func TestGetCompiledModule_CacheHit(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	// First call: compiles and caches.
	mod1, err := reg.GetCompiledModule(ctx, "go")
	require.NoError(t, err)
	require.NotNil(t, mod1)

	// Second call: should return cached module.
	mod2, err := reg.GetCompiledModule(ctx, "go")
	require.NoError(t, err)
	require.NotNil(t, mod2)

	// Both should be the same compiled module instance.
	assert.Equal(t, mod1, mod2, "second call should return cached module")
}

func TestGetCompiledModule_UnknownLanguage(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	tests := []struct {
		name string
		lang string
	}{
		{name: "empty string", lang: ""},
		{name: "unknown language", lang: "brainfuck"},
		{name: "similar to known", lang: "golang"},
		{name: "case sensitive", lang: "Go"},
		{name: "uppercase", lang: "PYTHON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod, err := reg.GetCompiledModule(ctx, tt.lang)
			require.Error(t, err)
			assert.Nil(t, mod)
			assert.True(t, errors.Is(err, ErrUnknownLanguage),
				"error should wrap ErrUnknownLanguage, got: %v", err)
			assert.Contains(t, err.Error(), tt.lang)
		})
	}
}

func TestGetCompiledModule_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() {
		// Use a fresh context for Close since the original is cancelled.
		_ = reg.Close(context.Background())
	}()

	// Cancel context before compilation.
	cancel()

	_, err = reg.GetCompiledModule(ctx, "go")
	// The behavior depends on whether wazero checks context during compilation.
	// With a cancelled context, it should either return an error or succeed
	// (if compilation was fast enough). We just verify it doesn't panic.
	if err != nil {
		t.Logf("GetCompiledModule with cancelled context returned error: %v", err)
	}
}

func TestSupportedLanguages(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	langs := reg.SupportedLanguages()

	assert.Len(t, langs, 8, "should have 8 supported languages")

	expected := []string{"c", "cpp", "go", "java", "javascript", "python", "rust", "typescript"}
	assert.Equal(t, expected, langs, "languages should be sorted alphabetically")

	// Verify the list is actually sorted.
	for i := 1; i < len(langs); i++ {
		assert.True(t, langs[i-1] < langs[i],
			"languages should be sorted: %s should come before %s", langs[i-1], langs[i])
	}
}

func TestHasLanguage(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	tests := []struct {
		name string
		lang string
		want bool
	}{
		{name: "go", lang: "go", want: true},
		{name: "typescript", lang: "typescript", want: true},
		{name: "javascript", lang: "javascript", want: true},
		{name: "python", lang: "python", want: true},
		{name: "rust", lang: "rust", want: true},
		{name: "java", lang: "java", want: true},
		{name: "c", lang: "c", want: true},
		{name: "cpp", lang: "cpp", want: true},
		{name: "unknown", lang: "ruby", want: false},
		{name: "empty", lang: "", want: false},
		{name: "case sensitive", lang: "Go", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.HasLanguage(tt.lang)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)

	// Compile a module before closing.
	_, err = reg.GetCompiledModule(ctx, "c")
	require.NoError(t, err)

	// Close should release all resources.
	err = reg.Close(ctx)
	require.NoError(t, err, "Close should succeed")

	// Compiled map should be empty after Close.
	assert.Empty(t, reg.compiled, "compiled map should be cleared after Close")
}

func TestClose_EmptyRegistry(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)

	// Close without compiling anything.
	err = reg.Close(ctx)
	require.NoError(t, err, "Close on empty registry should succeed")
}

func TestRuntime(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	rt := reg.Runtime()
	assert.NotNil(t, rt, "Runtime should return the underlying wazero runtime")
	assert.Equal(t, reg.runtime, rt, "Runtime should return the same runtime instance")
}

func TestGetCompiledModule_Concurrent(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	// Launch multiple goroutines requesting the same language concurrently.
	const goroutines = 10
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	mods := make([]interface{}, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			mod, err := reg.GetCompiledModule(ctx, "go")
			errs[idx] = err
			mods[idx] = mod
		}(i)
	}
	wg.Wait()

	// All goroutines should succeed.
	for i, err := range errs {
		require.NoError(t, err, "goroutine %d should not error", i)
		assert.NotNil(t, mods[i], "goroutine %d should get a compiled module", i)
	}
}

func TestGetCompiledModule_ConcurrentDifferentLanguages(t *testing.T) {
	ctx := context.Background()
	logger := testLogger(t)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(t, err)
	defer func() { require.NoError(t, reg.Close(ctx)) }()

	languages := reg.SupportedLanguages()

	var wg sync.WaitGroup
	wg.Add(len(languages))

	results := make(map[string]error)
	var mu sync.Mutex

	for _, lang := range languages {
		go func(l string) {
			defer wg.Done()
			_, err := reg.GetCompiledModule(ctx, l)
			mu.Lock()
			results[l] = err
			mu.Unlock()
		}(lang)
	}
	wg.Wait()

	for lang, err := range results {
		assert.NoError(t, err, "concurrent compilation of %s should succeed", lang)
	}
}