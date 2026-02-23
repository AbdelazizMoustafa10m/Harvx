package tokenizer_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/harvx/harvx/internal/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTokenizer_SupportedNames verifies that all documented encoding names
// produce a valid Tokenizer with the correct Name() return value.
func TestNewTokenizer_SupportedNames(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
	}{
		{input: "cl100k_base", wantName: "cl100k_base"},
		{input: "o200k_base", wantName: "o200k_base"},
		{input: "none", wantName: "none"},
		{input: "", wantName: "cl100k_base"}, // empty string -> default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(tt.input)
			require.NoError(t, err)
			require.NotNil(t, tok)
			assert.Equal(t, tt.wantName, tok.Name())
		})
	}
}

// TestNewTokenizer_UnknownName verifies that an unrecognised encoding name
// returns ErrUnknownTokenizer.
func TestNewTokenizer_UnknownName(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"invalid"},
		{"gpt2"},
		{"p50k_base"},
		{"CL100K_BASE"}, // case-sensitive
		{"cl100k"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(tt.name)
			assert.Nil(t, tok)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tokenizer.ErrUnknownTokenizer),
				"expected ErrUnknownTokenizer, got: %v", err)
		})
	}
}

// TestNewTokenizer_InterfaceCompliance verifies that NewTokenizer returns a
// value that satisfies the Tokenizer interface.
func TestNewTokenizer_InterfaceCompliance(t *testing.T) {
	names := []string{"cl100k_base", "o200k_base", "none"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)

			// Compile-time check: Tokenizer interface satisfied.
			var _ tokenizer.Tokenizer = tok
			assert.NotEmpty(t, tok.Name())
		})
	}
}

// TestTokenizer_CountEmpty verifies that Count("") returns 0 for all
// implementations.
func TestTokenizer_CountEmpty(t *testing.T) {
	names := []string{"cl100k_base", "o200k_base", "none"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)
			assert.Equal(t, 0, tok.Count(""))
		})
	}
}

// TestTokenizer_CountNonNegative verifies that Count never returns a negative
// value for any implementation.
func TestTokenizer_CountNonNegative(t *testing.T) {
	texts := []string{"", "a", "hello world", "こんにちは"}
	names := []string{"cl100k_base", "o200k_base", "none"}

	for _, name := range names {
		tok, err := tokenizer.NewTokenizer(name)
		require.NoError(t, err)

		for _, text := range texts {
			tok := tok   // capture loop variable
			text := text // capture loop variable
			t.Run(name+"/"+text, func(t *testing.T) {
				t.Parallel()
				count := tok.Count(text)
				assert.GreaterOrEqual(t, count, 0,
					"Count(%q) returned negative value %d for %s", text, count, name)
			})
		}
	}
}

// TestTokenizer_ConcurrentSafety verifies that all Tokenizer implementations
// are safe to use from multiple goroutines simultaneously.
func TestTokenizer_ConcurrentSafety(t *testing.T) {
	const goroutines = 10
	const iters = 50
	text := "hello world, this is a concurrent test"

	names := []string{"cl100k_base", "o200k_base", "none"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			tok, err := tokenizer.NewTokenizer(name)
			require.NoError(t, err)

			var wg sync.WaitGroup
			results := make([]int, goroutines)

			for i := range goroutines {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					for range iters {
						results[idx] = tok.Count(text)
					}
				}(i)
			}
			wg.Wait()

			// All goroutines should have seen the same count.
			expected := results[0]
			for i, r := range results {
				assert.Equal(t, expected, r, "goroutine %d got different result", i)
			}
		})
	}
}

// TestTokenizer_NameConstants verifies exported name constants match
// NewTokenizer behaviour.
func TestTokenizer_NameConstants(t *testing.T) {
	pairs := []struct {
		constant string
	}{
		{tokenizer.NameCL100K},
		{tokenizer.NameO200K},
		{tokenizer.NameNone},
	}

	for _, p := range pairs {
		t.Run(p.constant, func(t *testing.T) {
			t.Parallel()
			tok, err := tokenizer.NewTokenizer(p.constant)
			require.NoError(t, err)
			assert.Equal(t, p.constant, tok.Name())
		})
	}
}

// TestNewTokenizer_EmptyStringIsDefaultCL100K explicitly verifies that the
// empty string input selects cl100k_base as the default, per spec requirement.
func TestNewTokenizer_EmptyStringIsDefaultCL100K(t *testing.T) {
	t.Parallel()
	tok, err := tokenizer.NewTokenizer("")
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.Equal(t, tokenizer.NameCL100K, tok.Name(),
		"empty string must select cl100k_base as the default tokenizer")
	// Confirm it counts tokens like a real BPE tokenizer, not an estimator.
	// "hello world" is 2 BPE tokens in cl100k_base.
	assert.Equal(t, 2, tok.Count("hello world"))
}

// TestTokenizer_ErrUnknownTokenizer_Wrapping verifies ErrUnknownTokenizer is
// accessible via errors.Is for programmatic error inspection.
func TestTokenizer_ErrUnknownTokenizer_Wrapping(t *testing.T) {
	t.Parallel()
	_, err := tokenizer.NewTokenizer("totally-unknown-encoding")
	require.Error(t, err)
	assert.True(t, errors.Is(err, tokenizer.ErrUnknownTokenizer))
	// The error message should include the unsupported name for diagnostics.
	assert.Contains(t, err.Error(), "totally-unknown-encoding")
}
