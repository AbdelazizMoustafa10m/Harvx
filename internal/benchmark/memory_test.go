//go:build bench

package benchmark

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/discovery"
)

// TestMemory_10KFiles_Under500MB verifies that live heap memory during
// full file discovery on a 10,000-file repository stays below 500MB. This
// guards against memory regressions that could make Harvx unusable on
// resource-constrained CI runners or developer laptops.
//
// The test measures runtime.MemStats.HeapAlloc (live heap objects) after a
// forced GC, which reflects the peak retained memory rather than cumulative
// allocation volume. Sys (total memory obtained from the OS) is checked as
// a secondary guard against address-space bloat.
func TestMemory_10KFiles_Under500MB(t *testing.T) {
	const (
		heapLimit = 500 * 1024 * 1024   // 500MB live heap (PRD SLO)
		sysLimit  = 1024 * 1024 * 1024  // 1GB system memory (generous ceiling)
	)

	dir := t.TempDir()
	GenerateTestRepo(t, dir, 10_000)

	walker := discovery.NewWalker()
	cfg := discovery.WalkerConfig{
		Root:                      dir,
		SuppressSensitiveWarnings: true,
	}
	ctx := context.Background()

	result, err := walker.Walk(ctx, cfg)
	require.NoError(t, err, "discovery must not error")
	require.NotNil(t, result, "discovery result must not be nil")
	require.Greater(t, len(result.Files), 0, "must discover at least one file")

	// Snapshot memory after the operation with forced GC to measure live heap.
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	t.Logf("10K discovery memory: HeapAlloc=%d MB, Sys=%d MB, files=%d",
		after.HeapAlloc/(1024*1024),
		after.Sys/(1024*1024),
		len(result.Files),
	)

	require.Less(t, after.HeapAlloc, uint64(heapLimit),
		"memory SLO violation: live heap %d MB exceeds %d MB limit",
		after.HeapAlloc/(1024*1024), heapLimit/(1024*1024))

	require.Less(t, after.Sys, uint64(sysLimit),
		"memory SLO violation: system memory %d MB exceeds %d MB limit",
		after.Sys/(1024*1024), sysLimit/(1024*1024))
}
