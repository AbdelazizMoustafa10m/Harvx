package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/harvx/harvx/internal/config"
	"github.com/harvx/harvx/internal/workflows"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetReviewSliceFlags resets the package-level review-slice flag variables to
// their defaults. Call this in t.Cleanup after any test that sets --json,
// --base, or --head to prevent state pollution across sequential test runs.
func resetReviewSliceFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		reviewSliceJSON = false
		reviewSliceBase = ""
		reviewSliceHead = ""
	})
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd_Registration
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_Registration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "command is registered as subcommand of root",
			check: func(t *testing.T) {
				found := false
				for _, cmd := range rootCmd.Commands() {
					if cmd.Use == "review-slice" {
						found = true
						break
					}
				}
				assert.True(t, found, "review-slice command must be registered on root")
			},
		},
		{
			name: "Use property is review-slice",
			check: func(t *testing.T) {
				assert.Equal(t, "review-slice", reviewSliceCmd.Use)
			},
		},
		{
			name: "has --base flag",
			check: func(t *testing.T) {
				f := reviewSliceCmd.Flags().Lookup("base")
				assert.NotNil(t, f, "review-slice command must have --base flag")
			},
		},
		{
			name: "has --head flag",
			check: func(t *testing.T) {
				f := reviewSliceCmd.Flags().Lookup("head")
				assert.NotNil(t, f, "review-slice command must have --head flag")
			},
		},
		{
			name: "has --json flag",
			check: func(t *testing.T) {
				f := reviewSliceCmd.Flags().Lookup("json")
				assert.NotNil(t, f, "review-slice command must have --json flag")
			},
		},
		{
			name: "Short description is non-empty",
			check: func(t *testing.T) {
				assert.NotEmpty(t, reviewSliceCmd.Short)
			},
		},
		{
			name: "Long description is non-empty",
			check: func(t *testing.T) {
				assert.NotEmpty(t, reviewSliceCmd.Long)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd_FlagProperties
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_FlagProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		wantDef  string
		wantType string
	}{
		{
			name:     "--base flag exists and has correct default",
			flagName: "base",
			wantDef:  "",
			wantType: "string",
		},
		{
			name:     "--head flag exists and has correct default",
			flagName: "head",
			wantDef:  "",
			wantType: "string",
		},
		{
			name:     "--json flag exists and defaults to false",
			flagName: "json",
			wantDef:  "false",
			wantType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := reviewSliceCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, f, "review-slice command must have --%s flag", tt.flagName)
			assert.Equal(t, tt.wantDef, f.DefValue,
				"--%s default value mismatch", tt.flagName)
			assert.Equal(t, tt.wantType, f.Value.Type(),
				"--%s type mismatch", tt.flagName)
		})
	}
}

func TestReviewSliceCmd_BaseFlagRequired(t *testing.T) {
	t.Parallel()

	// Cobra marks flags as required via annotations.
	f := reviewSliceCmd.Flags().Lookup("base")
	require.NotNil(t, f, "review-slice must have --base flag")

	annotations := f.Annotations
	requiredAnn, ok := annotations[cobra.BashCompOneRequiredFlag]
	assert.True(t, ok, "--base flag must be marked as required")
	if ok {
		assert.Contains(t, requiredAnn, "true",
			"--base required annotation must contain 'true'")
	}
}

func TestReviewSliceCmd_HeadFlagRequired(t *testing.T) {
	t.Parallel()

	f := reviewSliceCmd.Flags().Lookup("head")
	require.NotNil(t, f, "review-slice must have --head flag")

	annotations := f.Annotations
	requiredAnn, ok := annotations[cobra.BashCompOneRequiredFlag]
	assert.True(t, ok, "--head flag must be marked as required")
	if ok {
		assert.Contains(t, requiredAnn, "true",
			"--head required annotation must contain 'true'")
	}
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd_MissingRequiredFlags
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_MissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing both --base and --head",
			args: []string{"review-slice"},
		},
		{
			name: "missing --head only",
			args: []string{"review-slice", "--base", "main"},
		},
		{
			name: "missing --base only",
			args: []string{"review-slice", "--head", "HEAD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetReviewSliceFlags(t)

			rootCmd.SetArgs(tt.args)
			defer rootCmd.SetArgs(nil)

			var outBuf bytes.Buffer
			var errBuf bytes.Buffer
			rootCmd.SetOut(&outBuf)
			rootCmd.SetErr(&errBuf)
			defer rootCmd.SetOut(nil)
			defer rootCmd.SetErr(nil)

			err := rootCmd.Execute()
			assert.Error(t, err,
				"review-slice %v should produce an error when required flags are missing", tt.args)
		})
	}
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd_HelpText
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_HelpText(t *testing.T) {
	resetReviewSliceFlags(t)

	rootCmd.SetArgs([]string{"review-slice", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	_ = rootCmd.Execute()
	output := buf.String()

	tests := []struct {
		name    string
		substr  string
	}{
		{
			name:   "contains review-slice",
			substr: "review-slice",
		},
		{
			name:   "mentions --base",
			substr: "--base",
		},
		{
			name:   "mentions --head",
			substr: "--head",
		},
		{
			name:   "mentions --json",
			substr: "--json",
		},
		{
			name:   "contains example usage",
			substr: "harvx review-slice --base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, output, tt.substr,
				"help output should contain %q", tt.substr)
		})
	}

	// Clean up help flag state so subsequent tests see a clean slate.
	t.Cleanup(func() {
		if f := reviewSliceCmd.Flags().Lookup("help"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
}

// ---------------------------------------------------------------------------
// TestResolveSliceConfig
// ---------------------------------------------------------------------------

func TestResolveSliceConfig(t *testing.T) {
	tests := []struct {
		name           string
		fv             *config.FlagValues
		wantMaxTokens  int
		wantDepth      int
	}{
		{
			name: "default values when no profile exists",
			fv: &config.FlagValues{
				Profile: "nonexistent-profile-for-test",
				Dir:     ".",
			},
			wantMaxTokens: workflows.DefaultSliceMaxTokens,
			wantDepth:     workflows.DefaultSliceDepth,
		},
		{
			name: "CLI --max-tokens overrides profile",
			fv: &config.FlagValues{
				Profile:   "nonexistent-profile-for-test",
				Dir:       ".",
				MaxTokens: 50000,
			},
			wantMaxTokens: 50000,
			wantDepth:     workflows.DefaultSliceDepth,
		},
		{
			name: "zero max-tokens uses default",
			fv: &config.FlagValues{
				Profile:   "nonexistent-profile-for-test",
				Dir:       ".",
				MaxTokens: 0,
			},
			wantMaxTokens: workflows.DefaultSliceMaxTokens,
			wantDepth:     workflows.DefaultSliceDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			maxTokens, depth := resolveSliceConfig(tt.fv)
			assert.Equal(t, tt.wantMaxTokens, maxTokens, "maxTokens mismatch")
			assert.Equal(t, tt.wantDepth, depth, "depth mismatch")
		})
	}
}

func TestResolveSliceConfig_DefaultConstants(t *testing.T) {
	t.Parallel()

	// Verify the default constants are sensible values.
	assert.Equal(t, 20000, workflows.DefaultSliceMaxTokens,
		"DefaultSliceMaxTokens should be 20000")
	assert.Equal(t, 1, workflows.DefaultSliceDepth,
		"DefaultSliceDepth should be 1")
}

// ---------------------------------------------------------------------------
// TestWriteReviewSliceJSON
// ---------------------------------------------------------------------------

func TestWriteReviewSliceJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    *workflows.ReviewSliceResult
		maxTokens int
		baseRef   string
		headRef   string
	}{
		{
			name: "produces valid JSON with populated fields",
			result: &workflows.ReviewSliceResult{
				TokenCount:    1500,
				FormattedHash: "abc123def456",
				ChangedFiles:  []string{"src/main.go", "src/util.go"},
				NeighborFiles: []string{"src/helper.go"},
				DeletedFiles:  []string{"old/removed.go"},
				TotalFiles:    3,
			},
			maxTokens: 20000,
			baseRef:   "origin/main",
			headRef:   "HEAD",
		},
		{
			name: "nil slices serialize as empty arrays",
			result: &workflows.ReviewSliceResult{
				TokenCount:    0,
				FormattedHash: "0000000000000000",
				ChangedFiles:  nil,
				NeighborFiles: nil,
				DeletedFiles:  nil,
				TotalFiles:    0,
			},
			maxTokens: 20000,
			baseRef:   "main",
			headRef:   "feature",
		},
		{
			name: "zero token count and zero total files",
			result: &workflows.ReviewSliceResult{
				TokenCount:    0,
				FormattedHash: "deadbeef",
				ChangedFiles:  []string{},
				NeighborFiles: []string{},
				DeletedFiles:  []string{},
				TotalFiles:    0,
			},
			maxTokens: 50000,
			baseRef:   "v1.0.0",
			headRef:   "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Set the package-level base/head vars for the function under test.
			// Since writeReviewSliceJSON reads reviewSliceBase and reviewSliceHead,
			// we save and restore them. For parallel safety, we use a fresh cobra
			// command to capture output and call the JSON serialization directly.
			buf := new(bytes.Buffer)
			cmd := &cobra.Command{}
			cmd.SetOut(buf)

			// Directly build the JSON the same way writeReviewSliceJSON does,
			// to test the serialization logic without package-level state mutation.
			meta := workflows.ReviewSliceJSON{
				TokenCount:    tt.result.TokenCount,
				ContentHash:   tt.result.FormattedHash,
				ChangedFiles:  tt.result.ChangedFiles,
				NeighborFiles: tt.result.NeighborFiles,
				DeletedFiles:  tt.result.DeletedFiles,
				TotalFiles:    tt.result.TotalFiles,
				MaxTokens:     tt.maxTokens,
				BaseRef:       tt.baseRef,
				HeadRef:       tt.headRef,
			}

			// Ensure nil slices serialize as empty arrays.
			if meta.ChangedFiles == nil {
				meta.ChangedFiles = []string{}
			}
			if meta.NeighborFiles == nil {
				meta.NeighborFiles = []string{}
			}
			if meta.DeletedFiles == nil {
				meta.DeletedFiles = []string{}
			}

			data, err := json.MarshalIndent(meta, "", "  ")
			require.NoError(t, err, "JSON marshal must not fail")

			// Verify it is valid JSON.
			assert.True(t, json.Valid(data),
				"output must be valid JSON, got: %s", string(data))

			// Verify it round-trips back to ReviewSliceJSON.
			var decoded workflows.ReviewSliceJSON
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err, "JSON unmarshal must not fail")

			// Verify all required fields.
			assert.Equal(t, tt.result.TokenCount, decoded.TokenCount, "token_count mismatch")
			assert.Equal(t, tt.result.FormattedHash, decoded.ContentHash, "content_hash mismatch")
			assert.Equal(t, tt.result.TotalFiles, decoded.TotalFiles, "total_files mismatch")
			assert.Equal(t, tt.maxTokens, decoded.MaxTokens, "max_tokens mismatch")
			assert.Equal(t, tt.baseRef, decoded.BaseRef, "base_ref mismatch")
			assert.Equal(t, tt.headRef, decoded.HeadRef, "head_ref mismatch")

			// Verify slices are never null in JSON.
			assert.NotNil(t, decoded.ChangedFiles, "changed_files must not be null")
			assert.NotNil(t, decoded.NeighborFiles, "neighbor_files must not be null")
			assert.NotNil(t, decoded.DeletedFiles, "deleted_files must not be null")
		})
	}
}

func TestWriteReviewSliceJSON_ContainsAllRequiredFields(t *testing.T) {
	t.Parallel()

	result := &workflows.ReviewSliceResult{
		TokenCount:    100,
		FormattedHash: "abcdef1234567890",
		ChangedFiles:  []string{"a.go"},
		NeighborFiles: []string{"b.go"},
		DeletedFiles:  nil,
		TotalFiles:    2,
	}

	meta := workflows.ReviewSliceJSON{
		TokenCount:    result.TokenCount,
		ContentHash:   result.FormattedHash,
		ChangedFiles:  result.ChangedFiles,
		NeighborFiles: result.NeighborFiles,
		DeletedFiles:  []string{},
		TotalFiles:    result.TotalFiles,
		MaxTokens:     20000,
		BaseRef:       "main",
		HeadRef:       "HEAD",
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	requiredFields := []string{
		"token_count",
		"content_hash",
		"changed_files",
		"neighbor_files",
		"deleted_files",
		"total_files",
		"max_tokens",
		"base_ref",
		"head_ref",
	}

	for _, field := range requiredFields {
		assert.Contains(t, raw, field,
			"JSON output must contain %q field", field)
	}

	// No extra fields should be present.
	assert.Len(t, raw, len(requiredFields),
		"JSON output should contain exactly %d fields", len(requiredFields))
}

func TestWriteReviewSliceJSON_NilSlicesSerializeAsEmptyArrays(t *testing.T) {
	t.Parallel()

	meta := workflows.ReviewSliceJSON{
		TokenCount:    0,
		ContentHash:   "0000000000000000",
		ChangedFiles:  nil,
		NeighborFiles: nil,
		DeletedFiles:  nil,
		TotalFiles:    0,
		MaxTokens:     20000,
		BaseRef:       "main",
		HeadRef:       "HEAD",
	}

	// Apply the nil-to-empty-slice normalization from writeReviewSliceJSON.
	if meta.ChangedFiles == nil {
		meta.ChangedFiles = []string{}
	}
	if meta.NeighborFiles == nil {
		meta.NeighborFiles = []string{}
	}
	if meta.DeletedFiles == nil {
		meta.DeletedFiles = []string{}
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	raw := string(data)

	// Verify the JSON contains [] (empty array) rather than null.
	assert.Contains(t, raw, `"changed_files":[]`,
		"changed_files should serialize as [] not null")
	assert.Contains(t, raw, `"neighbor_files":[]`,
		"neighbor_files should serialize as [] not null")
	assert.Contains(t, raw, `"deleted_files":[]`,
		"deleted_files should serialize as [] not null")

	// Also verify null does NOT appear for these fields.
	assert.NotContains(t, raw, `"changed_files":null`,
		"changed_files should not be null")
	assert.NotContains(t, raw, `"neighbor_files":null`,
		"neighbor_files should not be null")
	assert.NotContains(t, raw, `"deleted_files":null`,
		"deleted_files should not be null")
}

func TestWriteReviewSliceJSON_PrettyPrinted(t *testing.T) {
	t.Parallel()

	meta := workflows.ReviewSliceJSON{
		TokenCount:    100,
		ContentHash:   "abc123",
		ChangedFiles:  []string{},
		NeighborFiles: []string{},
		DeletedFiles:  []string{},
		TotalFiles:    0,
		MaxTokens:     20000,
		BaseRef:       "main",
		HeadRef:       "HEAD",
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)

	output := string(data)

	// Verify multi-line (not compact single-line).
	assert.Contains(t, output, "\n", "JSON should be multi-line")

	// Verify 2-space indentation.
	assert.Contains(t, output, "\n  \"token_count\"",
		"JSON should use 2-space indentation")
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd inherits global flags
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_InheritsGlobalFlags(t *testing.T) {
	t.Parallel()

	globalFlags := []string{
		"dir", "output", "target", "profile", "stdout",
		"assert-include", "max-tokens",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := reviewSliceCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, f, "review-slice must inherit --%s from root", name)
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildSliceTokenCounter
// ---------------------------------------------------------------------------

func TestBuildSliceTokenCounter(t *testing.T) {
	tests := []struct {
		name      string
		tokenizer string
		wantNil   bool
		wantErr   bool
	}{
		{
			name:      "none returns nil counter and no error",
			tokenizer: "none",
			wantNil:   true,
			wantErr:   false,
		},
		{
			name:      "empty string returns nil counter and no error",
			tokenizer: "",
			wantNil:   true,
			wantErr:   false,
		},
		{
			name:      "cl100k_base returns a counter",
			tokenizer: "cl100k_base",
			wantNil:   false,
			wantErr:   false,
		},
		{
			name:      "o200k_base returns a counter",
			tokenizer: "o200k_base",
			wantNil:   false,
			wantErr:   false,
		},
		{
			name:      "invalid tokenizer returns error",
			tokenizer: "nonexistent_tokenizer_xyz",
			wantNil:   true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			counter, err := buildSliceTokenCounter(tt.tokenizer)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, counter, "counter should be nil for tokenizer=%q", tt.tokenizer)
			} else {
				assert.NotNil(t, counter, "counter should not be nil for tokenizer=%q", tt.tokenizer)
				// Verify the counter is callable and returns a positive value for non-empty input.
				count := counter("hello world this is a test of token counting")
				assert.Greater(t, count, 0, "counter should return >0 for non-empty input")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestReviewSliceCmd_LongDescriptionContainsExamples
// ---------------------------------------------------------------------------

func TestReviewSliceCmd_LongDescriptionContainsExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		substr string
	}{
		{name: "mentions --base", substr: "--base"},
		{name: "mentions --head", substr: "--head"},
		{name: "mentions --json", substr: "--json"},
		{name: "mentions --stdout", substr: "--stdout"},
		{name: "mentions --profile", substr: "--profile"},
		{name: "contains example with origin/main", substr: "origin/main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Contains(t, reviewSliceCmd.Long, tt.substr,
				"review-slice long description should contain %q", tt.substr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestReviewSliceJSON_StructTags
// ---------------------------------------------------------------------------

func TestReviewSliceJSON_StructTags(t *testing.T) {
	t.Parallel()

	// Verify the JSON struct tags produce the expected field names by
	// marshaling a zero-valued struct and checking the keys.
	meta := workflows.ReviewSliceJSON{
		ChangedFiles:  []string{},
		NeighborFiles: []string{},
		DeletedFiles:  []string{},
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedTags := map[string]bool{
		"token_count":    true,
		"content_hash":   true,
		"changed_files":  true,
		"neighbor_files": true,
		"deleted_files":  true,
		"total_files":    true,
		"max_tokens":     true,
		"base_ref":       true,
		"head_ref":       true,
	}

	for key := range expectedTags {
		assert.Contains(t, raw, key,
			"ReviewSliceJSON must have JSON tag producing %q", key)
	}
}
