//go:build bench

package benchmark

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/harvx/harvx/internal/output"
)

// benchmarkOutput is the shared helper that drives all output rendering
// benchmarks. It generates fileCount file descriptors with realistic content,
// configures the output pipeline for the given format, and measures the
// wall-clock time and allocations of rendering the full context document.
//
// Output is discarded via io.Discard to isolate rendering cost from I/O. A
// fixed timestamp ensures deterministic rendering across iterations.
func benchmarkOutput(b *testing.B, fileCount int, format string) {
	b.Helper()

	dir := b.TempDir()
	fds := GenerateFileDescriptors(dir, fileCount)

	cfg := output.OutputConfig{
		Format:        format,
		UseStdout:     true,
		ProjectName:   "bench",
		TokenizerName: "none",
		Timestamp:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Writer:        output.NewOutputWriterWithStreams(io.Discard, io.Discard),
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		result, err := output.RenderOutput(ctx, cfg, fds)
		if err != nil {
			b.Fatalf("RenderOutput failed: %v", err)
		}
		// Prevent the compiler from optimizing away the RenderOutput call.
		if result == nil {
			b.Fatal("unexpected nil result")
		}
	}
}

// BenchmarkOutputRendering_1K measures output rendering performance on 1,000
// file descriptors. Sub-benchmarks compare Markdown and XML formats to
// quantify the cost difference between template-based and XML rendering.
func BenchmarkOutputRendering_1K(b *testing.B) {
	b.Run("markdown", func(b *testing.B) {
		benchmarkOutput(b, 1_000, "markdown")
	})

	b.Run("xml", func(b *testing.B) {
		benchmarkOutput(b, 1_000, "xml")
	})
}

// BenchmarkOutputRendering_10K measures output rendering performance on 10,000
// file descriptors using the Markdown format. This benchmark validates that the
// rendering stage scales acceptably with file count and stays within the PRD's
// performance SLO (< 3s for 10K files end-to-end).
func BenchmarkOutputRendering_10K(b *testing.B) {
	benchmarkOutput(b, 10_000, "markdown")
}
