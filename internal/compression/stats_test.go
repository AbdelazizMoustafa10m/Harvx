package compression

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Atomic increment safety
// ---------------------------------------------------------------------------

func TestCompressionStatsAtomicIncrements(t *testing.T) {
	stats := &CompressionStats{}

	const goroutines = 50
	const incrementsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 4) // 4 counter types

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				stats.addCompressed()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				stats.addFailed()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				stats.addSkipped()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				stats.addTimedOut()
			}
		}()
	}

	wg.Wait()

	expected := int64(goroutines * incrementsPerGoroutine)
	assert.Equal(t, expected, atomic.LoadInt64(&stats.FilesCompressed), "FilesCompressed mismatch")
	assert.Equal(t, expected, atomic.LoadInt64(&stats.FilesFailed), "FilesFailed mismatch")
	assert.Equal(t, expected, atomic.LoadInt64(&stats.FilesSkipped), "FilesSkipped mismatch")
	assert.Equal(t, expected, atomic.LoadInt64(&stats.FilesTimedOut), "FilesTimedOut mismatch")
	assert.Equal(t, expected*4, stats.TotalFiles(), "TotalFiles mismatch")
}

// ---------------------------------------------------------------------------
// TotalFiles
// ---------------------------------------------------------------------------

func TestCompressionStatsTotalFiles(t *testing.T) {
	tests := []struct {
		name       string
		compressed int
		failed     int
		skipped    int
		timedOut   int
		wantTotal  int64
	}{
		{
			name:      "all zero",
			wantTotal: 0,
		},
		{
			name:       "compressed only",
			compressed: 5,
			wantTotal:  5,
		},
		{
			name:      "failed only",
			failed:    3,
			wantTotal: 3,
		},
		{
			name:      "skipped only",
			skipped:   7,
			wantTotal: 7,
		},
		{
			name:      "timed out only",
			timedOut:  2,
			wantTotal: 2,
		},
		{
			name:       "all categories",
			compressed: 10,
			failed:     2,
			skipped:    5,
			timedOut:   1,
			wantTotal:  18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &CompressionStats{
				FilesCompressed: int64(tt.compressed),
				FilesFailed:     int64(tt.failed),
				FilesSkipped:    int64(tt.skipped),
				FilesTimedOut:   int64(tt.timedOut),
			}
			assert.Equal(t, tt.wantTotal, stats.TotalFiles())
		})
	}
}

// ---------------------------------------------------------------------------
// TokenSavings
// ---------------------------------------------------------------------------

func TestCompressionStatsTokenSavings(t *testing.T) {
	tests := []struct {
		name        string
		original    int64
		compressed  int64
		wantSavings int64
	}{
		{
			name:        "typical savings",
			original:    1000,
			compressed:  400,
			wantSavings: 600,
		},
		{
			name:        "no savings",
			original:    500,
			compressed:  500,
			wantSavings: 0,
		},
		{
			name:        "zero original",
			original:    0,
			compressed:  0,
			wantSavings: 0,
		},
		{
			name:        "large values",
			original:    1000000,
			compressed:  250000,
			wantSavings: 750000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &CompressionStats{
				OriginalTokens:   tt.original,
				CompressedTokens: tt.compressed,
			}
			assert.Equal(t, tt.wantSavings, stats.TokenSavings())
		})
	}
}

// ---------------------------------------------------------------------------
// AverageRatio
// ---------------------------------------------------------------------------

func TestCompressionStatsAverageRatio(t *testing.T) {
	tests := []struct {
		name       string
		original   int64
		compressed int64
		wantRatio  float64
	}{
		{
			name:      "zero original returns 1.0",
			original:  0,
			wantRatio: 1.0,
		},
		{
			name:       "no compression",
			original:   1000,
			compressed: 1000,
			wantRatio:  1.0,
		},
		{
			name:       "50 percent compression",
			original:   1000,
			compressed: 500,
			wantRatio:  0.5,
		},
		{
			name:       "typical compression",
			original:   1000,
			compressed: 400,
			wantRatio:  0.4,
		},
		{
			name:       "full compression (zero output)",
			original:   1000,
			compressed: 0,
			wantRatio:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &CompressionStats{
				OriginalTokens:   tt.original,
				CompressedTokens: tt.compressed,
			}
			assert.InDelta(t, tt.wantRatio, stats.AverageRatio(), 0.001)
		})
	}
}

// ---------------------------------------------------------------------------
// String
// ---------------------------------------------------------------------------

func TestCompressionStatsString(t *testing.T) {
	stats := &CompressionStats{
		FilesCompressed:  10,
		FilesFailed:      2,
		FilesSkipped:     5,
		FilesTimedOut:    1,
		OriginalTokens:   10000,
		CompressedTokens: 4000,
	}

	s := stats.String()

	assert.Contains(t, s, "10 compressed")
	assert.Contains(t, s, "2 failed")
	assert.Contains(t, s, "5 skipped")
	assert.Contains(t, s, "1 timed out")
	assert.Contains(t, s, "10000")
	assert.Contains(t, s, "4000")
	assert.Contains(t, s, "saved 6000")
	assert.Contains(t, s, "ratio 0.40")
	assert.True(t, strings.HasPrefix(s, "Compression:"), "expected prefix 'Compression:'")
}

func TestCompressionStatsStringZero(t *testing.T) {
	stats := &CompressionStats{}

	s := stats.String()

	assert.Contains(t, s, "0 compressed")
	assert.Contains(t, s, "0 failed")
	assert.Contains(t, s, "0 skipped")
	assert.Contains(t, s, "0 timed out")
	assert.Contains(t, s, "ratio 1.00")
}

// ---------------------------------------------------------------------------
// Zero values
// ---------------------------------------------------------------------------

func TestCompressionStatsZeroValues(t *testing.T) {
	stats := &CompressionStats{}

	assert.Equal(t, int64(0), stats.TotalFiles())
	assert.Equal(t, int64(0), stats.TokenSavings())
	assert.InDelta(t, 1.0, stats.AverageRatio(), 0.001)
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesCompressed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesFailed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesSkipped))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesTimedOut))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.OriginalTokens))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.CompressedTokens))
}

// ---------------------------------------------------------------------------
// addTokens concurrent safety
// ---------------------------------------------------------------------------

func TestCompressionStatsAddTokensConcurrent(t *testing.T) {
	stats := &CompressionStats{}

	const goroutines = 50
	const tokensPerCall = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			stats.addTokens(tokensPerCall, tokensPerCall/2)
		}()
	}

	wg.Wait()

	expectedOrig := int64(goroutines * tokensPerCall)
	expectedComp := int64(goroutines * tokensPerCall / 2)
	assert.Equal(t, expectedOrig, atomic.LoadInt64(&stats.OriginalTokens))
	assert.Equal(t, expectedComp, atomic.LoadInt64(&stats.CompressedTokens))
	assert.Equal(t, expectedOrig-expectedComp, stats.TokenSavings())
}
