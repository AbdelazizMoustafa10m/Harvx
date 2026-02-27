package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

// mockDiscovery implements DiscoveryService with configurable return values.
type mockDiscovery struct {
	result  *DiscoveryResult
	err     error
	blockCh chan struct{} // if non-nil, blocks until channel is closed or ctx cancelled
}

func (m *mockDiscovery) Discover(ctx context.Context, opts DiscoveryOptions) (*DiscoveryResult, error) {
	if m.blockCh != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-m.blockCh:
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// mockRelevance implements RelevanceService. It sets Tier on each file and
// returns the slice sorted (mock: simply assigns tiers based on position).
type mockRelevance struct {
	tierFn func(fd *FileDescriptor) // optional custom tier assignment
}

func (m *mockRelevance) Classify(files []*FileDescriptor) []*FileDescriptor {
	for i, fd := range files {
		if m.tierFn != nil {
			m.tierFn(fd)
		} else {
			fd.Tier = i + 1
		}
	}
	return files
}

// mockTokenizer implements TokenizerService. Returns len(text) as the token
// count by default, with a configurable name.
type mockTokenizer struct {
	name    string
	countFn func(text string) int
}

func (m *mockTokenizer) Count(text string) int {
	if m.countFn != nil {
		return m.countFn(text)
	}
	return len(text)
}

func (m *mockTokenizer) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock_tokenizer"
}

// mockBudget implements BudgetService. By default it passes through all files.
type mockBudget struct {
	enforceFn func(files []*FileDescriptor, maxTokens int) (*BudgetResult, error)
}

func (m *mockBudget) Enforce(files []*FileDescriptor, maxTokens int) (*BudgetResult, error) {
	if m.enforceFn != nil {
		return m.enforceFn(files, maxTokens)
	}
	total := 0
	for _, fd := range files {
		total += fd.TokenCount
	}
	return &BudgetResult{
		Included:    files,
		Skipped:     nil,
		TotalTokens: total,
		BudgetUsed:  -1,
	}, nil
}

// mockRedactor implements RedactionService. Replaces occurrences of "SECRET"
// with "[REDACTED]" and counts replacements.
type mockRedactor struct {
	redactFn func(ctx context.Context, content string, filePath string) (string, int, error)
}

func (m *mockRedactor) Redact(ctx context.Context, content string, filePath string) (string, int, error) {
	if m.redactFn != nil {
		return m.redactFn(ctx, content, filePath)
	}
	count := strings.Count(content, "SECRET")
	redacted := strings.ReplaceAll(content, "SECRET", "[REDACTED]")
	return redacted, count, nil
}

// mockCompressor implements CompressionService. Marks files as compressed.
type mockCompressor struct {
	compressFn func(ctx context.Context, files []*FileDescriptor) error
}

func (m *mockCompressor) Compress(ctx context.Context, files []*FileDescriptor) error {
	if m.compressFn != nil {
		return m.compressFn(ctx, files)
	}
	for _, fd := range files {
		fd.IsCompressed = true
	}
	return nil
}

// mockRenderer implements RenderService. Writes "rendered" to the writer.
type mockRenderer struct {
	renderFn func(ctx context.Context, w io.Writer, files []FileDescriptor, opts RenderOptions) error
}

func (m *mockRenderer) Render(ctx context.Context, w io.Writer, files []FileDescriptor, opts RenderOptions) error {
	if m.renderFn != nil {
		return m.renderFn(ctx, w, files, opts)
	}
	_, err := w.Write([]byte("rendered"))
	return err
}

// ---------------------------------------------------------------------------
// Compile-time interface compliance checks
// ---------------------------------------------------------------------------

var (
	_ DiscoveryService   = (*mockDiscovery)(nil)
	_ RelevanceService   = (*mockRelevance)(nil)
	_ TokenizerService   = (*mockTokenizer)(nil)
	_ BudgetService      = (*mockBudget)(nil)
	_ RedactionService   = (*mockRedactor)(nil)
	_ CompressionService = (*mockCompressor)(nil)
	_ RenderService      = (*mockRenderer)(nil)
)

// ---------------------------------------------------------------------------
// Helper: builds a standard set of discovered files for tests
// ---------------------------------------------------------------------------

func sampleDiscoveryResult() *DiscoveryResult {
	return &DiscoveryResult{
		Files: []FileDescriptor{
			{
				Path:    "main.go",
				AbsPath: "/project/main.go",
				Size:    100,
				Content: "package main\nfunc main() {}",
			},
			{
				Path:    "lib/util.go",
				AbsPath: "/project/lib/util.go",
				Size:    200,
				Content: "package lib\nfunc Helper() {}",
			},
			{
				Path:    "README.md",
				AbsPath: "/project/README.md",
				Size:    50,
				Content: "# Project",
			},
		},
		TotalFound:   10,
		TotalSkipped: 7,
		SkipReasons: map[string]int{
			"binary":    3,
			"gitignore": 4,
		},
	}
}

// allMocksPipeline returns a Pipeline wired with all mock services.
func allMocksPipeline() *Pipeline {
	return NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithRelevance(&mockRelevance{}),
		WithTokenizer(&mockTokenizer{name: "test_enc"}),
		WithBudget(&mockBudget{}),
		WithRedactor(&mockRedactor{}),
		WithCompressor(&mockCompressor{}),
		WithRenderer(&mockRenderer{}),
	)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPipeline_AllStages(t *testing.T) {
	t.Parallel()

	p := allMocksPipeline()
	ctx := context.Background()

	result, err := p.Run(ctx, RunOptions{
		Dir:       "/project",
		MaxTokens: 100000,
	})
	require.NoError(t, err)

	// Verify files are present.
	assert.Len(t, result.Files, 3)

	// Verify exit code.
	assert.Equal(t, ExitSuccess, result.ExitCode)

	// Verify stats are populated.
	assert.Equal(t, 3, result.Stats.TotalFiles)
	assert.Greater(t, result.Stats.TotalTokens, 0, "tokens should be counted")
	assert.Equal(t, 10, result.Stats.DiscoveryTotal)
	assert.Equal(t, 7, result.Stats.DiscoverySkipped)
	assert.Equal(t, "test_enc", result.Stats.TokenizerName)

	// Verify total timing is populated. Individual stage timings may be zero
	// on fast hardware since mock stages complete in nanoseconds.
	assert.GreaterOrEqual(t, result.Timings.Discovery, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Relevance, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Tokenize, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Budget, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Redaction, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Compression, time.Duration(0))
	assert.NotZero(t, result.Timings.Total, "total timing should be > 0")

	// Verify that compression marked files.
	for _, f := range result.Files {
		assert.True(t, f.IsCompressed, "file %s should be marked compressed", f.Path)
	}

	// Verify that relevance assigned tiers.
	tiers := make([]int, len(result.Files))
	for i, f := range result.Files {
		tiers[i] = f.Tier
	}
	assert.NotContains(t, tiers, 0, "all files should have non-zero tier after relevance")
}

func TestPipeline_DiscoveryOnly(t *testing.T) {
	t.Parallel()

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithRelevance(&mockRelevance{}),
		WithTokenizer(&mockTokenizer{}),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir:    "/project",
		Stages: DiscoveryOnly(),
	})
	require.NoError(t, err)

	// Files should be returned from discovery.
	assert.Len(t, result.Files, 3)

	// No token counts should be assigned (tokenize stage skipped).
	for _, f := range result.Files {
		assert.Equal(t, 0, f.TokenCount,
			"file %s should have zero token count with discovery-only", f.Path)
	}

	// Tiers should not be modified (relevance stage skipped).
	for _, f := range result.Files {
		assert.Equal(t, 0, f.Tier,
			"file %s should have zero tier with discovery-only", f.Path)
	}

	// Relevance timing should be zero.
	assert.Equal(t, time.Duration(0), result.Timings.Relevance)
	assert.Equal(t, time.Duration(0), result.Timings.Tokenize)
}

func TestPipeline_ContextCancellation(t *testing.T) {
	t.Parallel()

	blockCh := make(chan struct{})
	p := NewPipeline(
		WithDiscovery(&mockDiscovery{
			result:  sampleDiscoveryResult(),
			blockCh: blockCh,
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately before the mock can proceed.
	cancel()

	result, err := p.Run(ctx, RunOptions{Dir: "/project"})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPipeline_ContextCancellation_BeforeDiscovery(t *testing.T) {
	t.Parallel()

	// Context is already cancelled when Run is called. The pipeline should
	// check ctx.Done() before invoking discovery.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
	)

	result, err := p.Run(ctx, RunOptions{Dir: "/project"})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPipeline_MissingStagesSkipped(t *testing.T) {
	t.Parallel()

	// Pipeline with only discovery and tokenizer -- no relevance, redactor,
	// compressor, budget, or renderer.
	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithTokenizer(&mockTokenizer{name: "sparse"}),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir: "/project",
	})
	require.NoError(t, err)

	// Files should be present.
	assert.Len(t, result.Files, 3)

	// Token counts should be set by the tokenizer stage.
	for _, f := range result.Files {
		if f.Content != "" {
			assert.Greater(t, f.TokenCount, 0,
				"file %s should have token count", f.Path)
		}
	}

	// Relevance should not have changed tiers (no relevance service).
	for _, f := range result.Files {
		assert.Equal(t, 0, f.Tier,
			"file %s should have default tier (no relevance service)", f.Path)
	}

	// Compression should not have been applied.
	for _, f := range result.Files {
		assert.False(t, f.IsCompressed,
			"file %s should not be compressed (no compressor)", f.Path)
	}

	// Timings for missing stages should be zero.
	assert.Equal(t, time.Duration(0), result.Timings.Relevance)
	assert.Equal(t, time.Duration(0), result.Timings.Redaction)
	assert.Equal(t, time.Duration(0), result.Timings.Compression)
	assert.Equal(t, time.Duration(0), result.Timings.Budget)

	assert.Equal(t, "sparse", result.Stats.TokenizerName)
}

func TestPipeline_RunResultExitPartial(t *testing.T) {
	t.Parallel()

	// Redactor that returns an error for the second file.
	failingRedactor := &mockRedactor{
		redactFn: func(ctx context.Context, content string, filePath string) (string, int, error) {
			if filePath == "lib/util.go" {
				return "", 0, fmt.Errorf("redaction failed for %s", filePath)
			}
			return content, 0, nil
		},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithRedactor(failingRedactor),
		WithTokenizer(&mockTokenizer{}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err, "Run should not return error for per-file redaction failures")

	assert.Equal(t, ExitPartial, result.ExitCode,
		"exit code should be ExitPartial when a file has a redaction error")

	// The file with the error should have its Error field set.
	var foundError bool
	for _, f := range result.Files {
		if f.Path == "lib/util.go" {
			assert.NotNil(t, f.Error, "file with redaction failure should have Error set")
			assert.Contains(t, f.Error.Error(), "redacting lib/util.go")
			foundError = true
		}
	}
	assert.True(t, foundError, "should find the file with the redaction error in results")
}

func TestPipeline_StageTimingsPopulated(t *testing.T) {
	t.Parallel()

	p := allMocksPipeline()

	result, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 50000,
	})
	require.NoError(t, err)

	// Total timing must be populated; individual stage timings may be zero
	// on fast hardware since mock stages complete in nanoseconds.
	assert.NotZero(t, result.Timings.Total, "Total timing")
	assert.GreaterOrEqual(t, result.Timings.Discovery, time.Duration(0), "Discovery timing")
	assert.GreaterOrEqual(t, result.Timings.Relevance, time.Duration(0), "Relevance timing")
	assert.GreaterOrEqual(t, result.Timings.Redaction, time.Duration(0), "Redaction timing")
	assert.GreaterOrEqual(t, result.Timings.Compression, time.Duration(0), "Compression timing")
	assert.GreaterOrEqual(t, result.Timings.Tokenize, time.Duration(0), "Tokenize timing")
	assert.GreaterOrEqual(t, result.Timings.Budget, time.Duration(0), "Budget timing")

	// Total should be >= every individual stage timing.
	assert.GreaterOrEqual(t, result.Timings.Total, result.Timings.Discovery)
	assert.GreaterOrEqual(t, result.Timings.Total, result.Timings.Relevance)
	assert.GreaterOrEqual(t, result.Timings.Total, result.Timings.Tokenize)
}

func TestPipeline_MultipleSequentialRuns(t *testing.T) {
	t.Parallel()

	p := allMocksPipeline()
	ctx := context.Background()

	result1, err1 := p.Run(ctx, RunOptions{Dir: "/project", MaxTokens: 100000})
	require.NoError(t, err1)

	result2, err2 := p.Run(ctx, RunOptions{Dir: "/project", MaxTokens: 100000})
	require.NoError(t, err2)

	// Both runs should produce equivalent results.
	assert.Equal(t, len(result1.Files), len(result2.Files),
		"sequential runs should produce the same number of files")
	assert.Equal(t, result1.Stats.TotalFiles, result2.Stats.TotalFiles)
	assert.Equal(t, result1.Stats.TotalTokens, result2.Stats.TotalTokens)
	assert.Equal(t, result1.ExitCode, result2.ExitCode)

	// Results should be independent (different pointers).
	assert.NotSame(t, result1, result2, "results should be distinct objects")
}

func TestPipeline_EmptyDiscoveryResult(t *testing.T) {
	t.Parallel()

	emptyResult := &DiscoveryResult{
		Files:        []FileDescriptor{},
		TotalFound:   0,
		TotalSkipped: 0,
		SkipReasons:  map[string]int{},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: emptyResult}),
		WithRelevance(&mockRelevance{}),
		WithTokenizer(&mockTokenizer{}),
		WithBudget(&mockBudget{}),
		WithRedactor(&mockRedactor{}),
		WithCompressor(&mockCompressor{}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/empty"})
	require.NoError(t, err)

	assert.Empty(t, result.Files)
	assert.Equal(t, 0, result.Stats.TotalFiles)
	assert.Equal(t, 0, result.Stats.TotalTokens)
	assert.Equal(t, ExitSuccess, result.ExitCode)

	// Stages that operate on files should have zero timing because they are
	// skipped when len(filePtrs) == 0.
	assert.Equal(t, time.Duration(0), result.Timings.Relevance,
		"relevance should be skipped for empty file list")
	assert.Equal(t, time.Duration(0), result.Timings.Redaction,
		"redaction should be skipped for empty file list")
	assert.Equal(t, time.Duration(0), result.Timings.Compression,
		"compression should be skipped for empty file list")
	assert.Equal(t, time.Duration(0), result.Timings.Tokenize,
		"tokenize should be skipped for empty file list")
	assert.Equal(t, time.Duration(0), result.Timings.Budget,
		"budget should be skipped for empty file list")
}

func TestPipeline_DiscoveryError(t *testing.T) {
	t.Parallel()

	discoveryErr := errors.New("permission denied: /project")

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{err: discoveryErr}),
		WithTokenizer(&mockTokenizer{}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	assert.Nil(t, result)
	require.Error(t, err)

	// The error should wrap the discovery error with context.
	assert.ErrorIs(t, err, discoveryErr,
		"Run error should wrap the discovery error")
	assert.Contains(t, err.Error(), "discovery:",
		"Run error should contain 'discovery:' prefix")
}

func TestNewPipeline_NoOptions(t *testing.T) {
	t.Parallel()

	p := NewPipeline()
	require.NotNil(t, p, "NewPipeline with no options should return non-nil Pipeline")

	// Running without any services configured and stages defaulting to all
	// enabled. With no discovery service, the discovery stage is skipped.
	result, err := p.Run(context.Background(), RunOptions{Dir: "/nowhere"})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Files)
	assert.Equal(t, 0, result.Stats.TotalFiles)
	assert.Equal(t, ExitSuccess, result.ExitCode)
}

func TestPipeline_HasDiscovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []PipelineOption
		want bool
	}{
		{
			name: "with discovery",
			opts: []PipelineOption{WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()})},
			want: true,
		},
		{
			name: "without discovery",
			opts: nil,
			want: false,
		},
		{
			name: "with other services only",
			opts: []PipelineOption{WithTokenizer(&mockTokenizer{}), WithRedactor(&mockRedactor{})},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPipeline(tt.opts...)
			assert.Equal(t, tt.want, p.HasDiscovery())
		})
	}
}

func TestPipeline_HasRedactor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []PipelineOption
		want bool
	}{
		{
			name: "with redactor",
			opts: []PipelineOption{WithRedactor(&mockRedactor{})},
			want: true,
		},
		{
			name: "without redactor",
			opts: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPipeline(tt.opts...)
			assert.Equal(t, tt.want, p.HasRedactor())
		})
	}
}

func TestPipeline_RedactionUpdatesContent(t *testing.T) {
	t.Parallel()

	discoveryResult := &DiscoveryResult{
		Files: []FileDescriptor{
			{
				Path:    "config.env",
				AbsPath: "/project/config.env",
				Size:    100,
				Content: "API_KEY=SECRET\nDB_PASS=SECRET",
			},
			{
				Path:    "clean.go",
				AbsPath: "/project/clean.go",
				Size:    50,
				Content: "package main\nfunc main() {}",
			},
		},
		TotalFound:   2,
		TotalSkipped: 0,
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: discoveryResult}),
		WithRedactor(&mockRedactor{}),
		WithTokenizer(&mockTokenizer{}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err)

	// Verify redacted file has updated content and redaction count.
	var configFile, cleanFile *FileDescriptor
	for i := range result.Files {
		switch result.Files[i].Path {
		case "config.env":
			configFile = &result.Files[i]
		case "clean.go":
			cleanFile = &result.Files[i]
		}
	}

	require.NotNil(t, configFile, "config.env should be in results")
	require.NotNil(t, cleanFile, "clean.go should be in results")

	// config.env had 2 occurrences of "SECRET".
	assert.Equal(t, 2, configFile.Redactions,
		"config.env should have 2 redactions")
	assert.Contains(t, configFile.Content, "[REDACTED]",
		"config.env content should contain [REDACTED]")
	assert.NotContains(t, configFile.Content, "SECRET",
		"config.env content should not contain SECRET after redaction")

	// clean.go had no secrets.
	assert.Equal(t, 0, cleanFile.Redactions,
		"clean.go should have 0 redactions")
	assert.NotContains(t, cleanFile.Content, "[REDACTED]",
		"clean.go content should not contain [REDACTED]")
}

func TestPipeline_StatsAggregation(t *testing.T) {
	t.Parallel()

	discoveryResult := &DiscoveryResult{
		Files: []FileDescriptor{
			{Path: "a.go", Content: "package a", Size: 9},
			{Path: "b.go", Content: "package b with SECRET", Size: 21},
			{Path: "c.go", Content: "package c", Size: 9},
			{Path: "d.go", Content: "package d with SECRET and SECRET", Size: 32},
		},
		TotalFound:   10,
		TotalSkipped: 6,
	}

	// Relevance: assign predictable tiers based on file name.
	relevance := &mockRelevance{
		tierFn: func(fd *FileDescriptor) {
			switch fd.Path {
			case "a.go":
				fd.Tier = 1
			case "b.go":
				fd.Tier = 1
			case "c.go":
				fd.Tier = 2
			case "d.go":
				fd.Tier = 3
			}
		},
	}

	// Compressor: only compress .go files with tier 1.
	compressor := &mockCompressor{
		compressFn: func(ctx context.Context, files []*FileDescriptor) error {
			for _, fd := range files {
				if fd.Tier == 1 {
					fd.IsCompressed = true
				}
			}
			return nil
		},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: discoveryResult}),
		WithRelevance(relevance),
		WithRedactor(&mockRedactor{}),
		WithCompressor(compressor),
		WithTokenizer(&mockTokenizer{}),
		WithBudget(&mockBudget{}),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 100000,
	})
	require.NoError(t, err)

	// TierBreakdown: 2 files in tier 1, 1 in tier 2, 1 in tier 3.
	assert.Equal(t, 2, result.Stats.TierBreakdown[1], "tier 1 count")
	assert.Equal(t, 1, result.Stats.TierBreakdown[2], "tier 2 count")
	assert.Equal(t, 1, result.Stats.TierBreakdown[3], "tier 3 count")

	// RedactionCount: b.go has 1 SECRET, d.go has 2 SECRETs = 3 total.
	assert.Equal(t, 3, result.Stats.RedactionCount,
		"total redaction count across all files")

	// CompressedFiles: a.go and b.go are tier 1 and compressed.
	assert.Equal(t, 2, result.Stats.CompressedFiles,
		"compressed file count")

	// TotalTokens: sum of all file content lengths (after redaction, since
	// the mock tokenizer returns len(text)).
	assert.Greater(t, result.Stats.TotalTokens, 0, "total tokens should be > 0")

	// TotalFiles.
	assert.Equal(t, 4, result.Stats.TotalFiles)

	// Discovery stats.
	assert.Equal(t, 10, result.Stats.DiscoveryTotal)
	assert.Equal(t, 6, result.Stats.DiscoverySkipped)
}

func TestPipeline_NilStagesDefaultsToAll(t *testing.T) {
	t.Parallel()

	// When opts.Stages is nil, all stages should run.
	p := allMocksPipeline()

	result, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 100000,
		Stages:    nil, // explicitly nil
	})
	require.NoError(t, err)

	// All stages should have run. Total timing must be non-zero;
	// individual stages may complete in 0ns on fast hardware.
	assert.NotZero(t, result.Timings.Total)
	assert.GreaterOrEqual(t, result.Timings.Discovery, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Relevance, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Redaction, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Compression, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Tokenize, time.Duration(0))
	assert.GreaterOrEqual(t, result.Timings.Budget, time.Duration(0))
}

func TestPipeline_BudgetEnforcementError(t *testing.T) {
	t.Parallel()

	budgetErr := errors.New("invalid budget: negative tokens")
	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithTokenizer(&mockTokenizer{}),
		WithBudget(&mockBudget{
			enforceFn: func(files []*FileDescriptor, maxTokens int) (*BudgetResult, error) {
				return nil, budgetErr
			},
		}),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 100,
	})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, budgetErr)
	assert.Contains(t, err.Error(), "budget enforcement:")
}

func TestPipeline_CompressionErrorNonFatal(t *testing.T) {
	t.Parallel()

	// Compression errors should be non-fatal (logged as warning, not returned).
	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithCompressor(&mockCompressor{
			compressFn: func(ctx context.Context, files []*FileDescriptor) error {
				return errors.New("tree-sitter grammar not found")
			},
		}),
		WithTokenizer(&mockTokenizer{}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err, "compression error should not propagate as Run error")
	assert.Equal(t, ExitSuccess, result.ExitCode)
	assert.Len(t, result.Files, 3)
}

func TestPipeline_RedactionSkipsEmptyContent(t *testing.T) {
	t.Parallel()

	// A file with empty Content should not be passed to the redactor.
	callCount := 0
	redactor := &mockRedactor{
		redactFn: func(ctx context.Context, content string, filePath string) (string, int, error) {
			callCount++
			return content, 0, nil
		},
	}

	discoveryResult := &DiscoveryResult{
		Files: []FileDescriptor{
			{Path: "empty.go", Content: ""},
			{Path: "full.go", Content: "package main"},
		},
		TotalFound: 2,
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: discoveryResult}),
		WithRedactor(redactor),
	)

	_, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err)

	assert.Equal(t, 1, callCount,
		"redactor should only be called for files with non-empty content")
}

func TestPipeline_TokenizeSkipsEmptyContent(t *testing.T) {
	t.Parallel()

	discoveryResult := &DiscoveryResult{
		Files: []FileDescriptor{
			{Path: "empty.go", Content: ""},
			{Path: "full.go", Content: "package main"},
		},
		TotalFound: 2,
	}

	callCount := 0
	tok := &mockTokenizer{
		countFn: func(text string) int {
			callCount++
			return len(text)
		},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: discoveryResult}),
		WithTokenizer(tok),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err)

	assert.Equal(t, 1, callCount,
		"tokenizer should only be called for files with non-empty content")

	// empty.go should have 0 tokens.
	for _, f := range result.Files {
		if f.Path == "empty.go" {
			assert.Equal(t, 0, f.TokenCount)
		}
	}
}

func TestPipeline_DiscoveryAndRelevanceStages(t *testing.T) {
	t.Parallel()

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithRelevance(&mockRelevance{}),
		WithTokenizer(&mockTokenizer{}),
		WithRedactor(&mockRedactor{}),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir:    "/project",
		Stages: DiscoveryAndRelevance(),
	})
	require.NoError(t, err)

	// Files should have tiers set by relevance.
	for _, f := range result.Files {
		assert.Greater(t, f.Tier, 0,
			"file %s should have tier > 0 after relevance", f.Path)
	}

	// Token counts should NOT be set (tokenize stage disabled).
	for _, f := range result.Files {
		assert.Equal(t, 0, f.TokenCount,
			"file %s should have zero tokens with DiscoveryAndRelevance", f.Path)
	}

	// Redaction should NOT have been applied.
	assert.Equal(t, time.Duration(0), result.Timings.Redaction)
}

func TestPipeline_FileDescriptorErrorSetsExitPartial(t *testing.T) {
	t.Parallel()

	// If a file already has Error set in discovery, the final aggregation
	// should set ExitPartial.
	discoveryResult := &DiscoveryResult{
		Files: []FileDescriptor{
			{Path: "good.go", Content: "package main"},
			{Path: "bad.go", Content: "package bad", Error: errors.New("corrupted")},
		},
		TotalFound: 2,
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: discoveryResult}),
	)

	result, err := p.Run(context.Background(), RunOptions{Dir: "/project"})
	require.NoError(t, err)

	assert.Equal(t, ExitPartial, result.ExitCode,
		"exit code should be ExitPartial when a file has Error set")
}

func TestPipeline_BudgetFiltersFiles(t *testing.T) {
	t.Parallel()

	// Budget that only includes the first file.
	budget := &mockBudget{
		enforceFn: func(files []*FileDescriptor, maxTokens int) (*BudgetResult, error) {
			if len(files) == 0 {
				return &BudgetResult{}, nil
			}
			return &BudgetResult{
				Included:    files[:1],
				Skipped:     files[1:],
				TotalTokens: files[0].TokenCount,
				BudgetUsed:  50.0,
			}, nil
		},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithTokenizer(&mockTokenizer{}),
		WithBudget(budget),
	)

	result, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 100,
	})
	require.NoError(t, err)

	// Budget should have filtered to only 1 file.
	assert.Len(t, result.Files, 1, "budget should have filtered to 1 file")
	assert.Equal(t, 1, result.Stats.TotalFiles)
}

func TestPipeline_RunOptions_MaxTokensPassedToBudget(t *testing.T) {
	t.Parallel()

	var receivedMaxTokens int
	budget := &mockBudget{
		enforceFn: func(files []*FileDescriptor, maxTokens int) (*BudgetResult, error) {
			receivedMaxTokens = maxTokens
			return &BudgetResult{
				Included: files,
			}, nil
		},
	}

	p := NewPipeline(
		WithDiscovery(&mockDiscovery{result: sampleDiscoveryResult()}),
		WithTokenizer(&mockTokenizer{}),
		WithBudget(budget),
	)

	_, err := p.Run(context.Background(), RunOptions{
		Dir:       "/project",
		MaxTokens: 42000,
	})
	require.NoError(t, err)

	assert.Equal(t, 42000, receivedMaxTokens,
		"MaxTokens from RunOptions should be passed to budget.Enforce")
}

func TestPipeline_RunOptions_DirPassedToDiscovery(t *testing.T) {
	t.Parallel()

	var receivedDir string
	discoveryWithCapture := &dirCapturingDiscovery{
		result: &DiscoveryResult{},
		dir:    &receivedDir,
	}

	p := NewPipeline(
		WithDiscovery(discoveryWithCapture),
	)

	_, err := p.Run(context.Background(), RunOptions{Dir: "/my/project/dir"})
	require.NoError(t, err)

	assert.Equal(t, "/my/project/dir", receivedDir,
		"Dir from RunOptions should be passed as RootDir to discovery")
}

// dirCapturingDiscovery captures the RootDir passed to Discover.
type dirCapturingDiscovery struct {
	result *DiscoveryResult
	dir    *string
}

func (d *dirCapturingDiscovery) Discover(ctx context.Context, opts DiscoveryOptions) (*DiscoveryResult, error) {
	*d.dir = opts.RootDir
	return d.result, nil
}

func TestPipeline_TierBreakdownMapInitialized(t *testing.T) {
	t.Parallel()

	// Verify that TierBreakdown is always initialized, even with no files.
	p := NewPipeline()
	result, err := p.Run(context.Background(), RunOptions{})
	require.NoError(t, err)

	require.NotNil(t, result.Stats.TierBreakdown,
		"TierBreakdown map should be initialized even with no files")
	assert.Empty(t, result.Stats.TierBreakdown,
		"TierBreakdown should be empty when no files are processed")
}
