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
