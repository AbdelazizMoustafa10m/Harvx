package tokenizer

import (
	"context"
	"fmt"
	"runtime"

	"golang.org/x/sync/errgroup"

	"github.com/harvx/harvx/internal/pipeline"
)

// TokenCounter wraps a Tokenizer and provides parallel per-file token counting.
// It is safe for concurrent use.
type TokenCounter struct {
	tokenizer Tokenizer
}

// NewTokenCounter creates a new TokenCounter using the given Tokenizer.
// The provided Tokenizer must be safe for concurrent use from multiple goroutines;
// all built-in implementations satisfy this requirement.
func NewTokenCounter(t Tokenizer) *TokenCounter {
	return &TokenCounter{tokenizer: t}
}

// CountFile populates fd.TokenCount from fd.Content.
// Empty content results in a token count of zero.
// This method is safe to call concurrently from multiple goroutines.
func (c *TokenCounter) CountFile(fd *pipeline.FileDescriptor) {
	fd.TokenCount = c.tokenizer.Count(fd.Content)
}

// CountFiles counts tokens for all files in parallel and returns the total
// token count across all files. Workers are bounded to runtime.NumCPU()
// concurrent goroutines. Context cancellation is respected: if ctx is
// cancelled before all files are processed, the outstanding goroutines are
// drained and the context error is returned.
func (c *TokenCounter) CountFiles(ctx context.Context, files []*pipeline.FileDescriptor) (int, error) {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	// totals collects per-goroutine results through a buffered channel to avoid
	// shared-memory races without requiring a mutex on the hot path.
	totals := make(chan int, len(files))

	for _, fd := range files {
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return fmt.Errorf("token counting cancelled: %w", err)
			}
			c.CountFile(fd)
			totals <- fd.TokenCount
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		close(totals)
		return 0, err
	}
	close(totals)

	total := 0
	for n := range totals {
		total += n
	}
	return total, nil
}

// EstimateOverhead estimates the token overhead introduced by the output
// document structure: header metadata, the file tree, and per-file section
// headers. The treeSize parameter is reserved for future use and currently
// has no effect on the result.
//
// Formula: overhead = 200 + (fileCount * 35)
func (c *TokenCounter) EstimateOverhead(fileCount int, treeSize int) int {
	return 200 + (fileCount * 35)
}
