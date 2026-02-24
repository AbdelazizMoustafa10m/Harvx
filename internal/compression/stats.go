package compression

import (
	"fmt"
	"sync/atomic"
	"time"
)

// CompressionStats holds aggregate statistics for a compression run.
// All counter fields are safe for concurrent updates via atomic operations.
type CompressionStats struct {
	FilesCompressed  int64
	FilesFailed      int64
	FilesSkipped     int64 // Unsupported language
	FilesTimedOut    int64
	OriginalTokens   int64
	CompressedTokens int64
	TotalDuration    time.Duration
}

// addCompressed increments the compressed file counter atomically.
func (s *CompressionStats) addCompressed() {
	atomic.AddInt64(&s.FilesCompressed, 1)
}

// addFailed increments the failed file counter atomically.
func (s *CompressionStats) addFailed() {
	atomic.AddInt64(&s.FilesFailed, 1)
}

// addSkipped increments the skipped file counter atomically.
func (s *CompressionStats) addSkipped() {
	atomic.AddInt64(&s.FilesSkipped, 1)
}

// addTimedOut increments the timed-out file counter atomically.
func (s *CompressionStats) addTimedOut() {
	atomic.AddInt64(&s.FilesTimedOut, 1)
}

// addTokens adds original and compressed token counts atomically.
func (s *CompressionStats) addTokens(original, compressed int) {
	atomic.AddInt64(&s.OriginalTokens, int64(original))
	atomic.AddInt64(&s.CompressedTokens, int64(compressed))
}

// TotalFiles returns the sum of all processed file categories.
func (s *CompressionStats) TotalFiles() int64 {
	return atomic.LoadInt64(&s.FilesCompressed) +
		atomic.LoadInt64(&s.FilesFailed) +
		atomic.LoadInt64(&s.FilesSkipped) +
		atomic.LoadInt64(&s.FilesTimedOut)
}

// TokenSavings returns the number of tokens saved by compression.
func (s *CompressionStats) TokenSavings() int64 {
	return atomic.LoadInt64(&s.OriginalTokens) - atomic.LoadInt64(&s.CompressedTokens)
}

// AverageRatio returns the average compression ratio across compressed files.
// Returns 1.0 if no tokens were processed or no files were compressed.
func (s *CompressionStats) AverageRatio() float64 {
	orig := atomic.LoadInt64(&s.OriginalTokens)
	if orig == 0 {
		return 1.0
	}
	comp := atomic.LoadInt64(&s.CompressedTokens)
	return float64(comp) / float64(orig)
}

// String returns a human-readable summary of compression statistics.
func (s *CompressionStats) String() string {
	compressed := atomic.LoadInt64(&s.FilesCompressed)
	failed := atomic.LoadInt64(&s.FilesFailed)
	skipped := atomic.LoadInt64(&s.FilesSkipped)
	timedOut := atomic.LoadInt64(&s.FilesTimedOut)
	origTokens := atomic.LoadInt64(&s.OriginalTokens)
	compTokens := atomic.LoadInt64(&s.CompressedTokens)

	return fmt.Sprintf(
		"Compression: %d compressed, %d failed, %d skipped, %d timed out (tokens: %d → %d, saved %d, ratio %.2f)",
		compressed, failed, skipped, timedOut,
		origTokens, compTokens,
		origTokens-compTokens,
		s.AverageRatio(),
	)
}
