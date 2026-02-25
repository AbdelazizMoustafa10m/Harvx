package cli

import (
	"bytes"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "diff" {
			found = true
			break
		}
	}
	assert.True(t, found, "diff command must be registered on root")
}

func TestDiffCommandProperties(t *testing.T) {
	assert.Equal(t, "diff", diffCmd.Use)
	assert.Contains(t, diffCmd.Short, "differential output")
	assert.NotEmpty(t, diffCmd.Long)
	assert.NotEmpty(t, diffCmd.Example)
}

func TestDiffCommandHasSinceFlag(t *testing.T) {
	flag := diffCmd.Flags().Lookup("since")
	require.NotNil(t, flag, "diff command must have --since flag")
	assert.Equal(t, "", flag.DefValue)
	assert.Contains(t, flag.Usage, "Git ref")
}

func TestDiffCommandHasBaseFlag(t *testing.T) {
	flag := diffCmd.Flags().Lookup("base")
	require.NotNil(t, flag, "diff command must have --base flag")
	assert.Equal(t, "", flag.DefValue)
	assert.Contains(t, flag.Usage, "Base git ref")
}

func TestDiffCommandHasHeadFlag(t *testing.T) {
	flag := diffCmd.Flags().Lookup("head")
	require.NotNil(t, flag, "diff command must have --head flag")
	assert.Equal(t, "", flag.DefValue)
	assert.Contains(t, flag.Usage, "Head git ref")
}

func TestDiffCommandHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"diff", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "diff")
	assert.Contains(t, output, "--since")
	assert.Contains(t, output, "--base")
	assert.Contains(t, output, "--head")
}

func TestDiffCommandInheritsGlobalFlags(t *testing.T) {
	globalFlags := []string{
		"dir", "verbose", "quiet", "profile", "diff-only",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			flag := diffCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "diff must inherit --%s from root", name)
		})
	}
}

func TestRootCommandHasDiffOnlyFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("diff-only")
	require.NotNil(t, flag, "root command must have --diff-only persistent flag")
	assert.Equal(t, "false", flag.DefValue)
	assert.Contains(t, flag.Usage, "changed files")
}

func TestRootCommandHasProfileFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("profile")
	require.NotNil(t, flag, "root command must have --profile persistent flag")
	assert.Equal(t, "default", flag.DefValue)
}

// TestDiffCommand_NoCacheShowsHelpfulMessage verifies that running `harvx diff`
// without cached state prints a helpful message instead of a stack trace.
// This test calls runDiff directly to avoid global rootCmd state interference.
func TestDiffCommand_NoCacheShowsHelpfulMessage(t *testing.T) {
	dir := t.TempDir()

	// Save and restore flagValues.Dir to avoid interfering with other tests.
	origDir := flagValues.Dir
	origProfile := flagValues.Profile
	flagValues.Dir = dir
	flagValues.Profile = "default"
	defer func() {
		flagValues.Dir = origDir
		flagValues.Profile = origProfile
	}()

	errBuf := new(bytes.Buffer)
	diffCmd.SetErr(errBuf)
	defer diffCmd.SetErr(nil)

	err := runDiff(diffCmd, nil)
	// Should succeed (nil error) even with no cache.
	require.NoError(t, err)

	// The helpful message should be in stderr.
	assert.Contains(t, errBuf.String(), "No cached state found")
	assert.Contains(t, errBuf.String(), "harvx generate")
}