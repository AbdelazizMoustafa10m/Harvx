package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatThousands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "0"},
		{name: "single digit", n: 5, want: "5"},
		{name: "hundreds", n: 420, want: "420"},
		{name: "thousands", n: 1234, want: "1,234"},
		{name: "ten thousands", n: 89420, want: "89,420"},
		{name: "hundred thousands", n: 200000, want: "200,000"},
		{name: "millions", n: 1234567, want: "1,234,567"},
		{name: "negative", n: -1234, want: "-1,234"},
		{name: "negative millions", n: -1234567, want: "-1,234,567"},
		{name: "one", n: 1, want: "1"},
		{name: "exact thousand", n: 1000, want: "1,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatThousands(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes int
		want  string
	}{
		{name: "zero", bytes: 0, want: "0 B"},
		{name: "bytes", bytes: 512, want: "512 B"},
		{name: "kilobytes", bytes: 2048, want: "2.0 KB"},
		{name: "large kilobytes", bytes: 102400, want: "100.0 KB"},
		{name: "megabytes", bytes: 1048576, want: "1.0 MB"},
		{name: "large megabytes", bytes: 2516582, want: "2.4 MB"},
		{name: "negative", bytes: -1, want: "0 B"},
		{name: "just under 1KB", bytes: 1023, want: "1023 B"},
		{name: "exactly 1KB", bytes: 1024, want: "1.0 KB"},
		{name: "just under 1MB", bytes: 1048575, want: "1024.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatSize(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEstimateOutputSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tokens int
		want   int
	}{
		{name: "zero", tokens: 0, want: 0},
		{name: "small", tokens: 100, want: 400},
		{name: "medium", tokens: 89420, want: 357680},
		{name: "large", tokens: 200000, want: 800000},
		{name: "negative", tokens: -5, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EstimateOutputSize(tt.tokens)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPercentage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{name: "zero", pct: 0, want: "0%"},
		{name: "half", pct: 50.0, want: "50%"},
		{name: "full", pct: 100.0, want: "100%"},
		{name: "over 100", pct: 150.0, want: "100%"},
		{name: "negative", pct: -10.0, want: "0%"},
		{name: "fraction", pct: 45.3, want: "45%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatPercentage(tt.pct)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		s        string
		maxWidth int
		want     string
	}{
		{name: "fits", s: "hello", maxWidth: 10, want: "hello"},
		{name: "exact fit", s: "hello", maxWidth: 5, want: "hello"},
		{name: "truncated", s: "hello world", maxWidth: 8, want: "hello..."},
		{name: "very short max", s: "hello", maxWidth: 3, want: "hel"},
		{name: "zero width", s: "hello", maxWidth: 0, want: ""},
		{name: "negative width", s: "hello", maxWidth: -1, want: ""},
		{name: "empty string", s: "", maxWidth: 10, want: ""},
		{name: "unicode CJK", s: "\u4f60\u597d\u4e16\u754c\u5417", maxWidth: 3, want: "\u4f60\u597d\u4e16"},
		{name: "unicode truncated with ellipsis", s: "\u4f60\u597d\u4e16\u754c\u5417", maxWidth: 4, want: "\u4f60..."},
		{name: "width of 1", s: "hello", maxWidth: 1, want: "h"},
		{name: "width of 4 exact ellipsis boundary", s: "hello world", maxWidth: 4, want: "h..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Truncate(tt.s, tt.maxWidth)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatThousands_LargeNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "10 million", n: 10000000, want: "10,000,000"},
		{name: "100 million", n: 100000000, want: "100,000,000"},
		{name: "1 billion", n: 1000000000, want: "1,000,000,000"},
		{name: "max typical tokens", n: 2000000, want: "2,000,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatThousands(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEstimateOutputSize_Integration(t *testing.T) {
	t.Parallel()

	// Verify the full pipeline: tokens -> estimated size -> formatted string.
	tests := []struct {
		name     string
		tokens   int
		wantSize string
	}{
		{name: "small file", tokens: 100, wantSize: "400 B"},
		{name: "medium file", tokens: 10000, wantSize: "39.1 KB"},
		{name: "large file", tokens: 600000, wantSize: "2.3 MB"},
		{name: "zero tokens", tokens: 0, wantSize: "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sizeBytes := EstimateOutputSize(tt.tokens)
			formatted := FormatSize(sizeBytes)
			assert.Equal(t, tt.wantSize, formatted)
		})
	}
}

func TestFormatPercentage_Precision(t *testing.T) {
	t.Parallel()

	// FormatPercentage uses %.0f, so it rounds.
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{name: "44.71 rounds to 45", pct: 44.71, want: "45%"},
		{name: "44.49 rounds to 44", pct: 44.49, want: "44%"},
		{name: "99.5 rounds to 100", pct: 99.5, want: "100%"},
		{name: "0.4 rounds to 0", pct: 0.4, want: "0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatPercentage(tt.pct)
			assert.Equal(t, tt.want, got)
		})
	}
}
