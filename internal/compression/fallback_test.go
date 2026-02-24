package compression

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// FallbackCompressor metadata tests
// ---------------------------------------------------------------------------

func TestFallbackCompressor_Language(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()
	assert.Equal(t, "fallback", c.Language())
}

func TestFallbackCompressor_SupportedNodeTypes(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()
	types := c.SupportedNodeTypes()
	assert.Equal(t, []string{"raw_content"}, types)
}

// ---------------------------------------------------------------------------
// Empty / nil input
// ---------------------------------------------------------------------------

func TestFallbackCompressor_EmptyInput(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)

	assert.Equal(t, "unknown", output.Language)
	assert.Equal(t, 0, output.NodeCount)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, 0, output.OutputSize)
	assert.Empty(t, output.Signatures)
}

func TestFallbackCompressor_NilInput(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	output, err := c.Compress(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, "unknown", output.Language)
	assert.Equal(t, 0, output.NodeCount)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, 0, output.OutputSize)
	assert.Empty(t, output.Signatures)
}

// ---------------------------------------------------------------------------
// Normal content (non-empty)
// ---------------------------------------------------------------------------

func TestFallbackCompressor_NormalContent(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple text",
			content: "Hello, world!",
		},
		{
			name:    "multi-line text",
			content: "line 1\nline 2\nline 3\n",
		},
		{
			name:    "go source code",
			content: "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n",
		},
		{
			name:    "single newline",
			content: "\n",
		},
		{
			name:    "unicode content",
			content: "Hello \u4e16\u754c\n\u00e9\u00e8\u00ea\n",
		},
		{
			name:    "whitespace only",
			content: "   \t\n  \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.content))
			require.NoError(t, err)

			// NodeCount is 1 for non-empty.
			assert.Equal(t, 1, output.NodeCount)
			// Language is "unknown".
			assert.Equal(t, "unknown", output.Language)
			// Single signature with full content.
			require.Len(t, output.Signatures, 1)
			sig := output.Signatures[0]
			assert.Equal(t, KindDocComment, sig.Kind)
			assert.Equal(t, "", sig.Name)
			assert.Equal(t, tt.content, sig.Source, "content should be returned unchanged")
			assert.Equal(t, 1, sig.StartLine)
			assert.Greater(t, sig.EndLine, 0)
		})
	}
}

// ---------------------------------------------------------------------------
// OriginalSize == OutputSize (no compression)
// ---------------------------------------------------------------------------

func TestFallbackCompressor_OriginalSizeEqualsOutput(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	tests := []struct {
		name    string
		content string
	}{
		{name: "short", content: "abc"},
		{name: "medium", content: strings.Repeat("hello world\n", 100)},
		{name: "single byte", content: "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.content))
			require.NoError(t, err)

			assert.Equal(t, len(tt.content), output.OriginalSize)
			assert.Equal(t, len(tt.content), output.OutputSize)
			assert.Equal(t, output.OriginalSize, output.OutputSize,
				"fallback compressor should not change size")
		})
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestFallbackCompressor_ContextCancellation(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	t.Run("already cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Compress(ctx, []byte("some content"))
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()

		_, err := c.Compress(ctx, []byte("some content"))
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("cancelled with empty input", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Compress(ctx, []byte{})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("cancelled with nil input", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Compress(ctx, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// ---------------------------------------------------------------------------
// Binary content
// ---------------------------------------------------------------------------

func TestFallbackCompressor_BinaryContent(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x89, 0x50, 0x4E, 0x47}
	output, err := c.Compress(context.Background(), binaryData)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	// Binary content should be returned unchanged.
	assert.Equal(t, string(binaryData), output.Signatures[0].Source)
	assert.Equal(t, len(binaryData), output.OriginalSize)
	assert.Equal(t, len(binaryData), output.OutputSize)
	assert.Equal(t, 1, output.NodeCount)
	assert.Equal(t, "unknown", output.Language)
}

// ---------------------------------------------------------------------------
// IsFallback function
// ---------------------------------------------------------------------------

func TestIsFallback_True(t *testing.T) {
	t.Parallel()

	// Output from the fallback compressor has Language="unknown".
	co := &CompressedOutput{
		Language:     "unknown",
		OriginalSize: 10,
		OutputSize:   10,
		NodeCount:    1,
	}
	assert.True(t, IsFallback(co))
}

func TestIsFallback_False(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language string
	}{
		{name: "go", language: "go"},
		{name: "python", language: "python"},
		{name: "toml", language: "toml"},
		{name: "rust", language: "rust"},
		{name: "empty string", language: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			co := &CompressedOutput{
				Language: tt.language,
			}
			assert.False(t, IsFallback(co))
		})
	}
}

func TestIsFallback_Nil(t *testing.T) {
	t.Parallel()
	assert.False(t, IsFallback(nil))
}

// ---------------------------------------------------------------------------
// Integration: IsFallback with actual Compress output
// ---------------------------------------------------------------------------

func TestIsFallback_Integration(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	t.Run("non-empty input is fallback", func(t *testing.T) {
		t.Parallel()
		output, err := c.Compress(context.Background(), []byte("content"))
		require.NoError(t, err)
		assert.True(t, IsFallback(output))
	})

	t.Run("empty input is fallback", func(t *testing.T) {
		t.Parallel()
		output, err := c.Compress(context.Background(), []byte{})
		require.NoError(t, err)
		assert.True(t, IsFallback(output))
	})
}

// ---------------------------------------------------------------------------
// CompressionRatio for fallback
// ---------------------------------------------------------------------------

func TestFallbackCompressor_CompressionRatio(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	t.Run("non-empty content has ratio 1.0", func(t *testing.T) {
		t.Parallel()
		output, err := c.Compress(context.Background(), []byte("some content"))
		require.NoError(t, err)
		assert.InDelta(t, 1.0, output.CompressionRatio(), 0.001,
			"fallback compressor should have compression ratio of 1.0")
	})

	t.Run("empty content has ratio 0.0", func(t *testing.T) {
		t.Parallel()
		output, err := c.Compress(context.Background(), []byte{})
		require.NoError(t, err)
		assert.InDelta(t, 0.0, output.CompressionRatio(), 0.001,
			"empty input should have compression ratio of 0.0")
	})
}

// ---------------------------------------------------------------------------
// EndLine calculation
// ---------------------------------------------------------------------------

func TestFallbackCompressor_EndLineCalculation(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	tests := []struct {
		name        string
		content     string
		wantEndLine int
	}{
		{name: "single line", content: "hello", wantEndLine: 1},
		{name: "two lines", content: "line1\nline2", wantEndLine: 2},
		{name: "three lines trailing newline", content: "a\nb\nc\n", wantEndLine: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.content))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)
			assert.Equal(t, tt.wantEndLine, output.Signatures[0].EndLine)
		})
	}
}

// ---------------------------------------------------------------------------
// Stateless / concurrent safety
// ---------------------------------------------------------------------------

func TestFallbackCompressor_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	c := NewFallbackCompressor()

	// Run multiple goroutines concurrently against the same compressor.
	// Use assert (not require) to avoid calling FailNow from non-test goroutines.
	const goroutines = 50
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			content := strings.Repeat("x", n*10+1)
			output, err := c.Compress(context.Background(), []byte(content))
			if err != nil {
				errs <- err
				return
			}
			if len(output.Signatures) != 1 {
				errs <- fmt.Errorf("expected 1 signature, got %d", len(output.Signatures))
				return
			}
			if output.Signatures[0].Source != content {
				errs <- fmt.Errorf("content mismatch for goroutine %d", n)
				return
			}
			errs <- nil
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		err := <-errs
		assert.NoError(t, err)
	}
}