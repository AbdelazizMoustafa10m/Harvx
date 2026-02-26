//go:build bench

package benchmark

import (
	"context"
	"testing"

	"github.com/harvx/harvx/internal/compression"
)

// BenchmarkCompression_1K measures compression performance on 1,000 file
// descriptors using the EngineAuto strategy. Each iteration resets the
// IsCompressed flag on all files so the orchestrator processes them fresh,
// producing accurate allocation and timing measurements for the full
// compress-all-files path.
func BenchmarkCompression_1K(b *testing.B) {
	dir := b.TempDir()
	fds := GenerateFileDescriptors(dir, 1_000)

	// Convert value slice of FileDescriptors to CompressibleFile pointers.
	files := make([]*compression.CompressibleFile, len(fds))
	for i := range fds {
		files[i] = &compression.CompressibleFile{
			Path:    fds[i].Path,
			Content: fds[i].Content,
		}
	}

	// Preserve original content so we can restore it between iterations.
	origContent := make([]string, len(files))
	for i, f := range files {
		origContent[i] = f.Content
	}

	cfg := compression.DefaultCompressionConfig()
	cfg.Enabled = true
	cfg.Engine = compression.EngineAuto

	orch := compression.NewOrchestrator(cfg)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		// Reset files to their original uncompressed state.
		for i, f := range files {
			f.IsCompressed = false
			f.Language = ""
			f.Content = origContent[i]
		}

		stats, err := orch.Compress(ctx, files)
		if err != nil {
			b.Fatalf("Compress failed: %v", err)
		}
		// Prevent the compiler from optimizing away the Compress call.
		if stats == nil {
			b.Fatal("unexpected nil stats")
		}
	}
}
