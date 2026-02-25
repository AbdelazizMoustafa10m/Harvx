package workflows

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyOutput_ExactMatch(t *testing.T) {
	t.Parallel()

	// Set up a temp directory with a source file.
	root := t.TempDir()
	srcContent := "package main\n\nfunc main() {}\n"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "main.go"), []byte(srcContent), 0644))

	// Create a Markdown output file that matches the source. Trailing newline
	// differences between fenced content extraction and the original source
	// are normalized during verification.
	outputContent := `# Harvx Context: test

## Files

### ` + "`src/main.go`" + `

> **Size:** 33 B | **Tokens:** 8 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main

func main() {}
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.TotalFiles)
	assert.Equal(t, 1, result.SampledFiles)
	assert.Equal(t, 1, result.PassedCount)
	assert.Equal(t, 0, result.WarningCount)
	require.Len(t, result.Files, 1)
	assert.Equal(t, VerifyMatch, result.Files[0].Status)
	assert.Equal(t, "Match", result.Files[0].Message)
}

func TestVerifyOutput_RedactionDiff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcContent := "api_key = \"sk-1234567890abcdef\"\nother = \"value\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.toml"), []byte(srcContent), 0644))

	// Create output with redacted content.
	outputContent := `# Harvx Context: test

## Files

### ` + "`config.toml`" + `

> **Size:** 50 B | **Tokens:** 12 | **Tier:** primary | **Compressed:** no

` + "```toml" + `
api_key = [REDACTED:api_key]
other = "value"
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	assert.Equal(t, VerifyRedactionDiff, result.Files[0].Status)
	assert.Contains(t, result.Files[0].Message, "1 redactions applied")
	assert.Equal(t, 1, result.Files[0].Redactions)
}

func TestVerifyOutput_CompressionDiff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcContent := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte(srcContent), 0644))

	outputContent := `# Harvx Context: test

## Files

### ` + "`main.go`" + `

> **Size:** 50 B | **Tokens:** 12 | **Tier:** primary | **Compressed:** yes

` + "```go" + `
func main()
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	assert.Equal(t, VerifyCompressionDiff, result.Files[0].Status)
	assert.Contains(t, result.Files[0].Message, "compressed: signatures only")
	assert.True(t, result.Files[0].Compressed)
}

func TestVerifyOutput_UnexpectedDiff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcContent := "package main\n\nfunc main() {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte(srcContent), 0644))

	// Output has different content than the source.
	outputContent := `# Harvx Context: test

## Files

### ` + "`main.go`" + `

> **Size:** 33 B | **Tokens:** 8 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main

func differentFunc() {}
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	assert.Equal(t, VerifyUnexpectedDiff, result.Files[0].Status)
	assert.NotEmpty(t, result.Files[0].DiffLines)
}

func TestVerifyOutput_FileChanged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Don't create the source file -- simulate deletion.

	outputContent := `# Harvx Context: test

## Files

### ` + "`deleted.go`" + `

> **Size:** 33 B | **Tokens:** 8 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.Len(t, result.Files, 1)
	assert.Equal(t, VerifyFileChanged, result.Files[0].Status)
	assert.Contains(t, result.Files[0].Message, "source file not found")
}

func TestVerifyOutput_SampleSize(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create 10 source files.
	for i := 0; i < 10; i++ {
		name := filepath.Join(root, "file"+string(rune('a'+i))+".go")
		content := "package main\n"
		require.NoError(t, os.WriteFile(name, []byte(content), 0644))
	}

	// Create output with 10 files.
	var b strings.Builder
	b.WriteString("# Harvx Context: test\n\n## Files\n")
	for i := 0; i < 10; i++ {
		name := "file" + string(rune('a'+i)) + ".go"
		b.WriteString("\n### `" + name + "`\n\n")
		b.WriteString("> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no\n\n")
		b.WriteString("```go\npackage main\n```\n")
	}

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(b.String()), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
		SampleSize: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, 10, result.TotalFiles)
	assert.Equal(t, 5, result.SampledFiles)
	assert.Len(t, result.Files, 5)
}

func TestVerifyOutput_SampleSizeLargerThanFileCount(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "only.go"), []byte("package main\n"), 0644))

	outputContent := `# Harvx Context: test

## Files

### ` + "`only.go`" + `

> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
		SampleSize: 100,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalFiles)
	assert.Equal(t, 1, result.SampledFiles)
	assert.Len(t, result.Files, 1)
}

func TestVerifyOutput_ReproducibleSampling(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for i := 0; i < 20; i++ {
		name := filepath.Join(root, "file"+string(rune('a'+i))+".go")
		require.NoError(t, os.WriteFile(name, []byte("package main\n"), 0644))
	}

	var b strings.Builder
	b.WriteString("# Harvx Context: test\n\n## Files\n")
	for i := 0; i < 20; i++ {
		name := "file" + string(rune('a'+i)) + ".go"
		b.WriteString("\n### `" + name + "`\n\n")
		b.WriteString("> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no\n\n")
		b.WriteString("```go\npackage main\n```\n")
	}

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(b.String()), 0644))

	// Run verification twice with same sample size.
	result1, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
		SampleSize: 5,
	})
	require.NoError(t, err)

	result2, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
		SampleSize: 5,
	})
	require.NoError(t, err)

	// Same content hash should produce the same sample set.
	require.Len(t, result1.Files, 5)
	require.Len(t, result2.Files, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, result1.Files[i].Path, result2.Files[i].Path)
	}
}

func TestVerifyOutput_SpecificPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "a.go"), []byte("package a\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "b.go"), []byte("package b\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "c.go"), []byte("package c\n"), 0644))

	var b strings.Builder
	b.WriteString("# Harvx Context: test\n\n## Files\n")
	for _, name := range []string{"src/a.go", "src/b.go", "src/c.go"} {
		b.WriteString("\n### `" + name + "`\n\n")
		b.WriteString("> **Size:** 10 B | **Tokens:** 2 | **Tier:** primary | **Compressed:** no\n\n")
		base := filepath.Base(name)
		pkg := base[:len(base)-3] // strip .go
		b.WriteString("```go\npackage " + pkg + "\n```\n")
	}

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(b.String()), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
		Paths:      []string{"src/a.go", "src/c.go"},
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalFiles)
	assert.Equal(t, 2, result.SampledFiles)
	require.Len(t, result.Files, 2)

	paths := make(map[string]bool)
	for _, f := range result.Files {
		paths[f.Path] = true
	}
	assert.True(t, paths["src/a.go"])
	assert.True(t, paths["src/c.go"])
}

func TestVerifyOutput_OutputNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	_, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: filepath.Join(root, "nonexistent.md"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading output file")
	assert.Contains(t, err.Error(), "--output")
}

func TestVerifyOutput_EmptyOutputPath(t *testing.T) {
	t.Parallel()

	_, err := VerifyOutput(VerifyOptions{
		RootDir: "/tmp",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "output path required")
}

func TestVerifyOutput_EmptyRootDir(t *testing.T) {
	t.Parallel()

	_, err := VerifyOutput(VerifyOptions{
		OutputPath: "/tmp/test.md",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "root directory required")
}

func TestVerifyOutput_BudgetInfo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644))

	outputContent := `# Harvx Context: test

## Files

### ` + "`main.go`" + `

> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	// Create metadata sidecar.
	budgetPct := 42.5
	meta := output.OutputMetadata{
		Version:   "1.0.0",
		Tokenizer: "cl100k_base",
		Statistics: output.Statistics{
			TotalFiles:        1,
			TotalTokens:       4250,
			MaxTokens:         10000,
			BudgetUsedPercent: &budgetPct,
			CompressedFiles:   0,
			RedactionsTotal:   0,
			FilesByTier:       map[string]int{},
			RedactionsByType:  map[string]int{},
		},
	}
	metaData, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outputPath+".meta.json", metaData, 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Budget)
	assert.Equal(t, "cl100k_base", result.Budget.Tokenizer)
	assert.Equal(t, 4250, result.Budget.TotalTokens)
	assert.Equal(t, 10000, result.Budget.MaxTokens)
	assert.InDelta(t, 42.5, result.Budget.BudgetUsedPct, 0.01)
}

func TestVerifyOutput_NoBudgetSidecar(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644))

	outputContent := `# Harvx Context: test

## Files

### ` + "`main.go`" + `

> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no

` + "```go" + `
package main
` + "```" + `
`

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(outputContent), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	assert.Nil(t, result.Budget)
}

func TestSimpleDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		packed   string
		maxLines int
		wantLen  int
	}{
		{
			name:     "identical content",
			original: "line1\nline2\nline3",
			packed:   "line1\nline2\nline3",
			maxLines: 10,
			wantLen:  0,
		},
		{
			name:     "single line difference",
			original: "line1\nline2\nline3",
			packed:   "line1\nchanged\nline3",
			maxLines: 10,
			wantLen:  2, // - line2, + changed
		},
		{
			name:     "max lines respected",
			original: "a\nb\nc\nd\ne",
			packed:   "1\n2\n3\n4\n5",
			maxLines: 4,
			wantLen:  4,
		},
		{
			name:     "added lines",
			original: "a\nb",
			packed:   "a\nb\nc\nd",
			maxLines: 10,
			wantLen:  2, // + c, + d
		},
		{
			name:     "removed lines",
			original: "a\nb\nc\nd",
			packed:   "a\nb",
			maxLines: 10,
			wantLen:  2, // - c, - d
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			diff := simpleDiff(tt.original, tt.packed, tt.maxLines)
			assert.Len(t, diff, tt.wantLen)
		})
	}
}

func TestIsRedactionDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		packed   string
		want     bool
	}{
		{
			name:     "exact match is not redaction diff",
			original: "line1\nline2",
			packed:   "line1\nline2",
			want:     true, // no diff at all; isRedactionDiff returns true since there are 0 unexplained diffs
		},
		{
			name:     "single redaction",
			original: "api_key = \"secret123\"\nother = \"value\"",
			packed:   "api_key = [REDACTED:api_key]\nother = \"value\"",
			want:     true,
		},
		{
			name:     "multiple redactions",
			original: "key1 = \"secret1\"\nkey2 = \"secret2\"",
			packed:   "key1 = [REDACTED:key1]\nkey2 = [REDACTED:key2]",
			want:     true,
		},
		{
			name:     "completely different content",
			original: "line1\nline2",
			packed:   "completely\ndifferent",
			want:     false,
		},
		{
			name:     "different line count",
			original: "line1\nline2\nline3\nline4",
			packed:   "line1",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isRedactionDiff(tt.original, tt.packed)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyOutput_MultipleStatuses(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// File that matches exactly.
	require.NoError(t, os.WriteFile(filepath.Join(root, "match.go"), []byte("package main\n"), 0644))
	// File that was compressed.
	require.NoError(t, os.WriteFile(filepath.Join(root, "compressed.go"), []byte("package main\n\nfunc foo() {}\n"), 0644))
	// File that was deleted.
	// (deleted.go does not exist on disk)

	var b strings.Builder
	b.WriteString("# Harvx Context: test\n\n## Files\n")

	// Match file.
	b.WriteString("\n### `match.go`\n\n")
	b.WriteString("> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no\n\n")
	b.WriteString("```go\npackage main\n```\n")

	// Compressed file.
	b.WriteString("\n### `compressed.go`\n\n")
	b.WriteString("> **Size:** 30 B | **Tokens:** 7 | **Tier:** primary | **Compressed:** yes\n\n")
	b.WriteString("```go\nfunc foo()\n```\n")

	// Deleted file.
	b.WriteString("\n### `deleted.go`\n\n")
	b.WriteString("> **Size:** 14 B | **Tokens:** 3 | **Tier:** primary | **Compressed:** no\n\n")
	b.WriteString("```go\npackage gone\n```\n")

	outputPath := filepath.Join(root, "harvx-output.md")
	require.NoError(t, os.WriteFile(outputPath, []byte(b.String()), 0644))

	result, err := VerifyOutput(VerifyOptions{
		RootDir:    root,
		OutputPath: outputPath,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalFiles)
	assert.Equal(t, 3, result.SampledFiles)
	assert.Equal(t, 2, result.PassedCount)  // match + compressed
	assert.Equal(t, 1, result.WarningCount) // deleted

	statusMap := make(map[string]VerifyStatus)
	for _, f := range result.Files {
		statusMap[f.Path] = f.Status
	}

	assert.Equal(t, VerifyMatch, statusMap["match.go"])
	assert.Equal(t, VerifyCompressionDiff, statusMap["compressed.go"])
	assert.Equal(t, VerifyFileChanged, statusMap["deleted.go"])
}
