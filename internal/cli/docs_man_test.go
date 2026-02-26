package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "docs" {
			found = true
			break
		}
	}
	assert.True(t, found, "docs subcommand must be registered on root command")
}

func TestDocsCommandIsHidden(t *testing.T) {
	assert.True(t, docsCmd.Hidden, "docs command must be hidden")
}

func TestDocsManSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range docsCmd.Commands() {
		if cmd.Name() == "man" {
			found = true
			break
		}
	}
	assert.True(t, found, "man subcommand must be registered on docs command")
}

func TestDocsManGeneratesManPages(t *testing.T) {
	outputDir := t.TempDir()

	rootCmd.SetArgs([]string{"docs", "man", "--output-dir", outputDir})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code, "docs man should succeed")

	output := buf.String()
	assert.Contains(t, output, "Generated", "output must report generation count")
	assert.Contains(t, output, "man pages", "output must mention man pages")

	// Verify that man pages were actually created.
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "man pages directory must not be empty")

	// The root command should produce a man page file.
	foundRoot := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "harvx") {
			foundRoot = true
			break
		}
	}
	assert.True(t, foundRoot, "must generate a man page for the root harvx command")
}

func TestDocsManGeneratesPageForSubcommands(t *testing.T) {
	outputDir := t.TempDir()

	rootCmd.SetArgs([]string{"docs", "man", "--output-dir", outputDir})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	// Collect all generated file names.
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)

	fileNames := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			fileNames = append(fileNames, e.Name())
		}
	}

	// At minimum, we expect man pages for several key commands.
	// Cobra generates files like harvx.1, harvx-generate.1, etc.
	expectedSubstrings := []string{
		"harvx",     // root command
		"generate",  // generate subcommand
		"version",   // version subcommand
		"completion", // completion subcommand
	}

	allFiles := strings.Join(fileNames, " ")
	for _, sub := range expectedSubstrings {
		assert.Contains(t, allFiles, sub,
			"man pages should include a page containing %q", sub)
	}
}

func TestDocsManPageContent(t *testing.T) {
	outputDir := t.TempDir()

	rootCmd.SetArgs([]string{"docs", "man", "--output-dir", outputDir})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	require.Equal(t, int(pipeline.ExitSuccess), code)

	// Read the root man page and verify it contains expected troff directives.
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "harvx.") && !e.IsDir() {
			content, err := os.ReadFile(filepath.Join(outputDir, e.Name()))
			require.NoError(t, err)

			text := string(content)
			// Man pages in troff format should contain section headers.
			assert.Contains(t, text, ".TH", "man page must contain .TH header")
			assert.Contains(t, text, "HARVX", "man page must reference HARVX title")
			break
		}
	}
}

func TestDocsManCreatesOutputDirectory(t *testing.T) {
	base := t.TempDir()
	outputDir := filepath.Join(base, "nested", "man", "pages")

	rootCmd.SetArgs([]string{"docs", "man", "--output-dir", outputDir})
	defer rootCmd.SetArgs(nil)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code)

	// Verify the nested directory was created.
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "output directory must be created")
}

func TestCompressEngineFlagCompletion(t *testing.T) {
	values, directive := completeCompressEngine(nil, nil, "")

	require.Len(t, values, 3)
	assert.Contains(t, values, "ast")
	assert.Contains(t, values, "regex")
	assert.Contains(t, values, "auto")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestProfileFlagCompletionRegistered(t *testing.T) {
	// Verify that the --profile flag has a completion function registered
	// by checking that the root command's flag completion functions map
	// includes "profile". We test this indirectly by verifying the
	// completeProfileNames function is callable and returns the expected
	// directive.
	names, directive := completeProfileNames(nil, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	// At minimum, the built-in "default" profile should always be present.
	assert.Contains(t, names, "default",
		"profile completion must include the built-in 'default' profile")
}
