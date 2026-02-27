//go:build bench

package benchmark

import (
	"context"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// benchmarkTokenization is the shared helper that drives all tokenization
// benchmarks. It generates fileCount file descriptors with realistic content,
// creates a tokenizer and counter for the given encoding name, then measures
// the wall-clock time and allocations of counting tokens across all files.
func benchmarkTokenization(b *testing.B, fileCount int, tokName string) {
	b.Helper()

	dir := b.TempDir()
	fds := GenerateFileDescriptors(dir, fileCount)

	// Convert value slice to pointer slice for CountFiles.
	ptrs := make([]*pipeline.FileDescriptor, len(fds))
	for i := range fds {
		ptrs[i] = &fds[i]
	}

	tok, err := tokenizer.NewTokenizer(tokName)
	if err != nil {
		b.Fatalf("creating tokenizer %q: %v", tokName, err)
	}

	counter := tokenizer.NewTokenCounter(tok)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		// Reset token counts so each iteration starts from a clean state.
		for _, fd := range ptrs {
			fd.TokenCount = 0
		}
		total, err := counter.CountFiles(ctx, ptrs)
		if err != nil {
			b.Fatalf("CountFiles failed: %v", err)
		}
		// Prevent the compiler from optimizing away the CountFiles call.
		if total == 0 {
			b.Fatal("unexpected zero total token count")
		}
	}
}

// BenchmarkTokenization_1K measures tokenization throughput on 1,000 file
// descriptors. Sub-benchmarks compare the "none" fast estimator against the
// "cl100k_base" BPE tokenizer to quantify the cost difference.
func BenchmarkTokenization_1K(b *testing.B) {
	b.Run("none", func(b *testing.B) {
		benchmarkTokenization(b, 1_000, "none")
	})

	b.Run("cl100k", func(b *testing.B) {
		benchmarkTokenization(b, 1_000, "cl100k_base")
	})
}

// BenchmarkTokenization_10K measures tokenization throughput on 10,000 file
// descriptors using the "none" fast estimator. This benchmark validates that
// the tokenization stage scales linearly with file count and stays within
// the PRD's performance SLO (< 3s for 10K files end-to-end).
func BenchmarkTokenization_10K(b *testing.B) {
	benchmarkTokenization(b, 10_000, "none")
}
