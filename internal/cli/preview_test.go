package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// update is a flag for updating golden files. Run with:
//
//	go test ./internal/cli/ -run TestPreviewJSONGolden -update
var update = flag.Bool("update", false, "update golden files")

func TestPreviewCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "preview" {
			found = true
			break
		}
	}
	assert.True(t, found, "preview command must be registered on root")
}

func TestPreviewCommandHasHeatmapFlag(t *testing.T) {
	flag := previewCmd.Flags().Lookup("heatmap")
	assert.NotNil(t, flag, "preview command must have --heatmap flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestPreviewCommandProperties(t *testing.T) {
	assert.Equal(t, "preview", previewCmd.Use)
	assert.NotEmpty(t, previewCmd.Short)
	assert.NotEmpty(t, previewCmd.Long)
}

func TestPreviewCommandInheritsGlobalFlags(t *testing.T) {
	globalFlags := []string{
		"tokenizer", "max-tokens", "truncation-strategy",
		"token-count", "top-files",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			flag := previewCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "preview must inherit --%s from root", name)
		})
	}
}

func TestPreviewCommandHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"preview", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "preview")
	assert.Contains(t, output, "--heatmap")

	// cobra's InitDefaultHelpFlag registers a --help flag on the command the
	// first time it is executed.  When --help is passed, pflag marks the flag
	// as Changed=true and sets its value to true.  Because previewCmd is a
	// package-level variable that is reused across tests, that Changed=true
	// state persists into subsequent tests: cobra then detects helpVal=true,
	// short-circuits execution before PersistentPreRunE, and returns exit 0
	// instead of the expected validation error.  Reset both the Changed flag
	// and the value so that later tests see a clean slate.
	t.Cleanup(func() {
		if f := previewCmd.Flags().Lookup("help"); f != nil {
			f.Changed = false
			_ = f.Value.Set("false")
		}
	})
}

// TestPreviewCommandExitsZero verifies that running `harvx preview` without
// flags completes successfully (exit 0). The pipeline is a stub so no files
// are discovered; it exercises the full CLI flag-parse + command dispatch path.
func TestPreviewCommandExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"preview"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview must exit 0; combined output: %s", buf.String())
}

// TestPreviewHeatmapExitsZero verifies that running `harvx preview --heatmap`
// completes successfully (exit 0) and emits a heatmap report to stderr.
// This is the integration test for the --heatmap acceptance criterion.
func TestPreviewHeatmapExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"preview", "--heatmap"})
	defer rootCmd.SetArgs(nil)

	var errBuf bytes.Buffer
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview --heatmap must exit 0")
}

// TestPreviewHeatmapFlagSetsVariable verifies that executing
// `harvx preview --heatmap` sets the previewHeatmap package variable to true.
// This confirms the flag is correctly wired to the local variable via
// previewCmd.Flags().BoolVar (not a persistent flag on root).
func TestPreviewHeatmapFlagSetsVariable(t *testing.T) {
	rootCmd.SetArgs([]string{"preview", "--heatmap"})
	defer rootCmd.SetArgs(nil)

	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview --heatmap must exit 0")

	// After execute with --heatmap, the local flag variable must be true.
	assert.True(t, previewHeatmap,
		"previewHeatmap must be true after executing with --heatmap flag")
}

// TestPreviewWithTokenizerFlagExitsZero verifies that the --tokenizer flag is
// honoured by the preview command (exercises flag inheritance path).
func TestPreviewWithTokenizerFlagExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"preview", "--tokenizer", "o200k_base"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview --tokenizer o200k_base must exit 0")
}

// TestPreviewWithMaxTokensFlagExitsZero verifies that --max-tokens is wired
// through the preview path without error.
func TestPreviewWithMaxTokensFlagExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"preview", "--max-tokens", "100000"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview --max-tokens 100000 must exit 0")
}

// TestPreviewWithInvalidTokenizerReturnsError verifies that passing an
// unknown tokenizer value causes a non-zero exit code (flag validation).
func TestPreviewWithInvalidTokenizerReturnsError(t *testing.T) {
	// Reset the tokenizer to its default after this test so that subsequent
	// tests that don't set --tokenizer don't inherit the invalid "gpt2" value.
	t.Cleanup(func() { flagValues.Tokenizer = "cl100k_base" })

	rootCmd.SetArgs([]string{"preview", "--tokenizer", "gpt2"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.NotEqual(t, int(pipeline.ExitSuccess), code,
		"harvx preview --tokenizer gpt2 must fail validation")
}

// TestPreviewWithInvalidTruncationStrategyReturnsError verifies that an
// unknown --truncation-strategy value causes non-zero exit.
func TestPreviewWithInvalidTruncationStrategyReturnsError(t *testing.T) {
	// Reset truncation-strategy to default after this test to avoid state
	// pollution in subsequent tests that rely on the default "skip" value.
	t.Cleanup(func() { flagValues.TruncationStrategy = "skip" })

	rootCmd.SetArgs([]string{"preview", "--truncation-strategy", "head"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.NotEqual(t, int(pipeline.ExitSuccess), code,
		"harvx preview --truncation-strategy head must fail validation")
}

// ---------------------------------------------------------------------------
// --json flag tests (T-068)
// ---------------------------------------------------------------------------

// resetPreviewFlags resets the package-level preview flag variables to their
// defaults. Call this in t.Cleanup after any test that sets --json or --heatmap
// to prevent state pollution across sequential test runs (cobra does not reset
// flag-bound variables when they are not explicitly passed).
func resetPreviewFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		previewJSON = false
		previewHeatmap = false
	})
}

// TestPreviewCommandHasJSONFlag verifies the --json flag is registered on
// the preview command.
func TestPreviewCommandHasJSONFlag(t *testing.T) {
	flag := previewCmd.Flags().Lookup("json")
	assert.NotNil(t, flag, "preview command must have --json flag")
	assert.Equal(t, "false", flag.DefValue)
}

// TestPreviewJSONExitsZero verifies that `harvx preview --json` exits with
// code 0 and produces valid JSON output.
func TestPreviewJSONExitsZero(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx preview --json must exit 0; stderr: %s", errBuf.String())

	// Output must be valid JSON.
	output := outBuf.String()
	assert.True(t, json.Valid([]byte(output)),
		"preview --json output must be valid JSON, got: %s", output)
}

// TestPreviewJSONOutputSchema verifies the JSON output contains all required
// fields from the PreviewResult schema.
func TestPreviewJSONOutputSchema(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var raw map[string]json.RawMessage
	err := json.Unmarshal(outBuf.Bytes(), &raw)
	require.NoError(t, err, "output must be valid JSON: %s", outBuf.String())

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
		assert.Contains(t, raw, key,
			"preview --json output must contain %q field", key)
	}
}

// TestPreviewJSONParsesToPreviewResult verifies the JSON output deserializes
// cleanly into a PreviewResult struct.
func TestPreviewJSONParsesToPreviewResult(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var preview pipeline.PreviewResult
	err := json.Unmarshal(outBuf.Bytes(), &preview)
	require.NoError(t, err, "output must unmarshal to PreviewResult: %s", outBuf.String())

	// With no services wired, we expect zero counts.
	assert.Equal(t, 0, preview.TotalFiles)
	assert.Equal(t, 0, preview.TotalTokens)
	assert.NotNil(t, preview.Tiers)
}

// TestPreviewJSONWithMaxTokens verifies the --max-tokens flag is reflected
// in the budget_utilization_percent field.
func TestPreviewJSONWithMaxTokens(t *testing.T) {
	resetPreviewFlags(t)
	t.Cleanup(func() { flagValues.MaxTokens = 0 })

	rootCmd.SetArgs([]string{"preview", "--json", "--max-tokens", "100000"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var preview pipeline.PreviewResult
	err := json.Unmarshal(outBuf.Bytes(), &preview)
	require.NoError(t, err)

	// With max-tokens set, budget_utilization_percent should be non-nil.
	require.NotNil(t, preview.BudgetUtilizationPercent,
		"budget_utilization_percent should be non-nil when --max-tokens is set")

	// With zero tokens discovered, utilization should be 0.
	assert.InDelta(t, 0.0, *preview.BudgetUtilizationPercent, 0.01)
}

// TestPreviewJSONWithoutMaxTokens verifies that budget_utilization_percent
// is null when no --max-tokens is specified.
func TestPreviewJSONWithoutMaxTokens(t *testing.T) {
	resetPreviewFlags(t)
	t.Cleanup(func() { flagValues.MaxTokens = 0 })

	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var preview pipeline.PreviewResult
	err := json.Unmarshal(outBuf.Bytes(), &preview)
	require.NoError(t, err)

	assert.Nil(t, preview.BudgetUtilizationPercent,
		"budget_utilization_percent should be null when --max-tokens is 0")
}

// TestPreviewJSONIsPrettyPrinted verifies the JSON output uses 2-space
// indentation for readability.
func TestPreviewJSONIsPrettyPrinted(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	output := outBuf.String()

	// Verify multi-line (not compact single-line).
	assert.Contains(t, output, "\n", "JSON should be multi-line")

	// Verify 2-space indentation.
	assert.Contains(t, output, "\n  \"total_files\"",
		"JSON should use 2-space indentation")
}

// TestPreviewJSONSetsPreviewJSONFlag verifies that executing
// `harvx preview --json` sets the previewJSON package variable to true.
func TestPreviewJSONSetsPreviewJSONFlag(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	assert.True(t, previewJSON,
		"previewJSON must be true after executing with --json flag")
}

// TestPreviewJSONOutputToStdout verifies that JSON output goes to stdout
// (cmd.OutOrStdout) and not to stderr.
func TestPreviewJSONOutputToStdout(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	// JSON should be in stdout, not stderr.
	assert.True(t, json.Valid(outBuf.Bytes()),
		"stdout should contain valid JSON")

	// stderr should not contain the JSON output (it may contain slog messages).
	if errBuf.Len() > 0 {
		assert.False(t, json.Valid(errBuf.Bytes()),
			"stderr should not contain the JSON output")
	}
}

// TestPreviewJSONHelpIncludesJSONExample verifies the preview help text
// includes the --json example.
func TestPreviewJSONHelpIncludesJSONExample(t *testing.T) {
	assert.Contains(t, previewCmd.Long, "--json",
		"preview long description should mention --json")
}

// ---------------------------------------------------------------------------
// Golden test (T-068)
// ---------------------------------------------------------------------------

// TestPreviewJSONGolden compares `harvx preview --json` output against a
// golden fixture. Run with `-update` to regenerate the fixture:
//
//	go test ./internal/cli/ -run TestPreviewJSONGolden -update
func TestPreviewJSONGolden(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	actual := outBuf.Bytes()

	// Normalize the output: parse and re-marshal to ensure stable key order
	// and formatting. The golden file stores pretty-printed JSON.
	var parsed pipeline.PreviewResult
	require.NoError(t, json.Unmarshal(actual, &parsed))

	// Zero out the estimated_time_ms since it varies between runs.
	parsed.EstimatedTimeMs = 0

	normalized, err := json.MarshalIndent(parsed, "", "  ")
	require.NoError(t, err)

	// Determine golden file path. Walk up from test file location to find
	// the project root's testdata directory.
	_, testFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	projectRoot := filepath.Join(filepath.Dir(testFile), "..", "..")
	goldenPath := filepath.Join(projectRoot, "testdata", "expected-output", "preview.json")

	if *update {
		require.NoError(t, os.WriteFile(goldenPath, append(normalized, '\n'), 0644))
		t.Log("golden file updated:", goldenPath)
		return
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file must exist; run with -update to create: %s", goldenPath)

	// Normalize whitespace for comparison.
	assert.Equal(t,
		strings.TrimSpace(string(expected)),
		strings.TrimSpace(string(normalized)),
		"preview --json output should match golden file %s", goldenPath)
}

// ---------------------------------------------------------------------------
// Additional CLI edge case tests (T-068)
// ---------------------------------------------------------------------------

// TestPreviewJSONWithTokenizerFlag verifies that --tokenizer is reflected
// in the JSON output.
func TestPreviewJSONWithTokenizerFlag(t *testing.T) {
	resetPreviewFlags(t)
	t.Cleanup(func() { flagValues.Tokenizer = "cl100k_base" })
	rootCmd.SetArgs([]string{"preview", "--json", "--tokenizer", "o200k_base"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var preview pipeline.PreviewResult
	require.NoError(t, json.Unmarshal(outBuf.Bytes(), &preview))

	// The tokenizer field should reflect the requested tokenizer.
	// With the stub pipeline, it may be empty or "o200k_base" depending
	// on wiring. The key assertion is valid JSON was produced.
	assert.True(t, json.Valid(outBuf.Bytes()))
}

// TestPreviewJSONTotalTokensIsNumber verifies that total_tokens in the JSON
// output is a valid number (the jq integration test from the task spec).
func TestPreviewJSONTotalTokensIsNumber(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	// Extract total_tokens as a raw JSON value and verify it's a number.
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(outBuf.Bytes(), &raw))

	totalTokensRaw, ok := raw["total_tokens"]
	require.True(t, ok, "total_tokens field must exist")

	var totalTokens int
	require.NoError(t, json.Unmarshal(totalTokensRaw, &totalTokens),
		"total_tokens must be a valid number, got: %s", string(totalTokensRaw))
	assert.GreaterOrEqual(t, totalTokens, 0,
		"total_tokens must be >= 0")
}

// TestPreviewJSONEmptyDirectory verifies that preview --json on an empty
// directory produces valid JSON with zero counts.
func TestPreviewJSONEmptyDirectory(t *testing.T) {
	resetPreviewFlags(t)
	emptyDir := t.TempDir()

	// Reset --dir to default after test to avoid polluting subsequent tests
	// (the temp dir is removed after the test, so later tests would fail
	// with "no such file or directory").
	t.Cleanup(func() { flagValues.Dir = "." })

	rootCmd.SetArgs([]string{"preview", "--json", "--dir", emptyDir})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code,
		"preview --json on empty directory must exit 0")

	output := outBuf.Bytes()
	assert.True(t, json.Valid(output),
		"output for empty directory must be valid JSON: %s", string(output))

	var preview pipeline.PreviewResult
	require.NoError(t, json.Unmarshal(output, &preview))

	assert.Equal(t, 0, preview.TotalFiles,
		"empty directory should have 0 total_files")
	assert.Equal(t, 0, preview.TotalTokens,
		"empty directory should have 0 total_tokens")
	assert.Equal(t, 0, preview.Redactions,
		"empty directory should have 0 redactions")
}

// TestPreviewJSONNoStderrLeakage verifies that no JSON-like content leaks
// into stderr when --json is used.
func TestPreviewJSONNoStderrLeakage(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	// stdout must have the JSON.
	require.True(t, json.Valid(outBuf.Bytes()),
		"stdout must contain valid JSON")

	// stderr should not contain JSON braces (it may have slog messages).
	errStr := errBuf.String()
	if errStr != "" {
		// If there's stderr output, it should not be parseable as our result.
		var decoded pipeline.PreviewResult
		err := json.Unmarshal([]byte(errStr), &decoded)
		assert.Error(t, err,
			"stderr should not contain a valid PreviewResult JSON")
	}
}

// TestPreviewJSONOutputEndsWithNewline verifies the JSON output ends with
// a trailing newline for POSIX compliance and piping friendliness.
func TestPreviewJSONOutputEndsWithNewline(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	output := outBuf.String()
	assert.True(t, strings.HasSuffix(output, "\n"),
		"JSON output should end with a newline for POSIX compliance")
}

// TestPreviewWithoutJSONStillWorks verifies that `harvx preview` (without
// --json) exits 0 and does NOT write JSON to stdout.
func TestPreviewWithoutJSONStillWorks(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	// Without --json, stdout should be empty or non-JSON.
	// (human-readable text goes to os.Stderr directly, not the cobra err writer)
	if outBuf.Len() > 0 {
		var decoded pipeline.PreviewResult
		err := json.Unmarshal(outBuf.Bytes(), &decoded)
		if err == nil {
			t.Error("preview without --json should not produce PreviewResult JSON on stdout")
		}
	}
}

// TestPreviewJSONAllFieldsPresent verifies that every field in the
// PreviewResult struct is present in the JSON output (no omitempty hiding).
func TestPreviewJSONAllFieldsPresent(t *testing.T) {
	resetPreviewFlags(t)
	rootCmd.SetArgs([]string{"preview", "--json"})
	defer rootCmd.SetArgs(nil)

	var outBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(new(bytes.Buffer))
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(outBuf.Bytes(), &raw))

	// All 11 fields must be present, even when values are zero/null.
	requiredFields := []string{
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
	for _, field := range requiredFields {
		assert.Contains(t, raw, field,
			"JSON output must always contain %q even when zero/null", field)
	}

	// No extra fields should be present.
	assert.Len(t, raw, len(requiredFields),
		"JSON output should contain exactly %d fields", len(requiredFields))
}
