//go:build bench

package benchmark

import (
	"context"
	"testing"

	"github.com/harvx/harvx/internal/security"
)

// benchmarkRedaction is the shared helper that drives all redaction benchmarks.
// It generates fileCount file descriptors with realistic content, creates a
// StreamRedactor with the default pattern registry and entropy analyzer, then
// measures the wall-clock time and allocations of redacting all files.
func benchmarkRedaction(b *testing.B, fileCount int) {
	b.Helper()

	dir := b.TempDir()
	fds := GenerateFileDescriptors(dir, fileCount)

	cfg := security.RedactionConfig{
		Enabled:             true,
		ConfidenceThreshold: security.ConfidenceMedium,
	}
	redactor := security.NewStreamRedactor(
		security.NewDefaultRegistry(),
		security.NewEntropyAnalyzer(),
		cfg,
	)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for i := range fds {
			_, _, err := redactor.Redact(ctx, fds[i].Content, fds[i].Path)
			if err != nil {
				b.Fatalf("Redact failed on %s: %v", fds[i].Path, err)
			}
		}
	}
}

// BenchmarkRedaction_1K measures redaction throughput on 1,000 file descriptors.
// Each iteration scans all 1,000 files through the default pattern registry and
// entropy analyzer with medium confidence threshold.
func BenchmarkRedaction_1K(b *testing.B) {
	benchmarkRedaction(b, 1_000)
}

// BenchmarkRedaction_10K measures redaction throughput on 10,000 file descriptors.
// This benchmark validates that the redaction stage scales linearly with file
// count and stays within acceptable performance bounds for large repositories.
func BenchmarkRedaction_10K(b *testing.B) {
	benchmarkRedaction(b, 10_000)
}
