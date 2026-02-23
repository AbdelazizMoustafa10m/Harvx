package tokenizer_test

import (
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCL100K_HelloWorld verifies that "hello world" tokenises to 2 tokens
// under cl100k_base encoding.
func TestCL100K_HelloWorld(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(t, err)
	// "hello" and " world" are each a single BPE token in cl100k_base.
	assert.Equal(t, 2, tok.Count("hello world"))
}

// TestO200K_HelloWorld verifies that "hello world" tokenises to 2 tokens
// under o200k_base encoding.
//
// o200k_base is the GPT-4o/o1 vocabulary. Like cl100k_base, it encodes
// "hello" and " world" as individual tokens, so the expected count is 2.
func TestO200K_HelloWorld(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(t, err)
	// "hello world" is 2 BPE tokens in o200k_base (same as cl100k_base).
	assert.Equal(t, 2, tok.Count("hello world"))
}

// TestCL100K_Unicode verifies that multi-byte Unicode characters are handled
// without panicking and return a positive token count.
func TestCL100K_Unicode(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"japanese", "こんにちは世界"},
		{"arabic", "مرحبا بالعالم"},
		{"emoji", "Hello 🌍 World 🚀"},
		{"mixed", "Héllo Wörld"},
	}

	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			count := tok.Count(tt.text)
			assert.Greater(t, count, 0, "expected positive token count for %q", tt.text)
		})
	}
}

// TestO200K_Unicode mirrors TestCL100K_Unicode for o200k_base.
func TestO200K_Unicode(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"japanese", "こんにちは世界"},
		{"arabic", "مرحبا بالعالم"},
		{"emoji", "Hello 🌍 World 🚀"},
		{"mixed", "Héllo Wörld"},
	}

	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			count := tok.Count(tt.text)
			assert.Greater(t, count, 0, "expected positive token count for %q", tt.text)
		})
	}
}

// TestCL100K_LargeText verifies that cl100k_base handles a large (~10KB) text
// without errors and returns a reasonable token count.
func TestCL100K_LargeText(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 200) // ~9KB
	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(t, err)

	count := tok.Count(text)
	// 9 tokens per sentence * 200 repetitions => roughly 1800 tokens.
	// Use a wide range to be resilient to exact BPE encoding changes.
	assert.Greater(t, count, 100, "expected >100 tokens for 10KB text")
	assert.Less(t, count, 10000, "expected <10000 tokens for 10KB text")
}

// TestO200K_LargeText mirrors TestCL100K_LargeText for o200k_base.
func TestO200K_LargeText(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 200)
	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(t, err)

	count := tok.Count(text)
	assert.Greater(t, count, 100)
	assert.Less(t, count, 10000)
}

// TestTiktoken_MoreTokensThanSingleChar verifies that a sentence produces more
// tokens than a single character.
func TestTiktoken_MoreTokensThanSingleChar(t *testing.T) {
	names := []string{"cl100k_base", "o200k_base"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)

			single := tok.Count("a")
			sentence := tok.Count("The quick brown fox jumps over the lazy dog.")
			assert.Greater(t, sentence, single,
				"sentence should have more tokens than a single char")
		})
	}
}

// TestTiktoken_CountMonotonicallyIncreases verifies that a longer text always
// produces a count greater than or equal to a shorter prefix of the same text.
// This is a basic sanity property of any reasonable tokenizer.
func TestTiktoken_CountMonotonicallyIncreases(t *testing.T) {
	names := []string{"cl100k_base", "o200k_base"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)

			short := tok.Count("Hello")
			long := tok.Count("Hello, my name is Harvx and I process codebases for LLMs.")
			assert.GreaterOrEqual(t, long, short,
				"longer text must produce >= tokens than a shorter prefix")
		})
	}
}

// TestTiktoken_GoCodeTokenCount verifies that typical Go source code is
// tokenized with a plausible token density (not wildly over- or under-counted).
func TestTiktoken_GoCodeTokenCount(t *testing.T) {
	goCode := `package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
`
	names := []string{"cl100k_base", "o200k_base"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)

			count := tok.Count(goCode)
			// The snippet is ~60 chars. BPE typically yields ~15-30 tokens for this.
			assert.Greater(t, count, 5, "expected >5 tokens for Go code snippet")
			assert.Less(t, count, 60, "expected <60 tokens for Go code snippet (not char-counted)")
		})
	}
}

// BenchmarkCL100K_1KB benchmarks cl100k_base on ~1KB of text.
func BenchmarkCL100K_1KB(b *testing.B) {
	text := strings.Repeat("x", 1024)
	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkCL100K_10KB benchmarks cl100k_base on ~10KB of text.
func BenchmarkCL100K_10KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 500) // ~10KB
	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkCL100K_100KB benchmarks cl100k_base on ~100KB of text.
func BenchmarkCL100K_100KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 5000) // ~100KB
	tok, err := tokenizer.NewTokenizer("cl100k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkO200K_1KB benchmarks o200k_base on ~1KB of text.
func BenchmarkO200K_1KB(b *testing.B) {
	text := strings.Repeat("x", 1024)
	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkO200K_10KB benchmarks o200k_base on ~10KB of text.
func BenchmarkO200K_10KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 500)
	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkO200K_100KB benchmarks o200k_base on ~100KB of text.
func BenchmarkO200K_100KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 5000)
	tok, err := tokenizer.NewTokenizer("o200k_base")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}
