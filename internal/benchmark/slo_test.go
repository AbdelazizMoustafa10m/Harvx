//go:build bench

package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/harvx/harvx/internal/discovery"
	"github.com/harvx/harvx/internal/pipeline"
	"github.com/harvx/harvx/internal/tokenizer"
)

// TestSLO_1KFiles_Under1Second verifies that full file discovery on a
// 1,000-file repository completes in under 1 second. This is a hard SLO
// from the PRD; exceeding it causes a test failure.
//
// The test generates a synthetic 1K-file repo, runs discovery once, and
// asserts the wall-clock duration is below the deadline. Unlike benchmarks,
// this runs the operation exactly once to match real-world single-invocation
// behavior.
func TestSLO_1KFiles_Under1Second(t *testing.T) {
	const deadline = 1 * time.Second

	dir := t.TempDir()
	GenerateTestRepo(t, dir, 1_000)

	walker := discovery.NewWalker()
	cfg := discovery.WalkerConfig{
		Root:                      dir,
		SuppressSensitiveWarnings: true,
	}
	ctx := context.Background()

	start := time.Now()
	result, err := walker.Walk(ctx, cfg)
	elapsed := time.Since(start)

	require.NoError(t, err, "discovery must not error")
	require.NotNil(t, result, "discovery result must not be nil")
	require.Greater(t, len(result.Files), 0, "must discover at least one file")

	t.Logf("1K discovery: %d files in %v", len(result.Files), elapsed)
	require.Less(t, elapsed, deadline,
		"SLO violation: 1K-file discovery took %v, deadline is %v", elapsed, deadline)
}

// TestSLO_10KFiles_Under3Seconds verifies that full file discovery on a
// 10,000-file repository completes in under 3 seconds. This SLO ensures
// Harvx remains interactive for medium-sized codebases.
func TestSLO_10KFiles_Under3Seconds(t *testing.T) {
	const deadline = 3 * time.Second

	dir := t.TempDir()
	GenerateTestRepo(t, dir, 10_000)

	walker := discovery.NewWalker()
	cfg := discovery.WalkerConfig{
		Root:                      dir,
		SuppressSensitiveWarnings: true,
	}
	ctx := context.Background()

	start := time.Now()
	result, err := walker.Walk(ctx, cfg)
	elapsed := time.Since(start)

	require.NoError(t, err, "discovery must not error")
	require.NotNil(t, result, "discovery result must not be nil")
	require.Greater(t, len(result.Files), 0, "must discover at least one file")

	t.Logf("10K discovery: %d files in %v", len(result.Files), elapsed)
	require.Less(t, elapsed, deadline,
		"SLO violation: 10K-file discovery took %v, deadline is %v", elapsed, deadline)
}

// TestSLO_TUITokenRecalc_Under300ms verifies that recalculating token counts
// for 1,000 in-memory file descriptors using the "none" estimator completes
// in under 300ms. This SLO ensures the TUI's interactive token recalculation
// (triggered when the user toggles file selection) feels instantaneous.
//
// The "none" tokenizer uses len(text)/4 estimation, which is the fast path
// used by the TUI. Real BPE tokenization is not tested here since the TUI
// defers to the estimator for responsiveness.
func TestSLO_TUITokenRecalc_Under300ms(t *testing.T) {
	const deadline = 300 * time.Millisecond

	dir := t.TempDir()
	fds := GenerateFileDescriptors(dir, 1_000)

	// Convert to pointer slice for CountFiles.
	ptrs := make([]*pipeline.FileDescriptor, len(fds))
	for i := range fds {
		ptrs[i] = &fds[i]
	}

	tok, err := tokenizer.NewTokenizer(tokenizer.NameNone)
	require.NoError(t, err, "creating 'none' tokenizer")

	counter := tokenizer.NewTokenCounter(tok)
	ctx := context.Background()

	start := time.Now()
	total, err := counter.CountFiles(ctx, ptrs)
	elapsed := time.Since(start)

	require.NoError(t, err, "CountFiles must not error")
	require.Greater(t, total, 0, "total token count must be positive")

	t.Logf("TUI token recalc: %d tokens across %d files in %v", total, len(ptrs), elapsed)
	require.Less(t, elapsed, deadline,
		"SLO violation: TUI token recalc took %v, deadline is %v", elapsed, deadline)
}
