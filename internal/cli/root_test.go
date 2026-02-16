package cli

import (
	"bytes"
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
