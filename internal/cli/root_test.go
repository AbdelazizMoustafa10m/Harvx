package cli

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommandUse(t *testing.T) {
	assert.Equal(t, "harvx", rootCmd.Use)
}

func TestRootCommandShort(t *testing.T) {
	assert.Equal(t, "Harvest your context.", rootCmd.Short)
}

func TestRootCommandSilenceUsage(t *testing.T) {
	assert.True(t, rootCmd.SilenceUsage, "SilenceUsage must be true to avoid printing usage on errors")
}

func TestRootCommandSilenceErrors(t *testing.T) {
	assert.True(t, rootCmd.SilenceErrors, "SilenceErrors must be true for manual error handling")
}

func TestRootCommandHasVerboseFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, flag, "root command must have --verbose persistent flag")
	assert.Equal(t, "v", flag.Shorthand)
}

func TestRootCommandHasQuietFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("quiet")
	require.NotNil(t, flag, "root command must have --quiet persistent flag")
	assert.Equal(t, "q", flag.Shorthand)
}

func TestRootCommandHasDirFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("dir")
	require.NotNil(t, flag, "root command must have --dir persistent flag")
	assert.Equal(t, "d", flag.Shorthand)
	assert.Equal(t, ".", flag.DefValue)
}

func TestRootCommandHasOutputFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("output")
	require.NotNil(t, flag, "root command must have --output persistent flag")
	assert.Equal(t, "o", flag.Shorthand)
	assert.Equal(t, "harvx-output.md", flag.DefValue)
}

func TestRootCommandHasFormatFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("format")
	require.NotNil(t, flag, "root command must have --format persistent flag")
	assert.Equal(t, "markdown", flag.DefValue)
}

func TestRootCommandHasTargetFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("target")
	require.NotNil(t, flag, "root command must have --target persistent flag")
	assert.Equal(t, "generic", flag.DefValue)
}

func TestRootCommandHasFilterFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("filter")
	require.NotNil(t, flag, "root command must have --filter persistent flag")
	assert.Equal(t, "f", flag.Shorthand)
}

func TestRootCommandHasSkipLargeFilesFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("skip-large-files")
	require.NotNil(t, flag, "root command must have --skip-large-files persistent flag")
	assert.Equal(t, "1MB", flag.DefValue)
}

func TestRootCommandHasBooleanFlags(t *testing.T) {
	boolFlags := []string{
		"git-tracked-only",
		"stdout",
		"line-numbers",
		"no-redact",
		"fail-on-redaction",
		"yes",
		"clear-cache",
	}
	for _, name := range boolFlags {
		t.Run(name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(name)
			require.NotNil(t, flag, "root command must have --%s persistent flag", name)
			assert.Equal(t, "false", flag.DefValue)
		})
	}
}

// --- T-033: Token counting persistent flags ---

func TestRootCommandHasTokenizerFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("tokenizer")
	require.NotNil(t, flag, "root command must have --tokenizer persistent flag")
	assert.Equal(t, "cl100k_base", flag.DefValue)
}

func TestRootCommandHasMaxTokensFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("max-tokens")
	require.NotNil(t, flag, "root command must have --max-tokens persistent flag")
	assert.Equal(t, "0", flag.DefValue)
}

func TestRootCommandHasTruncationStrategyFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("truncation-strategy")
	require.NotNil(t, flag, "root command must have --truncation-strategy persistent flag")
	assert.Equal(t, "skip", flag.DefValue)
}

func TestRootCommandHasTokenCountFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("token-count")
	require.NotNil(t, flag, "root command must have --token-count persistent flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRootCommandHasTopFilesFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("top-files")
	require.NotNil(t, flag, "root command must have --top-files persistent flag")
	assert.Equal(t, "0", flag.DefValue)
}

func TestExecuteWithHelp(t *testing.T) {
	// Running with --help should succeed (exit 0).
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
	assert.Contains(t, buf.String(), "LLM-optimized context documents")
}

func TestExecuteHelpShowsAllFlags(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	expectedFlags := []string{
		"--dir", "--output", "--filter", "--include", "--exclude",
		"--format", "--target", "--git-tracked-only", "--skip-large-files",
		"--stdout", "--line-numbers", "--no-redact", "--fail-on-redaction",
		"--verbose", "--quiet", "--yes", "--clear-cache",
		"--tokenizer", "--max-tokens", "--truncation-strategy",
		"--token-count", "--top-files",
	}
	for _, flag := range expectedFlags {
		assert.Contains(t, output, flag, "help output should show %s flag", flag)
	}
}

func TestExecuteWithNoArgs(t *testing.T) {
	// Running with no args should print help and succeed.
	rootCmd.SetArgs([]string{})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
}

func TestExecuteWithUnknownFlag(t *testing.T) {
	// Running with an unknown flag should return a non-zero exit code.
	rootCmd.SetArgs([]string{"--nonexistent-flag"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitError), code)
}

func TestRootCmdReturnsCommand(t *testing.T) {
	cmd := RootCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "harvx", cmd.Use)
}

func TestRootCommandLongDescription(t *testing.T) {
	assert.Contains(t, rootCmd.Long, "LLM-optimized context documents")
}

func TestGlobalFlagsReturnsValues(t *testing.T) {
	fv := GlobalFlags()
	require.NotNil(t, fv, "GlobalFlags() should return non-nil FlagValues")
}

func TestExtractExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error returns ExitSuccess",
			err:  nil,
			want: int(pipeline.ExitSuccess),
		},
		{
			name: "generic error returns ExitError",
			err:  errors.New("something went wrong"),
			want: int(pipeline.ExitError),
		},
		{
			name: "HarvxError with ExitError code",
			err:  pipeline.NewError("fatal error", errors.New("cause")),
			want: int(pipeline.ExitError),
		},
		{
			name: "HarvxError with ExitPartial code",
			err:  pipeline.NewPartialError("partial failure", errors.New("some files failed")),
			want: int(pipeline.ExitPartial),
		},
		{
			name: "redaction error returns ExitError",
			err:  pipeline.NewRedactionError("secrets detected"),
			want: int(pipeline.ExitError),
		},
		{
			name: "wrapped HarvxError preserves exit code",
			err:  fmt.Errorf("command failed: %w", pipeline.NewPartialError("partial", nil)),
			want: int(pipeline.ExitPartial),
		},
		{
			name: "deeply wrapped HarvxError preserves exit code",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", pipeline.NewError("deep", nil))),
			want: int(pipeline.ExitError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractExitCode(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractExitCode_NilReturnsZero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, extractExitCode(nil))
}

func TestExtractExitCode_GenericErrorReturnsOne(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 1, extractExitCode(errors.New("generic")))
}

func TestExtractExitCode_PartialErrorReturnsTwo(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2, extractExitCode(pipeline.NewPartialError("partial", nil)))
}

func TestExtractExitCode_WrappedGenericErrorReturnsOne(t *testing.T) {
	t.Parallel()

	// A generic error wrapped with fmt.Errorf (no HarvxError in the chain)
	// should still return ExitError (1).
	wrappedGeneric := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", errors.New("root")))
	assert.Equal(t, 1, extractExitCode(wrappedGeneric))
}

func TestExtractExitCode_RedactionErrorReturnsOne(t *testing.T) {
	t.Parallel()

	// Explicitly verify NewRedactionError maps to exit code 1 through extractExitCode.
	assert.Equal(t, 1, extractExitCode(pipeline.NewRedactionError("secrets found")))
}
