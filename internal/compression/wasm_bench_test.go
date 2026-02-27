package compression

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// benchLogger returns a silent logger for benchmarks.
func benchLogger(b *testing.B) *slog.Logger {
	b.Helper()
	return slog.New(slog.NewTextHandler(testLogWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

// BenchmarkGetCompiledModule_ColdStart measures the time to compile each grammar
// from scratch (no cache).
func BenchmarkGetCompiledModule_ColdStart(b *testing.B) {
	languages := []string{"c", "cpp", "go", "java", "javascript", "python", "rust", "typescript"}

	for _, lang := range languages {
		b.Run(lang, func(b *testing.B) {
			ctx := context.Background()
			logger := benchLogger(b)

			for i := 0; i < b.N; i++ {
				// Create a fresh registry each iteration to force cold compilation.
				reg, err := NewGrammarRegistry(ctx, logger)
				require.NoError(b, err)

				_, err = reg.GetCompiledModule(ctx, lang)
				require.NoError(b, err)

				require.NoError(b, reg.Close(ctx))
			}
		})
	}
}

// BenchmarkGetCompiledModule_WarmCache measures the time to retrieve a
// previously compiled grammar from the cache.
func BenchmarkGetCompiledModule_WarmCache(b *testing.B) {
	ctx := context.Background()
	logger := benchLogger(b)

	reg, err := NewGrammarRegistry(ctx, logger)
	require.NoError(b, err)
	defer func() { require.NoError(b, reg.Close(ctx)) }()

	// Pre-compile the module.
	_, err = reg.GetCompiledModule(ctx, "go")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reg.GetCompiledModule(ctx, "go")
		require.NoError(b, err)
	}
}

// BenchmarkCompileAllGrammars measures the total time to compile all 8 grammars
// from scratch.
func BenchmarkCompileAllGrammars(b *testing.B) {
	ctx := context.Background()
	logger := benchLogger(b)
	languages := []string{"c", "cpp", "go", "java", "javascript", "python", "rust", "typescript"}

	for i := 0; i < b.N; i++ {
		reg, err := NewGrammarRegistry(ctx, logger)
		require.NoError(b, err)

		for _, lang := range languages {
			_, err = reg.GetCompiledModule(ctx, lang)
			require.NoError(b, err)
		}

		require.NoError(b, reg.Close(ctx))
	}
}