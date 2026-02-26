//go:build bench

package benchmark

import (
	"context"
	"testing"

	"github.com/harvx/harvx/internal/discovery"
)

// benchmarkDiscovery is the shared helper that drives all discovery benchmarks.
// It generates a synthetic repository with fileCount files, then measures the
// wall-clock time and allocations of walking that repository repeatedly.
func benchmarkDiscovery(b *testing.B, fileCount int) {
	b.Helper()

	dir := b.TempDir()
	GenerateTestRepo(b, dir, fileCount)

	walker := discovery.NewWalker()
	cfg := discovery.WalkerConfig{
		Root:                      dir,
		SuppressSensitiveWarnings: true,
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		result, err := walker.Walk(ctx, cfg)
		if err != nil {
			b.Fatalf("walker.Walk failed: %v", err)
		}
		// Prevent the compiler from optimizing away the Walk call.
		if result == nil {
			b.Fatal("unexpected nil result")
		}
	}
}

// BenchmarkDiscovery_1K measures discovery performance on a 1,000-file repository.
func BenchmarkDiscovery_1K(b *testing.B) {
	benchmarkDiscovery(b, 1_000)
}

// BenchmarkDiscovery_10K measures discovery performance on a 10,000-file repository.
func BenchmarkDiscovery_10K(b *testing.B) {
	benchmarkDiscovery(b, 10_000)
}

// BenchmarkDiscovery_50K measures discovery performance on a 50,000-file repository.
func BenchmarkDiscovery_50K(b *testing.B) {
	benchmarkDiscovery(b, 50_000)
}
