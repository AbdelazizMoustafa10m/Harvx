package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "generate" {
			found = true
			break
		}
	}
	assert.True(t, found, "generate command must be registered on root")
}

func TestGenerateCommandAlias(t *testing.T) {
	assert.Equal(t, []string{"gen"}, generateCmd.Aliases)
}

func TestGenerateCommandProperties(t *testing.T) {
	assert.Equal(t, "generate", generateCmd.Use)
	assert.Contains(t, generateCmd.Short, "Generate LLM-optimized context")
	assert.NotEmpty(t, generateCmd.Long)
}

func TestGenerateCommandHasPreviewFlag(t *testing.T) {
	flag := generateCmd.Flags().Lookup("preview")
	require.NotNil(t, flag, "generate command must have --preview flag")
	assert.Equal(t, "false", flag.DefValue)
}

func TestGenerateCommandInheritsGlobalFlags(t *testing.T) {
	globalFlags := []string{
		"dir", "output", "filter", "format", "target",
		"verbose", "quiet", "stdout", "line-numbers",
	}
	for _, name := range globalFlags {
		t.Run(name, func(t *testing.T) {
			flag := generateCmd.InheritedFlags().Lookup(name)
			assert.NotNil(t, flag, "generate must inherit --%s from root", name)
		})
	}
}

func TestGenerateCommandHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"generate", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "--preview")
}

func TestHelpGenerateCommand(t *testing.T) {
	rootCmd.SetArgs([]string{"help", "generate"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
	assert.Contains(t, buf.String(), "Recursively discover files")
}

func TestGenAliasWorks(t *testing.T) {
	rootCmd.SetArgs([]string{"gen", "--help"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
	assert.Contains(t, buf.String(), "generate")
}

func TestGenerateRunCallsPipeline(t *testing.T) {
	// Running generate with default flags (dir=".") should succeed
	// because the pipeline stub returns nil.
	rootCmd.SetArgs([]string{"generate"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
}

func TestRootNoSubcommandDelegatesToGenerate(t *testing.T) {
	// Running harvx with no subcommand should delegate to generate.
	rootCmd.SetArgs([]string{})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
}

func TestGenerateContextCancellation(t *testing.T) {
	// Pass a cancelled context and verify the command still handles it.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	rootCmd.SetContext(ctx)
	defer rootCmd.SetContext(nil)

	rootCmd.SetArgs([]string{"generate"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	// The stub doesn't check ctx yet, so it should still succeed.
	// This verifies that context is threaded through without panicking.
	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)
}
