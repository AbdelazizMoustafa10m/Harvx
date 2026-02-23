package tokenizer_test

import (
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEstimator_Name verifies that the estimator reports its name as "none".
func TestEstimator_Name(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)
	assert.Equal(t, "none", tok.Name())
}

// TestEstimator_Empty verifies Count("") returns 0.
func TestEstimator_Empty(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)
	assert.Equal(t, 0, tok.Count(""))
}

// TestEstimator_HelloWorld verifies the len/4 heuristic for a known string.
// "hello world" has 11 bytes -> 11/4 = 2 (integer division).
func TestEstimator_HelloWorld(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)
	// len("hello world") == 11; 11/4 == 2
	assert.Equal(t, 2, tok.Count("hello world"))
}

// TestEstimator_LenDivFour is a table-driven test that exhaustively verifies
// the len(text)/4 formula for various input lengths.
func TestEstimator_LenDivFour(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 0},
		{"1 char", "a", 0},             // 1/4 = 0
		{"3 chars", "abc", 0},          // 3/4 = 0
		{"4 chars", "abcd", 1},         // 4/4 = 1
		{"5 chars", "abcde", 1},        // 5/4 = 1
		{"8 chars", "abcdefgh", 2},     // 8/4 = 2
		{"11 chars", "hello world", 2}, // 11/4 = 2
		{"12 chars", "hello world!", 3}, // 12/4 = 3
		{"40 chars", strings.Repeat("a", 40), 10},
		{"100 chars", strings.Repeat("x", 100), 25},
		{"1024 chars", strings.Repeat("y", 1024), 256},
	}

	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tok.Count(tt.text)
			assert.Equal(t, tt.want, got,
				"Count(%q) = %d, want %d", tt.text, got, tt.want)
		})
	}
}

// TestEstimator_LargeText verifies that the estimator handles large inputs
// correctly and consistently.
func TestEstimator_LargeText(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("The quick brown fox. ", 500) // ~10KB
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)

	want := len(text) / 4
	got := tok.Count(text)
	assert.Equal(t, want, got)
}

// TestEstimator_ConsistentResults verifies that calling Count multiple times
// on the same text always returns the same value.
func TestEstimator_ConsistentResults(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)

	text := "Consistency check text."
	expected := tok.Count(text)
	for i := range 10 {
		got := tok.Count(text)
		assert.Equal(t, expected, got, "call %d returned different result", i)
	}
}

// TestEstimator_UnicodeByteCounting verifies that the estimator counts bytes,
// not Unicode runes. Multi-byte characters (e.g., Japanese, emoji) consume
// more bytes than their rune count, so the estimator reflects byte length.
//
// This documents the intentional semantics: the heuristic is based on the
// byte length of the string, which for Latin text is similar to char count
// but differs for multi-byte encodings.
func TestEstimator_UnicodeByteCounting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		text     string
		wantExpr string // description of the expected value
		wantFunc func(string) int
	}{
		{
			name:     "japanese kana (3 bytes each)",
			text:     "こんにちは", // 5 runes * 3 bytes = 15 bytes; 15/4 = 3
			wantExpr: "len(text)/4 = 15/4 = 3",
			wantFunc: func(s string) int { return len(s) / 4 },
		},
		{
			name:     "emoji (4 bytes each)",
			text:     "🚀🌍🎉", // 3 runes * 4 bytes = 12 bytes; 12/4 = 3
			wantExpr: "len(text)/4 = 12/4 = 3",
			wantFunc: func(s string) int { return len(s) / 4 },
		},
		{
			name:     "mixed ascii and multibyte",
			text:     "Hi 世界", // "Hi " = 3 bytes, "世" = 3, "界" = 3 -> 9 bytes; 9/4 = 2
			wantExpr: "len(text)/4 = 9/4 = 2",
			wantFunc: func(s string) int { return len(s) / 4 },
		},
	}

	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			want := tt.wantFunc(tt.text)
			got := tok.Count(tt.text)
			assert.Equal(t, want, got,
				"Count(%q): expected %s = %d, got %d", tt.text, tt.wantExpr, want, got)
		})
	}
}

// BenchmarkEstimator_1KB benchmarks the estimator on ~1KB of text.
func BenchmarkEstimator_1KB(b *testing.B) {
	text := strings.Repeat("x", 1024)
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkEstimator_10KB benchmarks the estimator on ~10KB of text.
func BenchmarkEstimator_10KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 500)
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}

// BenchmarkEstimator_100KB benchmarks the estimator on ~100KB of text.
func BenchmarkEstimator_100KB(b *testing.B) {
	text := strings.Repeat("The quick brown fox. ", 5000)
	tok, err := tokenizer.NewTokenizer("none")
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		tok.Count(text)
	}
}
