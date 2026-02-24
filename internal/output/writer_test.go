package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalRenderData returns a minimal RenderData suitable for tests.
func minimalRenderData() *RenderData {
	return &RenderData{
		ProjectName:   "test-project",
		Timestamp:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		ContentHash:   "abcdef1234567890",
		ProfileName:   "default",
		TokenizerName: "cl100k_base",
		TotalTokens:   42,
		TotalFiles:    1,
		Files: []FileRenderEntry{
			{
				Path:       "main.go",
				Size:       100,
				TokenCount: 42,
				Tier:       0,
				TierLabel:  "critical",
				Language:   "go",
				Content:    "package main",
			},
		},
		TreeString: ".",
		TierCounts: map[int]int{0: 1},
	}
}

func TestNewOutputWriter(t *testing.T) {
	t.Parallel()

	ow := NewOutputWriter()
	require.NotNil(t, ow)
	assert.Equal(t, os.Stdout, ow.stdout)
	assert.Equal(t, os.Stderr, ow.stderr)
}

func TestNewOutputWriterWithStreams(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	require.NotNil(t, ow)
	assert.Equal(t, &stdout, ow.stdout)
	assert.Equal(t, &stderr, ow.stderr)
}

func TestOutputWriter_Write_NilData(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	result, err := ow.Write(context.Background(), nil, OutputOpts{
		Format: "markdown",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render data is nil")
}

func TestOutputWriter_Write_InvalidFormat(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	result, err := ow.Write(context.Background(), minimalRenderData(), OutputOpts{
		Format: "json",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestOutputWriter_Write_CancelledContext(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := ow.Write(ctx, minimalRenderData(), OutputOpts{
		Format: "markdown",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestOutputWriter_Write_StdoutMarkdown(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		Format:    "markdown",
		UseStdout: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Path)
	assert.NotZero(t, result.Hash)
	assert.NotEmpty(t, result.HashHex)
	assert.Equal(t, 16, len(result.HashHex))
	assert.Equal(t, data.TotalTokens, result.TotalTokens)
	assert.Greater(t, result.BytesWritten, int64(0))

	// Verify content was written to stdout.
	output := stdout.String()
	assert.Contains(t, output, "# Harvx Context: test-project")
	assert.Contains(t, output, "main.go")
}

func TestOutputWriter_Write_StdoutXML(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		Format:    "xml",
		UseStdout: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Empty(t, result.Path)
	assert.NotZero(t, result.Hash)

	output := stdout.String()
	assert.Contains(t, output, "<repository>")
	assert.Contains(t, output, "test-project")
}

func TestOutputWriter_Write_FileMarkdown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "markdown",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, outPath, result.Path)
	assert.NotZero(t, result.Hash)
	assert.NotEmpty(t, result.HashHex)
	assert.Equal(t, data.TotalTokens, result.TotalTokens)
	assert.Greater(t, result.BytesWritten, int64(0))

	// Verify the file was created with content.
	content, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Harvx Context: test-project")

	// Verify no temp files remain.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".harvx-"), "temp file should be cleaned up: %s", e.Name())
	}
}

func TestOutputWriter_Write_FileXML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.xml")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "xml",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, outPath, result.Path)

	content, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "<repository>")
}

func TestOutputWriter_Write_FileDefaultPath(t *testing.T) {
	// Not parallel: os.Chdir affects the entire process.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		Format: "markdown",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "harvx-output.md", result.Path)

	// Clean up the created file.
	os.Remove(filepath.Join(dir, "harvx-output.md"))
}

func TestOutputWriter_Write_FileProfileOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	profilePath := filepath.Join(dir, "custom-output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		ProfileOutput: profilePath,
		Format:        "markdown",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, profilePath, result.Path)
}

func TestOutputWriter_Write_CLIOverridesProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "cli-output.md")
	profilePath := filepath.Join(dir, "profile-output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath:    cliPath,
		ProfileOutput: profilePath,
		Format:        "markdown",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, cliPath, result.Path)

	// Verify profile path was NOT created.
	_, err = os.Stat(profilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestOutputWriter_Write_OutputDirNotExist(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: "/nonexistent/dir/output.md",
		Format:     "markdown",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output directory")
}

func TestOutputWriter_Write_AtomicNoPartialFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	_, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "markdown",
	})
	require.NoError(t, err)

	// Verify no temp files remain in the directory.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".harvx-"),
			"temp file should not remain: %s", e.Name())
	}
}

func TestOutputWriter_Write_HashConsistency(t *testing.T) {
	t.Parallel()

	data := minimalRenderData()

	// Write twice and verify hashes match.
	var stdout1, stdout2, stderr bytes.Buffer
	ow1 := NewOutputWriterWithStreams(&stdout1, &stderr)
	ow2 := NewOutputWriterWithStreams(&stdout2, &stderr)

	result1, err := ow1.Write(context.Background(), data, OutputOpts{
		Format:    "markdown",
		UseStdout: true,
	})
	require.NoError(t, err)

	result2, err := ow2.Write(context.Background(), data, OutputOpts{
		Format:    "markdown",
		UseStdout: true,
	})
	require.NoError(t, err)

	assert.Equal(t, result1.Hash, result2.Hash)
	assert.Equal(t, result1.HashHex, result2.HashHex)
	assert.Equal(t, result1.BytesWritten, result2.BytesWritten)
}

func TestOutputWriter_Write_FileHashMatchesStdout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "output.md")
	data := minimalRenderData()

	// Write to file.
	var stdout1, stderr1 bytes.Buffer
	owFile := NewOutputWriterWithStreams(&stdout1, &stderr1)
	resultFile, err := owFile.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "markdown",
	})
	require.NoError(t, err)

	// Write to stdout.
	var stdout2, stderr2 bytes.Buffer
	owStdout := NewOutputWriterWithStreams(&stdout2, &stderr2)
	resultStdout, err := owStdout.Write(context.Background(), data, OutputOpts{
		Format:    "markdown",
		UseStdout: true,
	})
	require.NoError(t, err)

	// Hashes should match since the same data is rendered.
	assert.Equal(t, resultFile.Hash, resultStdout.Hash)
	assert.Equal(t, resultFile.HashHex, resultStdout.HashHex)
	assert.Equal(t, resultFile.BytesWritten, resultStdout.BytesWritten)
}

func TestOutputWriter_Write_StdoutDoesNotCreateFile(t *testing.T) {
	// Not parallel: os.Chdir affects the entire process.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		os.Chdir(origDir)
	})

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		Format:    "markdown",
		UseStdout: true,
	})
	require.NoError(t, err)
	assert.Empty(t, result.Path)

	// Verify no file was created.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "stdout mode should not create any files")
}

func TestCountingWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		writes  []string
		wantN   int64
		wantOut string
	}{
		{
			name:    "single write",
			writes:  []string{"hello"},
			wantN:   5,
			wantOut: "hello",
		},
		{
			name:    "multiple writes",
			writes:  []string{"hello", " ", "world"},
			wantN:   11,
			wantOut: "hello world",
		},
		{
			name:    "empty write",
			writes:  []string{""},
			wantN:   0,
			wantOut: "",
		},
		{
			name:    "no writes",
			writes:  nil,
			wantN:   0,
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			cw := &countingWriter{w: &buf}

			for _, s := range tt.writes {
				n, err := cw.Write([]byte(s))
				require.NoError(t, err)
				assert.Equal(t, len(s), n)
			}

			assert.Equal(t, tt.wantN, cw.written)
			assert.Equal(t, tt.wantOut, buf.String())
		})
	}
}

func TestOutputWriter_Write_AppendExtensionNoExt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "myoutput")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "markdown",
	})
	require.NoError(t, err)

	// ResolveOutputPath should have appended .md.
	assert.Equal(t, outPath+".md", result.Path)
}

func TestOutputWriter_Write_AppendXMLExtension(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "myoutput")

	var stdout, stderr bytes.Buffer
	ow := NewOutputWriterWithStreams(&stdout, &stderr)

	data := minimalRenderData()
	result, err := ow.Write(context.Background(), data, OutputOpts{
		OutputPath: outPath,
		Format:     "xml",
	})
	require.NoError(t, err)

	assert.Equal(t, outPath+".xml", result.Path)
}
