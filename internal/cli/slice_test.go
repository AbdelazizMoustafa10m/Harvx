package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/harvx/harvx/internal/workflows"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetSliceFlags resets the package-level slice flag variables to their
// defaults. Call this in t.Cleanup after any test that sets --json or --path
// to prevent state pollution across sequential test runs.
func resetSliceFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		sliceJSON = false
		slicePaths = nil
	})
}

// ---------------------------------------------------------------------------
// TestSliceCmd_Registration
// ---------------------------------------------------------------------------

func TestSliceCmd_Registration(t *testing.T) {
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
					if cmd.Use == "slice" {
						found = true
						break
					}
				}
				assert.True(t, found, "slice command must be registered on root")
			},
		},
		{
			name: "Use property is slice",
			check: func(t *testing.T) {
				assert.Equal(t, "slice", sliceCmd.Use)
			},
		},
		{
			name: "has --path flag",
			check: func(t *testing.T) {
				f := sliceCmd.Flags().Lookup("path")
				assert.NotNil(t, f, "slice command must have --path flag")
			},
		},
		{
			name: "has --json flag",
			check: func(t *testing.T) {
				f := sliceCmd.Flags().Lookup("json")
				assert.NotNil(t, f, "slice command must have --json flag")
			},
		},
		{
			name: "Short description is non-empty",
			check: func(t *testing.T) {
				assert.NotEmpty(t, sliceCmd.Short)
			},
		},
		{
			name: "Long description is non-empty",
			check: func(t *testing.T) {
				assert.NotEmpty(t, sliceCmd.Long)
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
// TestSliceCmd_FlagProperties
// ---------------------------------------------------------------------------

func TestSliceCmd_FlagProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		wantDef  string
		wantType string
	}{
		{
			name:     "--path flag exists and has correct default",
			flagName: "path",
			wantDef:  "[]",
			wantType: "stringArray",
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

			f := sliceCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, f, "slice command must have --%s flag", tt.flagName)
			assert.Equal(t, tt.wantDef, f.DefValue,
				"--%s default value mismatch", tt.flagName)
			assert.Equal(t, tt.wantType, f.Value.Type(),
				"--%s type mismatch", tt.flagName)
		})
	}
}

// ---------------------------------------------------------------------------
// TestSliceCmd_PathFlagRequired
// ---------------------------------------------------------------------------

func TestSliceCmd_PathFlagRequired(t *testing.T) {
	t.Parallel()

	// Cobra marks flags as required via annotations.
	f := sliceCmd.Flags().Lookup("path")
	require.NotNil(t, f, "slice must have --path flag")

	annotations := f.Annotations
	requiredAnn, ok := annotations[cobra.BashCompOneRequiredFlag]
	assert.True(t, ok, "--path flag must be marked as required")
	if ok {
		assert.Contains(t, requiredAnn, "true",
			"--path required annotation must contain 'true'")
	}
}

// ---------------------------------------------------------------------------
// TestSliceCmd_MissingRequiredFlags
// ---------------------------------------------------------------------------

func TestSliceCmd_MissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing --path",
			args: []string{"slice"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSliceFlags(t)

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
				"slice %v should produce an error when required flags are missing", tt.args)
		})
	}
}

// ---------------------------------------------------------------------------
// TestSliceCmd_HelpText
// ---------------------------------------------------------------------------

func TestSliceCmd_HelpText(t *testing.T) {
	resetSliceFlags(t)

	rootCmd.SetArgs([]string{"slice", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	_ = rootCmd.Execute()
	output := buf.String()

	tests := []struct {
		name   string
		substr string
	}{
		{
			name:   "contains slice",
			substr: "slice",
		},
		{
			name:   "mentions --path",
			substr: "--path",
		},
		{
			name:   "mentions --json",
			substr: "--json",
		},
		{
			name:   "contains example with --path",
			substr: "harvx slice --path",
		},
		{
			name:   "mentions --stdout",
			substr: "--stdout",
		},
		{
			name:   "mentions --max-tokens",
			substr: "--max-tokens",
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
		if f := sliceCmd.Flags().Lookup("help"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
}

// ---------------------------------------------------------------------------
// TestWriteSliceJSON
// ---------------------------------------------------------------------------

func TestWriteSliceJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		result    *workflows.ModuleSliceResult
		maxTokens int
		paths     []string
	}{
		{
			name: "produces valid JSON with populated fields",
			result: &workflows.ModuleSliceResult{
				TokenCount:    1500,
				FormattedHash: "abc123def456",
				ModuleFiles:   []string{"internal/auth/auth.go", "internal/auth/handler.go"},
				NeighborFiles: []string{"internal/middleware/jwt.go"},
				TotalFiles:    3,
			},
			maxTokens: 20000,
			paths:     []string{"internal/auth"},
		},
		{
			name: "nil slices serialize as empty arrays",
			result: &workflows.ModuleSliceResult{
				TokenCount:    0,
				FormattedHash: "0000000000000000",
				ModuleFiles:   nil,
				NeighborFiles: nil,
				TotalFiles:    0,
			},
			maxTokens: 20000,
			paths:     []string{"internal/empty"},
		},
		{
			name: "multiple paths in JSON output",
			result: &workflows.ModuleSliceResult{
				TokenCount:    500,
				FormattedHash: "deadbeef12345678",
				ModuleFiles:   []string{"src/a.go"},
				NeighborFiles: []string{},
				TotalFiles:    1,
			},
			maxTokens: 50000,
			paths:     []string{"src/auth", "src/middleware"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			cmd := &cobra.Command{}
			cmd.SetOut(buf)

			// Build the JSON the same way writeSliceJSON does, to test the
			// serialization logic without package-level state mutation.
			meta := workflows.ModuleSliceJSON{
				TokenCount:    tt.result.TokenCount,
				ContentHash:   tt.result.FormattedHash,
				ModuleFiles:   tt.result.ModuleFiles,
				NeighborFiles: tt.result.NeighborFiles,
				TotalFiles:    tt.result.TotalFiles,
				MaxTokens:     tt.maxTokens,
				Paths:         tt.paths,
			}

			// Ensure nil slices serialize as empty arrays.
			if meta.ModuleFiles == nil {
				meta.ModuleFiles = []string{}
			}
			if meta.NeighborFiles == nil {
				meta.NeighborFiles = []string{}
			}
			if meta.Paths == nil {
				meta.Paths = []string{}
			}

			data, err := json.MarshalIndent(meta, "", "  ")
			require.NoError(t, err, "JSON marshal must not fail")

			// Verify it is valid JSON.
			assert.True(t, json.Valid(data),
				"output must be valid JSON, got: %s", string(data))

			// Verify it round-trips back to ModuleSliceJSON.
			var decoded workflows.ModuleSliceJSON
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err, "JSON unmarshal must not fail")

			// Verify all required fields.
			assert.Equal(t, tt.result.TokenCount, decoded.TokenCount, "token_count mismatch")
			assert.Equal(t, tt.result.FormattedHash, decoded.ContentHash, "content_hash mismatch")
			assert.Equal(t, tt.result.TotalFiles, decoded.TotalFiles, "total_files mismatch")
			assert.Equal(t, tt.maxTokens, decoded.MaxTokens, "max_tokens mismatch")

			// Verify slices are never null in JSON.
			assert.NotNil(t, decoded.ModuleFiles, "module_files must not be null")
			assert.NotNil(t, decoded.NeighborFiles, "neighbor_files must not be null")
			assert.NotNil(t, decoded.Paths, "paths must not be null")
		})
	}
}

// ---------------------------------------------------------------------------
// TestWriteSliceJSON_ContainsAllRequiredFields
// ---------------------------------------------------------------------------

func TestWriteSliceJSON_ContainsAllRequiredFields(t *testing.T) {
	t.Parallel()

	meta := workflows.ModuleSliceJSON{
		TokenCount:    100,
		ContentHash:   "abcdef1234567890",
		ModuleFiles:   []string{"a.go"},
		NeighborFiles: []string{"b.go"},
		TotalFiles:    2,
		MaxTokens:     20000,
		Paths:         []string{"internal/auth"},
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	requiredFields := []string{
		"token_count",
		"content_hash",
		"module_files",
		"neighbor_files",
		"total_files",
		"max_tokens",
		"paths",
	}

	for _, field := range requiredFields {
		assert.Contains(t, raw, field,
			"JSON output must contain %q field", field)
	}

	// No extra fields should be present.
	assert.Len(t, raw, len(requiredFields),
		"JSON output should contain exactly %d fields", len(requiredFields))
}

// ---------------------------------------------------------------------------
// TestWriteSliceJSON_NilSlicesSerializeAsEmptyArrays
// ---------------------------------------------------------------------------

func TestWriteSliceJSON_NilSlicesSerializeAsEmptyArrays(t *testing.T) {
	t.Parallel()

	meta := workflows.ModuleSliceJSON{
		TokenCount:    0,
		ContentHash:   "0000000000000000",
		ModuleFiles:   nil,
		NeighborFiles: nil,
		TotalFiles:    0,
		MaxTokens:     20000,
		Paths:         nil,
	}

	// Apply the nil-to-empty-slice normalization from writeSliceJSON.
	if meta.ModuleFiles == nil {
		meta.ModuleFiles = []string{}
	}
	if meta.NeighborFiles == nil {
		meta.NeighborFiles = []string{}
	}
	if meta.Paths == nil {
		meta.Paths = []string{}
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	raw := string(data)

	// Verify the JSON contains [] (empty array) rather than null.
	assert.Contains(t, raw, `"module_files":[]`,
		"module_files should serialize as [] not null")
	assert.Contains(t, raw, `"neighbor_files":[]`,
		"neighbor_files should serialize as [] not null")
	assert.Contains(t, raw, `"paths":[]`,
		"paths should serialize as [] not null")

	// Also verify null does NOT appear for these fields.
	assert.NotContains(t, raw, `"module_files":null`,
		"module_files should not be null")
	assert.NotContains(t, raw, `"neighbor_files":null`,
		"neighbor_files should not be null")
	assert.NotContains(t, raw, `"paths":null`,
		"paths should not be null")
}

// ---------------------------------------------------------------------------
// TestWriteSliceJSON_PrettyPrinted
// ---------------------------------------------------------------------------

func TestWriteSliceJSON_PrettyPrinted(t *testing.T) {
	t.Parallel()

	meta := workflows.ModuleSliceJSON{
		TokenCount:    100,
		ContentHash:   "abc123",
		ModuleFiles:   []string{},
		NeighborFiles: []string{},
		TotalFiles:    0,
		MaxTokens:     20000,
		Paths:         []string{"internal/auth"},
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
// TestSliceCmd_InheritsGlobalFlags
// ---------------------------------------------------------------------------

func TestSliceCmd_InheritsGlobalFlags(t *testing.T) {
	t.Parallel()

	globalFlags := []string{
		"dir", "output", "target", "profile", "stdout",
		"assert-include", "max-tokens",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := sliceCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, f, "slice must inherit --%s from root", name)
		})
	}
}

// ---------------------------------------------------------------------------
// TestSliceCmd_LongDescriptionContainsExamples
// ---------------------------------------------------------------------------

func TestSliceCmd_LongDescriptionContainsExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		substr string
	}{
		{name: "mentions --path", substr: "--path"},
		{name: "mentions --json", substr: "--json"},
		{name: "mentions --stdout", substr: "--stdout"},
		{name: "mentions --max-tokens", substr: "--max-tokens"},
		{name: "mentions --target", substr: "--target"},
		{name: "contains example with internal/auth", substr: "internal/auth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Contains(t, sliceCmd.Long, tt.substr,
				"slice long description should contain %q", tt.substr)
		})
	}
}

// ---------------------------------------------------------------------------
// TestModuleSliceJSON_StructTags
// ---------------------------------------------------------------------------

func TestModuleSliceJSON_StructTags(t *testing.T) {
	t.Parallel()

	// Verify the JSON struct tags produce the expected field names by
	// marshaling a zero-valued struct and checking the keys.
	meta := workflows.ModuleSliceJSON{
		ModuleFiles:   []string{},
		NeighborFiles: []string{},
		Paths:         []string{},
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedTags := map[string]bool{
		"token_count":    true,
		"content_hash":   true,
		"module_files":   true,
		"neighbor_files": true,
		"total_files":    true,
		"max_tokens":     true,
		"paths":          true,
	}

	for key := range expectedTags {
		assert.Contains(t, raw, key,
			"ModuleSliceJSON must have JSON tag producing %q", key)
	}
}

// ---------------------------------------------------------------------------
// TestSliceCmd_PathFlagIsStringArray
// ---------------------------------------------------------------------------

func TestSliceCmd_PathFlagIsStringArray(t *testing.T) {
	t.Parallel()

	// Verify --path uses StringArrayVar (not StringSliceVar) so that
	// commas in paths are not parsed as separators.
	f := sliceCmd.Flags().Lookup("path")
	require.NotNil(t, f, "slice must have --path flag")
	assert.Equal(t, "stringArray", f.Value.Type(),
		"--path must use StringArrayVar, not StringSliceVar")
}
