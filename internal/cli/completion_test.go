package cli

import (
	"bytes"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "completion" {
			found = true
			break
		}
	}
	assert.True(t, found, "completion subcommand must be registered on root command")
}

func TestCompletionCommandProperties(t *testing.T) {
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
	assert.Equal(t, "Generate shell completion scripts", completionCmd.Short)
	assert.NotEmpty(t, completionCmd.Long)
}

func TestCompletionCommandValidArgs(t *testing.T) {
	expected := []string{"bash", "zsh", "fish", "powershell"}
	assert.Equal(t, expected, completionCmd.ValidArgs)
}

func TestCompletionShellScripts(t *testing.T) {
	shells := []struct {
		name     string
		contains string // a substring expected in the generated script
	}{
		{name: "bash", contains: "bash"},
		{name: "zsh", contains: "zsh"},
		{name: "fish", contains: "harvx"},
		{name: "powershell", contains: "harvx"},
	}

	for _, tt := range shells {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs([]string{"completion", tt.name})
			defer rootCmd.SetArgs(nil)

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			defer rootCmd.SetOut(nil)

			code := Execute()
			assert.Equal(t, int(pipeline.ExitSuccess), code)

			output := buf.String()
			assert.NotEmpty(t, output, "completion script for %s must not be empty", tt.name)
			assert.Contains(t, output, tt.contains,
				"completion script for %s must contain %q", tt.name, tt.contains)
		})
	}
}

func TestCompletionNoArgsShowsHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"completion"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	output := buf.String()
	// The help text should include installation instructions for each shell.
	assert.Contains(t, output, "source <(harvx completion bash)")
	assert.Contains(t, output, `"${fpath[1]}/_harvx"`)
	assert.Contains(t, output, "~/.config/fish/completions/harvx.fish")
	assert.Contains(t, output, "Out-String | Invoke-Expression")
}

func TestCompletionInvalidShellReturnsError(t *testing.T) {
	rootCmd.SetArgs([]string{"completion", "invalid"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitError), code)
}

func TestCompletionTooManyArgsReturnsError(t *testing.T) {
	rootCmd.SetArgs([]string{"completion", "bash", "extra"})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitError), code)
}

func TestCompletionHelpContainsInstallInstructions(t *testing.T) {
	long := completionCmd.Long

	instructions := []string{
		"harvx completion bash",
		"/etc/bash_completion.d/harvx",
		"harvx completion zsh",
		`"${fpath[1]}/_harvx"`,
		"harvx completion fish",
		"~/.config/fish/completions/harvx.fish",
		"harvx completion powershell",
		"Out-String | Invoke-Expression",
	}

	for _, inst := range instructions {
		assert.Contains(t, long, inst,
			"Long help must contain installation instruction: %s", inst)
	}
}

func TestFormatFlagCompletion(t *testing.T) {
	values, directive := completeFormat(nil, nil, "")

	require.Len(t, values, 2)
	assert.Contains(t, values, "markdown")
	assert.Contains(t, values, "xml")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestTargetFlagCompletion(t *testing.T) {
	values, directive := completeTarget(nil, nil, "")

	require.Len(t, values, 3)
	assert.Contains(t, values, "claude")
	assert.Contains(t, values, "chatgpt")
	assert.Contains(t, values, "generic")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestSubcommandNamesComplete(t *testing.T) {
	// Verify that all expected subcommands are registered and would appear
	// in tab completion. Cobra handles subcommand completion automatically
	// when commands are registered.
	expectedSubcommands := []string{"generate", "version", "completion"}
	for _, name := range expectedSubcommands {
		t.Run(name, func(t *testing.T) {
			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == name {
					found = true
					break
				}
			}
			assert.True(t, found, "subcommand %q must be registered for tab completion", name)
		})
	}
}
