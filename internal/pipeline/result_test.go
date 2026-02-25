package pipeline

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStageSelection(t *testing.T) {
	t.Parallel()

	ss := NewStageSelection()
	require.NotNil(t, ss)

	assert.True(t, ss.Discovery, "Discovery should be true")
	assert.True(t, ss.Relevance, "Relevance should be true")
	assert.True(t, ss.Tokenize, "Tokenize should be true")
	assert.True(t, ss.Budget, "Budget should be true")
	assert.True(t, ss.Redaction, "Redaction should be true")
	assert.True(t, ss.Compression, "Compression should be true")
	assert.True(t, ss.Render, "Render should be true")
}

func TestDiscoveryOnly(t *testing.T) {
	t.Parallel()

	ss := DiscoveryOnly()
	require.NotNil(t, ss)

	assert.True(t, ss.Discovery, "Discovery should be true")
	assert.False(t, ss.Relevance, "Relevance should be false")
	assert.False(t, ss.Tokenize, "Tokenize should be false")
	assert.False(t, ss.Budget, "Budget should be false")
	assert.False(t, ss.Redaction, "Redaction should be false")
	assert.False(t, ss.Compression, "Compression should be false")
	assert.False(t, ss.Render, "Render should be false")
}

func TestDiscoveryAndRelevance(t *testing.T) {
	t.Parallel()

	ss := DiscoveryAndRelevance()
	require.NotNil(t, ss)

	assert.True(t, ss.Discovery, "Discovery should be true")
	assert.True(t, ss.Relevance, "Relevance should be true")
	assert.False(t, ss.Tokenize, "Tokenize should be false")
	assert.False(t, ss.Budget, "Budget should be false")
	assert.False(t, ss.Redaction, "Redaction should be false")
	assert.False(t, ss.Compression, "Compression should be false")
	assert.False(t, ss.Render, "Render should be false")
}

func TestStageTimings_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timings StageTimings
		want    map[string]string
	}{
		{
			name: "all zero durations",
			timings: StageTimings{},
			want: map[string]string{
				"discovery":   "0s",
				"relevance":   "0s",
				"tokenize":    "0s",
				"budget":      "0s",
				"redaction":   "0s",
				"compression": "0s",
				"render":      "0s",
				"total":       "0s",
			},
		},
		{
			name: "realistic durations",
			timings: StageTimings{
				Discovery:   150 * time.Millisecond,
				Relevance:   25 * time.Millisecond,
				Tokenize:    200 * time.Millisecond,
				Budget:      5 * time.Millisecond,
				Redaction:   75 * time.Millisecond,
				Compression: 300 * time.Millisecond,
				Render:      50 * time.Millisecond,
				Total:       805 * time.Millisecond,
			},
			want: map[string]string{
				"discovery":   "150ms",
				"relevance":   "25ms",
				"tokenize":    "200ms",
				"budget":      "5ms",
				"redaction":   "75ms",
				"compression": "300ms",
				"render":      "50ms",
				"total":       "805ms",
			},
		},
		{
			name: "sub-millisecond durations",
			timings: StageTimings{
				Discovery: 500 * time.Microsecond,
				Total:     500 * time.Microsecond,
			},
			want: map[string]string{
				"discovery":   "500µs",
				"relevance":   "0s",
				"tokenize":    "0s",
				"budget":      "0s",
				"redaction":   "0s",
				"compression": "0s",
				"render":      "0s",
				"total":       "500µs",
			},
		},
		{
			name: "second-scale durations",
			timings: StageTimings{
				Discovery: 2*time.Second + 345*time.Millisecond,
				Total:     2*time.Second + 345*time.Millisecond,
			},
			want: map[string]string{
				"discovery":   "2.345s",
				"relevance":   "0s",
				"tokenize":    "0s",
				"budget":      "0s",
				"redaction":   "0s",
				"compression": "0s",
				"render":      "0s",
				"total":       "2.345s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.timings)
			require.NoError(t, err)

			var got map[string]string
			err = json.Unmarshal(data, &got)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunResult_JSON_Roundtrip(t *testing.T) {
	t.Parallel()

	original := RunResult{
		Files: []FileDescriptor{
			{
				Path:        "cmd/main.go",
				AbsPath:     "/repo/cmd/main.go",
				Size:        1024,
				Tier:        1,
				TokenCount:  256,
				ContentHash: 12345678,
				Content:     "package main",
				Language:    "go",
			},
			{
				Path:       "README.md",
				AbsPath:    "/repo/README.md",
				Size:       512,
				Tier:       0,
				TokenCount: 128,
				Content:    "# Project",
				Language:   "markdown",
				Redactions: 2,
			},
		},
		Stats: RunStats{
			TotalFiles:       2,
			TotalTokens:      384,
			TokenizerName:    "cl100k_base",
			TierBreakdown:    map[int]int{0: 1, 1: 1},
			RedactionCount:   2,
			CompressedFiles:  0,
			DiscoveryTotal:   10,
			DiscoverySkipped: 8,
		},
		ContentHash: 9876543210,
		ExitCode:    ExitSuccess,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded RunResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify Files roundtrip.
	require.Len(t, decoded.Files, 2)
	assert.Equal(t, original.Files[0].Path, decoded.Files[0].Path)
	assert.Equal(t, original.Files[0].AbsPath, decoded.Files[0].AbsPath)
	assert.Equal(t, original.Files[0].Size, decoded.Files[0].Size)
	assert.Equal(t, original.Files[0].Tier, decoded.Files[0].Tier)
	assert.Equal(t, original.Files[0].TokenCount, decoded.Files[0].TokenCount)
	assert.Equal(t, original.Files[0].ContentHash, decoded.Files[0].ContentHash)
	assert.Equal(t, original.Files[0].Content, decoded.Files[0].Content)
	assert.Equal(t, original.Files[0].Language, decoded.Files[0].Language)

	assert.Equal(t, original.Files[1].Path, decoded.Files[1].Path)
	assert.Equal(t, original.Files[1].Redactions, decoded.Files[1].Redactions)

	// Verify Stats roundtrip.
	assert.Equal(t, original.Stats.TotalFiles, decoded.Stats.TotalFiles)
	assert.Equal(t, original.Stats.TotalTokens, decoded.Stats.TotalTokens)
	assert.Equal(t, original.Stats.TokenizerName, decoded.Stats.TokenizerName)
	assert.Equal(t, original.Stats.TierBreakdown, decoded.Stats.TierBreakdown)
	assert.Equal(t, original.Stats.RedactionCount, decoded.Stats.RedactionCount)
	assert.Equal(t, original.Stats.CompressedFiles, decoded.Stats.CompressedFiles)
	assert.Equal(t, original.Stats.DiscoveryTotal, decoded.Stats.DiscoveryTotal)
	assert.Equal(t, original.Stats.DiscoverySkipped, decoded.Stats.DiscoverySkipped)

	// Verify top-level fields roundtrip.
	assert.Equal(t, original.ContentHash, decoded.ContentHash)
	assert.Equal(t, original.ExitCode, decoded.ExitCode)

	// Note: StageTimings won't roundtrip because MarshalJSON produces strings
	// (e.g., "0s") but UnmarshalJSON expects integer nanoseconds. This is by design;
	// the custom marshal is for human-readable output.
}

func TestRunOptions_JSON_Roundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts RunOptions
	}{
		{
			name: "minimal options",
			opts: RunOptions{
				Dir: "/tmp/repo",
			},
		},
		{
			name: "full options without stages",
			opts: RunOptions{
				Dir:        "/home/user/project",
				MaxTokens:  100000,
				PathFilter: []string{"src/", "lib/"},
				GitRef:     "HEAD~1",
				GitBaseRef: "main",
				GitHeadRef: "feature-branch",
			},
		},
		{
			name: "options with stage selection",
			opts: RunOptions{
				Dir:    "/repo",
				Stages: DiscoveryOnly(),
			},
		},
		{
			name: "options with all stages",
			opts: RunOptions{
				Dir:    "/repo",
				Stages: NewStageSelection(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.opts)
			require.NoError(t, err)

			var decoded RunOptions
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.opts.Dir, decoded.Dir)
			assert.Equal(t, tt.opts.MaxTokens, decoded.MaxTokens)
			assert.Equal(t, tt.opts.PathFilter, decoded.PathFilter)
			assert.Equal(t, tt.opts.GitRef, decoded.GitRef)
			assert.Equal(t, tt.opts.GitBaseRef, decoded.GitBaseRef)
			assert.Equal(t, tt.opts.GitHeadRef, decoded.GitHeadRef)

			if tt.opts.Stages != nil {
				require.NotNil(t, decoded.Stages)
				assert.Equal(t, tt.opts.Stages.Discovery, decoded.Stages.Discovery)
				assert.Equal(t, tt.opts.Stages.Relevance, decoded.Stages.Relevance)
				assert.Equal(t, tt.opts.Stages.Tokenize, decoded.Stages.Tokenize)
				assert.Equal(t, tt.opts.Stages.Budget, decoded.Stages.Budget)
				assert.Equal(t, tt.opts.Stages.Redaction, decoded.Stages.Redaction)
				assert.Equal(t, tt.opts.Stages.Compression, decoded.Stages.Compression)
				assert.Equal(t, tt.opts.Stages.Render, decoded.Stages.Render)
			} else {
				assert.Nil(t, decoded.Stages)
			}
		})
	}
}

func TestRunStats_JSON(t *testing.T) {
	t.Parallel()

	stats := RunStats{
		TotalFiles:       42,
		TotalTokens:      15000,
		TokenizerName:    "cl100k_base",
		TierBreakdown:    map[int]int{0: 5, 1: 10, 2: 20, 3: 7},
		RedactionCount:   3,
		CompressedFiles:  12,
		DiscoveryTotal:   100,
		DiscoverySkipped: 58,
	}

	data, err := json.Marshal(stats)
	require.NoError(t, err)

	// Verify snake_case JSON keys by unmarshaling into a raw map.
	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	expectedKeys := []string{
		"total_files",
		"total_tokens",
		"tokenizer_name",
		"tier_breakdown",
		"redaction_count",
		"compressed_files",
		"discovery_total",
		"discovery_skipped",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "JSON output should contain snake_case key %q", key)
	}

	// Verify no camelCase keys leaked through.
	unexpectedKeys := []string{
		"TotalFiles",
		"TotalTokens",
		"TokenizerName",
		"TierBreakdown",
		"RedactionCount",
		"CompressedFiles",
		"DiscoveryTotal",
		"DiscoverySkipped",
		"totalFiles",
		"totalTokens",
	}
	for _, key := range unexpectedKeys {
		assert.NotContains(t, raw, key, "JSON output should not contain key %q", key)
	}

	// Verify values roundtrip correctly.
	var decoded RunStats
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, stats, decoded)
}

func TestRunResult_ZeroValue(t *testing.T) {
	t.Parallel()

	var result RunResult

	// Zero-value RunResult must marshal without panic.
	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Verify it produces valid JSON.
	assert.True(t, json.Valid(data), "marshaled zero-value RunResult should be valid JSON")

	// Verify we can unmarshal it back.
	var decoded RunResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Zero-value checks.
	assert.Nil(t, decoded.Files, "zero-value Files should be nil after roundtrip")
	assert.Equal(t, 0, decoded.Stats.TotalFiles)
	assert.Equal(t, 0, decoded.Stats.TotalTokens)
	assert.Empty(t, decoded.Stats.TokenizerName)
	assert.Nil(t, decoded.Stats.TierBreakdown)
	assert.Equal(t, uint64(0), decoded.ContentHash)
	assert.Equal(t, ExitSuccess, decoded.ExitCode)

	// Verify the JSON contains expected structure.
	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)
	assert.Contains(t, raw, "files")
	assert.Contains(t, raw, "stats")
	assert.Contains(t, raw, "timings")
	assert.Contains(t, raw, "content_hash")
	assert.Contains(t, raw, "exit_code")
}
