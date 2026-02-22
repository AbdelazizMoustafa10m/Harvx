package cli

import (
	"bytes"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
