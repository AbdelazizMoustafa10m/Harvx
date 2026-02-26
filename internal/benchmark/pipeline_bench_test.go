//go:build bench

package benchmark

import (
	"context"
	"testing"

	"github.com/harvx/harvx/internal/discovery"
	"github.com/harvx/harvx/internal/pipeline"
)

// discoveryAdapter wraps a Walker to implement pipeline.DiscoveryService.
// It bridges the discovery.Walker API (which takes a WalkerConfig) to the
// pipeline's DiscoveryService interface (which takes DiscoveryOptions).
type discoveryAdapter struct {
	walker *discovery.Walker
	cfg    discovery.WalkerConfig
}

// Discover satisfies pipeline.DiscoveryService by delegating to the wrapped
// Walker. The RootDir from opts overrides cfg.Root so the pipeline controls
// which directory is scanned.
func (a *discoveryAdapter) Discover(ctx context.Context, opts pipeline.DiscoveryOptions) (*pipeline.DiscoveryResult, error) {
	a.cfg.Root = opts.RootDir
	return a.walker.Walk(ctx, a.cfg)
}

// benchmarkFullPipeline is the shared helper that drives all full-pipeline
// benchmarks. It constructs a Pipeline with only a discovery service (the
// real I/O workload) and measures repeated discovery-only runs against the
// provided directory. The directory must already contain synthetic files.
func benchmarkFullPipeline(b *testing.B, dir string) {
	b.Helper()

	adapter := &discoveryAdapter{
		walker: discovery.NewWalker(),
		cfg: discovery.WalkerConfig{
			SuppressSensitiveWarnings: true,
		},
	}

	p := pipeline.NewPipeline(
		pipeline.WithDiscovery(adapter),
	)

	opts := pipeline.RunOptions{
		Dir:    dir,
		Stages: pipeline.DiscoveryOnly(),
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		result, err := p.Run(ctx, opts)
		if err != nil {
			b.Fatalf("pipeline.Run failed: %v", err)
		}
		// Prevent the compiler from optimizing away the Run call.
		if result == nil {
			b.Fatal("unexpected nil result")
		}
	}
}

// BenchmarkFullPipeline_1K measures end-to-end pipeline performance with
// discovery-only on a 1,000-file synthetic repository. This exercises the
// full Pipeline.Run path including stage selection, timing, and result
// aggregation.
func BenchmarkFullPipeline_1K(b *testing.B) {
	dir := b.TempDir()
	GenerateTestRepo(b, dir, 1_000)
	benchmarkFullPipeline(b, dir)
}

// BenchmarkFullPipeline_10K measures end-to-end pipeline performance with
// discovery-only on a 10,000-file synthetic repository. This benchmark
// validates that the pipeline overhead scales linearly with file count.
func BenchmarkFullPipeline_10K(b *testing.B) {
	dir := b.TempDir()
	GenerateTestRepo(b, dir, 10_000)
	benchmarkFullPipeline(b, dir)
}

// BenchmarkFullPipeline_50K measures end-to-end pipeline performance with
// discovery-only on a 50,000-file synthetic repository. The fixture is
// generated once via GenerateCachedLargeRepo and reused across iterations
// to avoid multi-minute setup costs dominating the benchmark.
func BenchmarkFullPipeline_50K(b *testing.B) {
	dir := GenerateCachedLargeRepo(b)
	benchmarkFullPipeline(b, dir)
}
