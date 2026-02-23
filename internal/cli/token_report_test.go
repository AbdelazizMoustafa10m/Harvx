package cli

import (
	"bytes"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestFD creates a FileDescriptor for testing.
func makeTestFD(path string, tokenCount, tier int) *pipeline.FileDescriptor {
	return &pipeline.FileDescriptor{
		Path:       path,
		TokenCount: tokenCount,
		Tier:       tier,
	}
}

func TestPrintTokenReport_WritesToWriter(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeTestFD("main.go", 500, 1),
		makeTestFD("config.toml", 100, 0),
	}

	var buf bytes.Buffer
	PrintTokenReport(&buf, files, "cl100k_base", 0)

	out := buf.String()
	assert.Contains(t, out, "Token Report (cl100k_base)")
	assert.Contains(t, out, "Total files:  2")
	assert.Contains(t, out, "Total tokens: 600")
}

func TestPrintTokenReport_EmptyFiles(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	PrintTokenReport(&buf, nil, "none", 0)

	out := buf.String()
	assert.Contains(t, out, "Token Report (none)")
	assert.Contains(t, out, "Total files:  0")
}

func TestPrintTopFiles_WritesToWriter(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeTestFD("a.go", 200, 1),
		makeTestFD("b.go", 800, 1),
		makeTestFD("c.go", 50, 2),
	}

	var buf bytes.Buffer
	PrintTopFiles(&buf, files, 2)

	out := buf.String()
	assert.Contains(t, out, "Top 2 Files by Token Count:")
	// b.go has the most tokens and should be first.
	assert.Contains(t, out, "b.go")
	// c.go should be excluded by the limit of 2.
	assert.NotContains(t, out, "c.go")
}

func TestPrintTopFiles_EmptyFiles(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	PrintTopFiles(&buf, nil, 10)

	out := buf.String()
	assert.Contains(t, out, "Top 10 Files by Token Count:")
	assert.Contains(t, out, "(no files)")
}

func TestPrintTopFiles_NZero_ShowsAll(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeTestFD("a.go", 200, 1),
		makeTestFD("b.go", 100, 1),
	}

	var buf bytes.Buffer
	PrintTopFiles(&buf, files, 0)

	out := buf.String()
	assert.Contains(t, out, "All Files by Token Count:")
	assert.Contains(t, out, "a.go")
	assert.Contains(t, out, "b.go")
}

// TestPrintTopFiles_ExactlyFive verifies that passing n=5 to PrintTopFiles
// with more than 5 files yields exactly 5 lines of output entries.
func TestPrintTopFiles_ExactlyFive(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeTestFD("a.go", 100, 1),
		makeTestFD("b.go", 600, 1),
		makeTestFD("c.go", 300, 2),
		makeTestFD("d.go", 900, 1),
		makeTestFD("e.go", 50, 3),
		makeTestFD("f.go", 750, 1),
		makeTestFD("g.go", 400, 2),
	}

	var buf bytes.Buffer
	PrintTopFiles(&buf, files, 5)

	out := buf.String()
	assert.Contains(t, out, "Top 5 Files by Token Count:")
	// Count rank entries: " 1.", " 2.", ... " 5." should be present.
	assert.Contains(t, out, " 1.")
	assert.Contains(t, out, " 5.")
	// " 6." must not appear -- only 5 files are shown.
	assert.NotContains(t, out, " 6.")
	// d.go (900) must be ranked first.
	assert.Contains(t, out, "d.go")
	// e.go (50) is the smallest and must be excluded.
	assert.NotContains(t, out, "e.go")
}

// TestPrintTopFiles_FewerThanN verifies that when the pool has fewer files
// than N, all files are printed and no index error occurs.
func TestPrintTopFiles_FewerThanN(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeTestFD("only.go", 300, 1),
		makeTestFD("two.go", 100, 2),
	}

	var buf bytes.Buffer
	PrintTopFiles(&buf, files, 5)

	out := buf.String()
	assert.Contains(t, out, "Top 5 Files by Token Count:")
	assert.Contains(t, out, "only.go")
	assert.Contains(t, out, "two.go")
	// No " 3." entry should appear since only 2 files exist.
	assert.NotContains(t, out, " 3.")
}

// --- Integration: --token-count CLI flag wiring ---

// TestTokenCountFlagIsRegistered verifies that the --token-count flag is
// registered on the root command and accessible to subcommands. This tests
// the flag wiring from the T-033 acceptance criteria.
func TestTokenCountFlagIsRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("token-count")
	require.NotNil(t, flag, "--token-count flag must be registered on root command")
	assert.Equal(t, "false", flag.DefValue, "--token-count default must be false")
}

// TestTokenCountFlagAccessibleOnGenerate verifies that --token-count is
// accessible as an inherited flag on the generate subcommand.
func TestTokenCountFlagAccessibleOnGenerate(t *testing.T) {
	flag := generateCmd.InheritedFlags().Lookup("token-count")
	assert.NotNil(t, flag, "generate must inherit --token-count from root")
}

// TestTokenCountCommandExitsZero verifies that running harvx with
// --token-count produces exit code 0. The pipeline is a stub so no output
// file is generated; the flag acceptance criterion requires only exit 0.
func TestTokenCountCommandExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"--token-count"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx --token-count must exit 0; stderr: %s", buf.String())
}

// TestTopFilesFlagIsRegistered verifies that --top-files is registered on the
// root command with the correct default value (0 = disabled).
func TestTopFilesFlagIsRegistered(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("top-files")
	require.NotNil(t, flag, "--top-files flag must be registered on root command")
	assert.Equal(t, "0", flag.DefValue, "--top-files default must be 0")
}

// TestTopFilesCommandExitsZero verifies that running harvx with --top-files 5
// produces exit code 0.
func TestTopFilesCommandExitsZero(t *testing.T) {
	rootCmd.SetArgs([]string{"--top-files", "5"})
	defer rootCmd.SetArgs(nil)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	defer rootCmd.SetOut(nil)
	defer rootCmd.SetErr(nil)

	code := Execute()
	assert.Equal(t, int(pipeline.ExitSuccess), code,
		"harvx --top-files 5 must exit 0; stderr: %s", buf.String())
}
