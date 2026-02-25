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

// ---------------------------------------------------------------------------
// PreviewResult tests (T-068)
// ---------------------------------------------------------------------------

func TestPreviewResult_JSONSchemaCompliance(t *testing.T) {
	t.Parallel()

	budgetPct := 44.71
	pr := PreviewResult{
		TotalFiles:               342,
		TotalTokens:              89420,
		Tokenizer:                "o200k_base",
		Tiers:                    map[string]int{"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
		Redactions:               3,
		EstimatedTimeMs:          850,
		ContentHash:              "a1b2c3d4e5f6",
		Profile:                  "finvault",
		BudgetUtilizationPercent: &budgetPct,
		FilesTruncated:           0,
		FilesOmitted:             12,
	}

	data, err := json.MarshalIndent(pr, "", "  ")
	require.NoError(t, err)

	// Verify all expected JSON keys are present.
	var raw map[string]json.RawMessage
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	expectedKeys := []string{
		"total_files",
		"total_tokens",
		"tokenizer",
		"tiers",
		"redactions",
		"estimated_time_ms",
		"content_hash",
		"profile",
		"budget_utilization_percent",
		"files_truncated",
		"files_omitted",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "JSON output should contain key %q", key)
	}

	// Verify no unexpected camelCase keys.
	unexpectedKeys := []string{
		"TotalFiles", "TotalTokens", "Tokenizer", "Tiers",
		"Redactions", "EstimatedTimeMs", "ContentHash",
		"Profile", "BudgetUtilizationPercent", "FilesTruncated", "FilesOmitted",
	}
	for _, key := range unexpectedKeys {
		assert.NotContains(t, raw, key, "JSON should not contain camelCase key %q", key)
	}
}

func TestPreviewResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	budgetPct := 55.5
	original := PreviewResult{
		TotalFiles:               100,
		TotalTokens:              50000,
		Tokenizer:                "cl100k_base",
		Tiers:                    map[string]int{"0": 10, "1": 20, "2": 50, "3": 15, "4": 5},
		Redactions:               7,
		EstimatedTimeMs:          1234,
		ContentHash:              "deadbeef01234567",
		Profile:                  "default",
		BudgetUtilizationPercent: &budgetPct,
		FilesTruncated:           2,
		FilesOmitted:             30,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded PreviewResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.TotalFiles, decoded.TotalFiles)
	assert.Equal(t, original.TotalTokens, decoded.TotalTokens)
	assert.Equal(t, original.Tokenizer, decoded.Tokenizer)
	assert.Equal(t, original.Tiers, decoded.Tiers)
	assert.Equal(t, original.Redactions, decoded.Redactions)
	assert.Equal(t, original.EstimatedTimeMs, decoded.EstimatedTimeMs)
	assert.Equal(t, original.ContentHash, decoded.ContentHash)
	assert.Equal(t, original.Profile, decoded.Profile)
	require.NotNil(t, decoded.BudgetUtilizationPercent)
	assert.InDelta(t, *original.BudgetUtilizationPercent, *decoded.BudgetUtilizationPercent, 0.001)
	assert.Equal(t, original.FilesTruncated, decoded.FilesTruncated)
	assert.Equal(t, original.FilesOmitted, decoded.FilesOmitted)
}

func TestPreviewResult_BudgetUtilizationNull(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		TotalFiles:               10,
		TotalTokens:              500,
		Tokenizer:                "cl100k_base",
		Tiers:                    map[string]int{"2": 10},
		BudgetUtilizationPercent: nil,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"budget_utilization_percent":null`,
		"nil budget should serialize as JSON null")
}

func TestPreviewResult_TiersStringKeys(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		Tiers: map[string]int{"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	jsonStr := string(data)
	// Verify string keys are quoted.
	assert.Contains(t, jsonStr, `"0"`)
	assert.Contains(t, jsonStr, `"5"`)
}

func TestBuildPreviewResult_BasicConversion(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:       42,
			TotalTokens:      15000,
			TokenizerName:    "o200k_base",
			TierBreakdown:    map[int]int{0: 5, 1: 10, 2: 20, 3: 7},
			RedactionCount:   3,
			DiscoveryTotal:   100,
			DiscoverySkipped: 58,
		},
		ContentHash: 0xa1b2c3d4e5f6,
		Timings: StageTimings{
			Total: 850 * time.Millisecond,
		},
	}

	preview := BuildPreviewResult(runResult, "finvault", 200000)

	assert.Equal(t, 42, preview.TotalFiles)
	assert.Equal(t, 15000, preview.TotalTokens)
	assert.Equal(t, "o200k_base", preview.Tokenizer)
	assert.Equal(t, map[string]int{"0": 5, "1": 10, "2": 20, "3": 7}, preview.Tiers)
	assert.Equal(t, 3, preview.Redactions)
	assert.Equal(t, int64(850), preview.EstimatedTimeMs)
	assert.Equal(t, "a1b2c3d4e5f6", preview.ContentHash)
	assert.Equal(t, "finvault", preview.Profile)
	require.NotNil(t, preview.BudgetUtilizationPercent)
	assert.InDelta(t, 7.5, *preview.BudgetUtilizationPercent, 0.01) // 15000/200000*100
	assert.Equal(t, 0, preview.FilesTruncated)
	assert.Equal(t, 58, preview.FilesOmitted)
}

func TestBuildPreviewResult_NoBudget(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:    10,
			TotalTokens:   500,
			TokenizerName: "cl100k_base",
			TierBreakdown: map[int]int{2: 10},
		},
		Timings: StageTimings{
			Total: 100 * time.Millisecond,
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)

	assert.Nil(t, preview.BudgetUtilizationPercent,
		"budget_utilization_percent should be nil when maxTokens=0")
}

func TestBuildPreviewResult_ZeroResult(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TierBreakdown: make(map[int]int),
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)

	assert.Equal(t, 0, preview.TotalFiles)
	assert.Equal(t, 0, preview.TotalTokens)
	assert.Equal(t, "", preview.Tokenizer)
	assert.NotNil(t, preview.Tiers)
	assert.Empty(t, preview.Tiers)
	assert.Equal(t, 0, preview.Redactions)
	assert.Equal(t, int64(0), preview.EstimatedTimeMs)
	assert.Equal(t, "0", preview.ContentHash) // zero hash
	assert.Equal(t, "default", preview.Profile)
	assert.Nil(t, preview.BudgetUtilizationPercent)
	assert.Equal(t, 0, preview.FilesTruncated)
	assert.Equal(t, 0, preview.FilesOmitted)

	// Must produce valid JSON.
	data, err := json.Marshal(preview)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}

func TestBuildPreviewResult_BudgetUtilization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		totalTokens int
		maxTokens   int
		wantNil     bool
		wantPercent float64
	}{
		{
			name:        "typical usage",
			totalTokens: 89420,
			maxTokens:   200000,
			wantNil:     false,
			wantPercent: 44.71,
		},
		{
			name:        "no budget",
			totalTokens: 89420,
			maxTokens:   0,
			wantNil:     true,
		},
		{
			name:        "100% usage",
			totalTokens: 100,
			maxTokens:   100,
			wantNil:     false,
			wantPercent: 100.0,
		},
		{
			name:        "zero tokens with budget",
			totalTokens: 0,
			maxTokens:   100,
			wantNil:     false,
			wantPercent: 0.0,
		},
		{
			name:        "over budget",
			totalTokens: 250000,
			maxTokens:   200000,
			wantNil:     false,
			wantPercent: 125.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runResult := &RunResult{
				Stats: RunStats{
					TotalTokens:   tt.totalTokens,
					TierBreakdown: make(map[int]int),
				},
			}

			preview := BuildPreviewResult(runResult, "default", tt.maxTokens)

			if tt.wantNil {
				assert.Nil(t, preview.BudgetUtilizationPercent)
			} else {
				require.NotNil(t, preview.BudgetUtilizationPercent)
				assert.InDelta(t, tt.wantPercent, *preview.BudgetUtilizationPercent, 0.01)
			}
		})
	}
}

func TestBuildPreviewResult_ContentHashHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hash    uint64
		wantHex string
	}{
		{name: "zero hash", hash: 0, wantHex: "0"},
		{name: "small hash", hash: 255, wantHex: "ff"},
		{name: "typical hash", hash: 0xa1b2c3d4e5f6, wantHex: "a1b2c3d4e5f6"},
		{name: "max uint64", hash: 0xffffffffffffffff, wantHex: "ffffffffffffffff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runResult := &RunResult{
				ContentHash: tt.hash,
				Stats: RunStats{
					TierBreakdown: make(map[int]int),
				},
			}

			preview := BuildPreviewResult(runResult, "default", 0)
			assert.Equal(t, tt.wantHex, preview.ContentHash)
		})
	}
}

func TestBuildPreviewResult_TierConversion(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TierBreakdown: map[int]int{0: 5, 1: 48, 2: 180, 3: 62, 4: 30, 5: 17},
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)

	expected := map[string]int{
		"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17,
	}
	assert.Equal(t, expected, preview.Tiers)
}

func TestPreviewStages(t *testing.T) {
	t.Parallel()

	ss := PreviewStages()
	require.NotNil(t, ss)

	assert.True(t, ss.Discovery, "Discovery should be true")
	assert.True(t, ss.Relevance, "Relevance should be true")
	assert.True(t, ss.Tokenize, "Tokenize should be true")
	assert.False(t, ss.Budget, "Budget should be false")
	assert.False(t, ss.Redaction, "Redaction should be false")
	assert.False(t, ss.Compression, "Compression should be false")
	assert.False(t, ss.Render, "Render should be false")
}

// ---------------------------------------------------------------------------
// Additional PreviewResult edge case tests (T-068)
// ---------------------------------------------------------------------------

// TestPreviewResult_EmptyTiersProducesEmptyObject verifies that an empty tiers
// map serializes as JSON {} and not null.
func TestPreviewResult_EmptyTiersProducesEmptyObject(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		TotalFiles:  0,
		TotalTokens: 0,
		Tokenizer:   "cl100k_base",
		Tiers:       map[string]int{},
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"tiers":{}`,
		"empty tiers map must serialize as {} not null")
	assert.NotContains(t, jsonStr, `"tiers":null`,
		"empty tiers must not be null")
}

// TestPreviewResult_NilTiersProducesNull verifies behavior when Tiers is nil.
// This documents the expected output so callers can guard against it.
func TestPreviewResult_NilTiersProducesNull(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		TotalFiles:  0,
		TotalTokens: 0,
		Tokenizer:   "cl100k_base",
		Tiers:       nil,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"tiers":null`,
		"nil tiers map serializes as null; callers must initialize")
}

// TestPreviewResult_EmptyProfileProducesValidJSON verifies that an empty
// profile string produces valid JSON with an empty string value.
func TestPreviewResult_EmptyProfileProducesValidJSON(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		TotalFiles:  5,
		TotalTokens: 1000,
		Tokenizer:   "cl100k_base",
		Tiers:       map[string]int{"2": 5},
		Profile:     "",
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	assert.True(t, json.Valid(data), "JSON with empty profile must be valid")
	assert.Contains(t, string(data), `"profile":""`,
		"empty profile should produce empty string in JSON")

	// Roundtrip.
	var decoded PreviewResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "", decoded.Profile)
}

// TestPreviewResult_LargeNumbers verifies that very large token counts and
// file counts do not overflow or cause serialization issues.
func TestPreviewResult_LargeNumbers(t *testing.T) {
	t.Parallel()

	budgetPct := 99999.99
	pr := PreviewResult{
		TotalFiles:               1<<31 - 1, // max int32
		TotalTokens:              1<<31 - 1,
		Tokenizer:                "o200k_base",
		Tiers:                    map[string]int{"0": 1<<31 - 1},
		Redactions:               1<<31 - 1,
		EstimatedTimeMs:          1<<62 - 1,
		ContentHash:              "ffffffffffffffff",
		Profile:                  "large-test",
		BudgetUtilizationPercent: &budgetPct,
		FilesTruncated:           1<<31 - 1,
		FilesOmitted:             1<<31 - 1,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)
	assert.True(t, json.Valid(data), "JSON with large numbers must be valid")

	var decoded PreviewResult
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, pr.TotalFiles, decoded.TotalFiles)
	assert.Equal(t, pr.TotalTokens, decoded.TotalTokens)
	assert.Equal(t, pr.EstimatedTimeMs, decoded.EstimatedTimeMs)
	assert.Equal(t, pr.Redactions, decoded.Redactions)
	assert.Equal(t, pr.FilesTruncated, decoded.FilesTruncated)
	assert.Equal(t, pr.FilesOmitted, decoded.FilesOmitted)
}

// TestPreviewResult_AllFieldsBoundaryValues verifies serialization with
// boundary values: zeros, empty strings, nil pointers, and empty maps.
func TestPreviewResult_AllFieldsBoundaryValues(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		TotalFiles:               0,
		TotalTokens:              0,
		Tokenizer:                "",
		Tiers:                    map[string]int{},
		Redactions:               0,
		EstimatedTimeMs:          0,
		ContentHash:              "",
		Profile:                  "",
		BudgetUtilizationPercent: nil,
		FilesTruncated:           0,
		FilesOmitted:             0,
	}

	data, err := json.MarshalIndent(pr, "", "  ")
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	var decoded PreviewResult
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, 0, decoded.TotalFiles)
	assert.Equal(t, 0, decoded.TotalTokens)
	assert.Equal(t, "", decoded.Tokenizer)
	assert.NotNil(t, decoded.Tiers)
	assert.Empty(t, decoded.Tiers)
	assert.Equal(t, 0, decoded.Redactions)
	assert.Equal(t, int64(0), decoded.EstimatedTimeMs)
	assert.Equal(t, "", decoded.ContentHash)
	assert.Equal(t, "", decoded.Profile)
	assert.Nil(t, decoded.BudgetUtilizationPercent)
	assert.Equal(t, 0, decoded.FilesTruncated)
	assert.Equal(t, 0, decoded.FilesOmitted)
}

// TestPreviewResult_ContentHashZeroInJSON verifies that a zero content hash
// produces the string "0" not an empty string when marshaled.
func TestPreviewResult_ContentHashZeroInJSON(t *testing.T) {
	t.Parallel()

	pr := PreviewResult{
		ContentHash: "0",
		Tiers:       map[string]int{},
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"content_hash":"0"`,
		"zero content hash should be the string \"0\"")
}

// TestBuildPreviewResult_NilTierBreakdown verifies that BuildPreviewResult
// handles a nil TierBreakdown map without panicking and produces an
// initialized empty map.
func TestBuildPreviewResult_NilTierBreakdown(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:    5,
			TotalTokens:   100,
			TokenizerName: "cl100k_base",
			TierBreakdown: nil, // nil map
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)

	// Should not panic and should produce a non-nil (but empty) map.
	require.NotNil(t, preview.Tiers)
	assert.Empty(t, preview.Tiers)
}

// TestBuildPreviewResult_EstimatedTimeMsSubMillisecond verifies that
// sub-millisecond durations produce 0 for estimated_time_ms.
func TestBuildPreviewResult_EstimatedTimeMsSubMillisecond(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TierBreakdown: make(map[int]int),
		},
		Timings: StageTimings{
			Total: 500 * time.Microsecond, // 0.5ms
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)

	// 500us -> 0ms (time.Duration.Milliseconds() truncates).
	assert.Equal(t, int64(0), preview.EstimatedTimeMs,
		"sub-millisecond durations should truncate to 0ms")
}

// TestBuildPreviewResult_EstimatedTimeMsLargeDuration verifies estimated time
// for multi-second durations.
func TestBuildPreviewResult_EstimatedTimeMsLargeDuration(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TierBreakdown: make(map[int]int),
		},
		Timings: StageTimings{
			Total: 5*time.Second + 123*time.Millisecond,
		},
	}

	preview := BuildPreviewResult(runResult, "default", 0)
	assert.Equal(t, int64(5123), preview.EstimatedTimeMs)
}

// TestBuildPreviewResult_FilesOmittedFromDiscoverySkipped verifies that
// FilesOmitted is mapped from DiscoverySkipped.
func TestBuildPreviewResult_FilesOmittedFromDiscoverySkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		discoverySkipped int
		want            int
	}{
		{name: "zero skipped", discoverySkipped: 0, want: 0},
		{name: "some skipped", discoverySkipped: 42, want: 42},
		{name: "many skipped", discoverySkipped: 10000, want: 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runResult := &RunResult{
				Stats: RunStats{
					TierBreakdown:    make(map[int]int),
					DiscoverySkipped: tt.discoverySkipped,
				},
			}

			preview := BuildPreviewResult(runResult, "default", 0)
			assert.Equal(t, tt.want, preview.FilesOmitted)
		})
	}
}

// TestBuildPreviewResult_FilesTruncatedAlwaysZero verifies that FilesTruncated
// is always 0 from BuildPreviewResult (populated later by budget stage).
func TestBuildPreviewResult_FilesTruncatedAlwaysZero(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:    100,
			TotalTokens:   50000,
			TierBreakdown: map[int]int{0: 10, 1: 20, 2: 70},
		},
	}

	preview := BuildPreviewResult(runResult, "default", 100000)
	assert.Equal(t, 0, preview.FilesTruncated,
		"FilesTruncated should always be 0 from BuildPreviewResult")
}

// TestBuildPreviewResult_TokenizerNamePassthrough verifies that the tokenizer
// name is correctly passed through from RunStats.
func TestBuildPreviewResult_TokenizerNamePassthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tokenizerName string
	}{
		{name: "cl100k_base", tokenizerName: "cl100k_base"},
		{name: "o200k_base", tokenizerName: "o200k_base"},
		{name: "none", tokenizerName: "none"},
		{name: "empty string", tokenizerName: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			runResult := &RunResult{
				Stats: RunStats{
					TokenizerName: tt.tokenizerName,
					TierBreakdown: make(map[int]int),
				},
			}

			preview := BuildPreviewResult(runResult, "default", 0)
			assert.Equal(t, tt.tokenizerName, preview.Tokenizer)
		})
	}
}

// TestPreviewResult_JSONFieldCount verifies that the PreviewResult struct has
// exactly 11 JSON fields as specified by the task schema.
func TestPreviewResult_JSONFieldCount(t *testing.T) {
	t.Parallel()

	budgetPct := 50.0
	pr := PreviewResult{
		TotalFiles:               10,
		TotalTokens:              5000,
		Tokenizer:                "cl100k_base",
		Tiers:                    map[string]int{"2": 10},
		Redactions:               1,
		EstimatedTimeMs:          100,
		ContentHash:              "abc123",
		Profile:                  "default",
		BudgetUtilizationPercent: &budgetPct,
		FilesTruncated:           0,
		FilesOmitted:             5,
	}

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Len(t, raw, 11,
		"PreviewResult should have exactly 11 JSON fields per T-068 schema")
}

// TestBuildPreviewResult_FullPipelineScenario verifies BuildPreviewResult
// with a realistic RunResult from a full pipeline execution.
func TestBuildPreviewResult_FullPipelineScenario(t *testing.T) {
	t.Parallel()

	runResult := &RunResult{
		Files: []FileDescriptor{
			{Path: "main.go", Tier: 0, TokenCount: 500, Size: 1024},
			{Path: "lib/util.go", Tier: 1, TokenCount: 300, Size: 512},
			{Path: "lib/helper.go", Tier: 1, TokenCount: 200, Size: 400},
			{Path: "test/test.go", Tier: 3, TokenCount: 150, Size: 300},
			{Path: "README.md", Tier: 4, TokenCount: 100, Size: 256},
		},
		Stats: RunStats{
			TotalFiles:       5,
			TotalTokens:      1250,
			TokenizerName:    "o200k_base",
			TierBreakdown:    map[int]int{0: 1, 1: 2, 3: 1, 4: 1},
			RedactionCount:   7,
			CompressedFiles:  2,
			DiscoveryTotal:   50,
			DiscoverySkipped: 45,
		},
		ContentHash: 0xdeadbeef01234567,
		Timings: StageTimings{
			Discovery: 100 * time.Millisecond,
			Relevance: 20 * time.Millisecond,
			Tokenize:  80 * time.Millisecond,
			Total:     250 * time.Millisecond,
		},
		ExitCode: ExitSuccess,
	}

	preview := BuildPreviewResult(runResult, "finvault", 100000)

	assert.Equal(t, 5, preview.TotalFiles)
	assert.Equal(t, 1250, preview.TotalTokens)
	assert.Equal(t, "o200k_base", preview.Tokenizer)
	assert.Equal(t, map[string]int{"0": 1, "1": 2, "3": 1, "4": 1}, preview.Tiers)
	assert.Equal(t, 7, preview.Redactions)
	assert.Equal(t, int64(250), preview.EstimatedTimeMs)
	assert.Equal(t, "deadbeef01234567", preview.ContentHash)
	assert.Equal(t, "finvault", preview.Profile)
	require.NotNil(t, preview.BudgetUtilizationPercent)
	assert.InDelta(t, 1.25, *preview.BudgetUtilizationPercent, 0.001) // 1250/100000*100
	assert.Equal(t, 0, preview.FilesTruncated)
	assert.Equal(t, 45, preview.FilesOmitted)

	// Verify the full result serializes to valid JSON.
	data, err := json.MarshalIndent(preview, "", "  ")
	require.NoError(t, err)
	assert.True(t, json.Valid(data))
}

// ---------------------------------------------------------------------------
// Benchmark tests for BuildPreviewResult (T-068)
// ---------------------------------------------------------------------------

func BenchmarkBuildPreviewResult_Small(b *testing.B) {
	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:       10,
			TotalTokens:      5000,
			TokenizerName:    "cl100k_base",
			TierBreakdown:    map[int]int{0: 2, 1: 3, 2: 5},
			RedactionCount:   1,
			DiscoveryTotal:   20,
			DiscoverySkipped: 10,
		},
		ContentHash: 0xabcdef0123456789,
		Timings: StageTimings{
			Total: 100 * time.Millisecond,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildPreviewResult(runResult, "default", 100000)
	}
}

func BenchmarkBuildPreviewResult_Large(b *testing.B) {
	tierBreakdown := make(map[int]int)
	for i := 0; i < 6; i++ {
		tierBreakdown[i] = 100 + i*50
	}

	runResult := &RunResult{
		Stats: RunStats{
			TotalFiles:       1000,
			TotalTokens:      500000,
			TokenizerName:    "o200k_base",
			TierBreakdown:    tierBreakdown,
			RedactionCount:   42,
			DiscoveryTotal:   5000,
			DiscoverySkipped: 4000,
		},
		ContentHash: 0xffffffffffffffff,
		Timings: StageTimings{
			Total: 5 * time.Second,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildPreviewResult(runResult, "finvault", 200000)
	}
}

func BenchmarkPreviewResult_Marshal(b *testing.B) {
	budgetPct := 44.71
	pr := &PreviewResult{
		TotalFiles:               342,
		TotalTokens:              89420,
		Tokenizer:                "o200k_base",
		Tiers:                    map[string]int{"0": 5, "1": 48, "2": 180, "3": 62, "4": 30, "5": 17},
		Redactions:               3,
		EstimatedTimeMs:          850,
		ContentHash:              "a1b2c3d4e5f6",
		Profile:                  "finvault",
		BudgetUtilizationPercent: &budgetPct,
		FilesTruncated:           0,
		FilesOmitted:             12,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.MarshalIndent(pr, "", "  ")
	}
}
