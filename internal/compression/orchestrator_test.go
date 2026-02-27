package compression

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: create a CompressionConfig suitable for tests.
// ---------------------------------------------------------------------------

func testConfig() CompressionConfig {
	return CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    runtime.NumCPU(),
	}
}

// ---------------------------------------------------------------------------
// TypeScript compression
// ---------------------------------------------------------------------------

func TestOrchestratorCompressTypeScript(t *testing.T) {
	orch := NewOrchestrator(testConfig())
	files := []*CompressibleFile{
		{
			Path:    "src/hello.ts",
			Content: "export function hello(): string {\n  return \"hello\";\n}\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	f := files[0]
	assert.True(t, f.IsCompressed, "file should be marked as compressed")
	assert.Equal(t, "typescript", f.Language)
	assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
		"compressed content should start with CompressedMarker")
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesSkipped))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesFailed))
}

// ---------------------------------------------------------------------------
// Go compression
// ---------------------------------------------------------------------------

func TestOrchestratorCompressGo(t *testing.T) {
	orch := NewOrchestrator(testConfig())
	files := []*CompressibleFile{
		{
			Path:    "pkg/greet.go",
			Content: "package greet\n\nfunc Hello() string {\n\treturn \"hello\"\n}\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	f := files[0]
	assert.True(t, f.IsCompressed, "file should be marked as compressed")
	assert.Equal(t, "go", f.Language)
	assert.True(t, strings.HasPrefix(f.Content, CompressedMarker))
	assert.Contains(t, f.Content, "func Hello() string")
	// Function body should not appear in compressed output.
	assert.NotContains(t, f.Content, `return "hello"`)
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
}

// ---------------------------------------------------------------------------
// Unsupported language (skip)
// ---------------------------------------------------------------------------

func TestOrchestratorUnsupportedLanguage(t *testing.T) {
	orch := NewOrchestrator(testConfig())
	originalContent := "# This is a Markdown file\n\nSome text here.\n"
	files := []*CompressibleFile{
		{
			Path:    "docs/README.md",
			Content: originalContent,
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	f := files[0]
	assert.False(t, f.IsCompressed, "unsupported file should not be marked compressed")
	assert.Equal(t, originalContent, f.Content, "content should be unchanged for unsupported language")
	assert.Equal(t, "", f.Language, "language should remain empty for unsupported files")
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesCompressed))
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesSkipped))
}

// ---------------------------------------------------------------------------
// CompressedMarker format
// ---------------------------------------------------------------------------

func TestOrchestratorCompressedMarker(t *testing.T) {
	orch := NewOrchestrator(testConfig())
	files := []*CompressibleFile{
		{
			Path:    "app.go",
			Content: "package app\n\nfunc Run() {\n\tfmt.Println(\"running\")\n}\n",
		},
	}

	_, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	f := files[0]
	require.True(t, f.IsCompressed)

	// The compressed content must start with the marker followed by a newline.
	expectedPrefix := CompressedMarker + "\n"
	assert.True(t, strings.HasPrefix(f.Content, expectedPrefix),
		"content should start with %q but got prefix: %q",
		expectedPrefix, f.Content[:min(len(expectedPrefix)+10, len(f.Content))])

	// Verify the marker constant value.
	assert.Equal(t, "<!-- Compressed: signatures only -->", CompressedMarker)
}

// ---------------------------------------------------------------------------
// Timeout handling
// ---------------------------------------------------------------------------

func TestOrchestratorTimeout(t *testing.T) {
	// Use an extremely short timeout to provoke timeout behavior.
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 1 * time.Nanosecond,
		Concurrency:    1,
	}
	orch := NewOrchestrator(cfg)

	// Use a Go file with enough content to potentially trigger timeout.
	var b strings.Builder
	b.WriteString("package main\n\n")
	for i := 0; i < 100; i++ {
		b.WriteString(fmt.Sprintf("func Func%d() {\n\t// body\n}\n\n", i))
	}
	originalContent := b.String()

	files := []*CompressibleFile{
		{
			Path:    "big.go",
			Content: originalContent,
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err, "timeout should not return a fatal error")

	// The file either timed out (original content preserved) or succeeded
	// (compressed). Both are valid outcomes depending on system speed.
	timedOut := atomic.LoadInt64(&stats.FilesTimedOut)
	compressed := atomic.LoadInt64(&stats.FilesCompressed)
	assert.Equal(t, int64(1), timedOut+compressed,
		"file should be either timed out or compressed, got timedOut=%d compressed=%d",
		timedOut, compressed)

	if timedOut > 0 {
		assert.Equal(t, originalContent, files[0].Content,
			"timed-out file should retain original content")
		assert.False(t, files[0].IsCompressed)
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestOrchestratorContextCancellation(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before calling Compress.

	files := []*CompressibleFile{
		{
			Path:    "main.go",
			Content: "package main\n\nfunc main() {}\n",
		},
	}

	_, err := orch.Compress(ctx, files)
	assert.Error(t, err, "should return error on cancelled context")
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Stats accumulation across multiple files
// ---------------------------------------------------------------------------

func TestOrchestratorStatsAccumulation(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	files := []*CompressibleFile{
		{
			Path:    "handler.ts",
			Content: "export function handler(): void {\n  console.log('handler');\n}\n",
		},
		{
			Path:    "server.go",
			Content: "package server\n\nfunc Serve() {\n\tfmt.Println(\"serving\")\n}\n",
		},
		{
			Path:    "README.md",
			Content: "# README\n\nThis is a readme.\n",
		},
		{
			Path:    "app.py",
			Content: "def main():\n    print('hello')\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	total := stats.TotalFiles()
	assert.Equal(t, int64(len(files)), total,
		"TotalFiles should equal the number of input files")

	// README.md is unsupported and should be skipped.
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesSkipped),
		"exactly one file should be skipped (README.md)")

	// The remaining 3 files should be compressed.
	assert.Equal(t, int64(3), atomic.LoadInt64(&stats.FilesCompressed),
		"three files should be compressed (.ts, .go, .py)")
}

// ---------------------------------------------------------------------------
// Mixed languages
// ---------------------------------------------------------------------------

func TestOrchestratorMixedLanguages(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	files := []*CompressibleFile{
		{
			Path:    "utils.ts",
			Content: "export function add(a: number, b: number): number {\n  return a + b;\n}\n",
		},
		{
			Path:    "lib.go",
			Content: "package lib\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n",
		},
		{
			Path:    "config.json",
			Content: `{"key": "value", "nested": {"a": 1}}`,
		},
		{
			Path:    "notes.md",
			Content: "# Notes\n\nSome notes.\n",
		},
		{
			Path:    "script.py",
			Content: "def greet(name: str) -> str:\n    return f'Hello, {name}'\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	// Verify per-file outcomes.
	tests := []struct {
		index        int
		wantCompress bool
		wantLang     string
	}{
		{0, true, "typescript"},
		{1, true, "go"},
		{2, true, "json"},
		{3, false, ""},  // .md unsupported
		{4, true, "python"},
	}

	for _, tt := range tests {
		f := files[tt.index]
		t.Run(f.Path, func(t *testing.T) {
			assert.Equal(t, tt.wantCompress, f.IsCompressed,
				"IsCompressed mismatch for %s", f.Path)
			if tt.wantCompress {
				assert.Equal(t, tt.wantLang, f.Language)
				assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
					"compressed file %s should have marker prefix", f.Path)
			} else {
				assert.Equal(t, tt.wantLang, f.Language)
			}
		})
	}

	assert.Equal(t, int64(4), atomic.LoadInt64(&stats.FilesCompressed))
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesSkipped))
	assert.Equal(t, int64(5), stats.TotalFiles())
}

// ---------------------------------------------------------------------------
// Concurrent race safety
// ---------------------------------------------------------------------------

func TestOrchestratorConcurrentRaceSafety(t *testing.T) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    8,
	}
	orch := NewOrchestrator(cfg)

	const fileCount = 30
	files := make([]*CompressibleFile, fileCount)
	for i := 0; i < fileCount; i++ {
		files[i] = &CompressibleFile{
			Path:    fmt.Sprintf("pkg/file_%d.go", i),
			Content: fmt.Sprintf("package pkg\n\nfunc Func%d(x int) int {\n\treturn x + %d\n}\n", i, i),
		}
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.Equal(t, int64(fileCount), stats.TotalFiles(),
		"all files should be processed")
	assert.Equal(t, int64(fileCount), atomic.LoadInt64(&stats.FilesCompressed),
		"all Go files should be compressed")

	// Verify every file was actually processed.
	for i, f := range files {
		assert.True(t, f.IsCompressed, "file %d should be compressed", i)
		assert.Equal(t, "go", f.Language, "file %d should have language 'go'", i)
		assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
			"file %d should have compressed marker", i)
	}
}

// ---------------------------------------------------------------------------
// Empty file list
// ---------------------------------------------------------------------------

func TestOrchestratorEmptyFiles(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	stats, err := orch.Compress(context.Background(), []*CompressibleFile{})
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.TotalFiles())
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesCompressed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesFailed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesSkipped))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesTimedOut))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.OriginalTokens))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.CompressedTokens))
}

func TestOrchestratorNilFiles(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	stats, err := orch.Compress(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.TotalFiles())
}

// ---------------------------------------------------------------------------
// Progress callback
// ---------------------------------------------------------------------------

func TestOrchestratorProgressCallback(t *testing.T) {
	// Use concurrency=1 to avoid race on lastTotal capture.
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    1,
	}
	orch := NewOrchestrator(cfg)

	var callCount int64
	var lastTotal int64

	orch.SetProgressFunc(func(current, total int) {
		atomic.AddInt64(&callCount, 1)
		atomic.StoreInt64(&lastTotal, int64(total))
	})

	files := []*CompressibleFile{
		{Path: "a.go", Content: "package a\n\nfunc A() {}\n"},
		{Path: "b.ts", Content: "export function b(): void {}\n"},
		{Path: "c.md", Content: "# Title\n"},
		{Path: "d.py", Content: "def d():\n    pass\n"},
		{Path: "e.json", Content: `{"e": true}`},
	}

	_, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.Equal(t, int64(len(files)), atomic.LoadInt64(&callCount),
		"progress callback should be called once per file")
	assert.Equal(t, int64(len(files)), atomic.LoadInt64(&lastTotal),
		"total in progress callback should match file count")
}

// ---------------------------------------------------------------------------
// Default config
// ---------------------------------------------------------------------------

func TestOrchestratorDefaultConfig(t *testing.T) {
	cfg := DefaultCompressionConfig()

	assert.False(t, cfg.Enabled, "default should be disabled")
	assert.Equal(t, 5*time.Second, cfg.TimeoutPerFile,
		"default timeout should be 5 seconds")
	assert.Equal(t, runtime.NumCPU(), cfg.Concurrency,
		"default concurrency should be runtime.NumCPU()")
}

// ---------------------------------------------------------------------------
// Token tracking
// ---------------------------------------------------------------------------

func TestOrchestratorTokenTracking(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	files := []*CompressibleFile{
		{
			Path:    "main.go",
			Content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n\tfmt.Println(\"goodbye world\")\n}\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)
	require.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))

	// The orchestrator tracks byte counts as a proxy for tokens.
	assert.Greater(t, atomic.LoadInt64(&stats.OriginalTokens), int64(0),
		"OriginalTokens should be positive after compression")
	assert.Greater(t, atomic.LoadInt64(&stats.CompressedTokens), int64(0),
		"CompressedTokens should be positive after compression")
	// Compressed output should be smaller (function body stripped).
	assert.Less(t, atomic.LoadInt64(&stats.CompressedTokens),
		atomic.LoadInt64(&stats.OriginalTokens),
		"compressed bytes should be less than original for Go files with bodies")
}

// ---------------------------------------------------------------------------
// TotalDuration is set
// ---------------------------------------------------------------------------

func TestOrchestratorTotalDuration(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	files := []*CompressibleFile{
		{Path: "x.go", Content: "package x\n\nfunc X() {}\n"},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.Greater(t, stats.TotalDuration, time.Duration(0),
		"TotalDuration should be positive")
}

// ---------------------------------------------------------------------------
// Supported language coverage via orchestrator
// ---------------------------------------------------------------------------

func TestOrchestratorAllSupportedLanguages(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	tests := []struct {
		path     string
		content  string
		wantLang string
	}{
		{
			path:     "app.ts",
			content:  "export function run(): void {\n  console.log('run');\n}\n",
			wantLang: "typescript",
		},
		{
			path:     "app.tsx",
			content:  "export function App(): JSX.Element {\n  return <div />;\n}\n",
			wantLang: "typescript",
		},
		{
			path:     "util.js",
			content:  "function util() {\n  return 42;\n}\n",
			wantLang: "javascript",
		},
		{
			path:     "util.jsx",
			content:  "function Component() {\n  return <span />;\n}\n",
			wantLang: "javascript",
		},
		{
			path:     "main.go",
			content:  "package main\n\nfunc main() {\n\tfmt.Println(\"go\")\n}\n",
			wantLang: "go",
		},
		{
			path:     "app.py",
			content:  "def main():\n    print('python')\n",
			wantLang: "python",
		},
		{
			path:     "lib.rs",
			content:  "pub fn greet() {\n    println!(\"rust\");\n}\n",
			wantLang: "rust",
		},
		{
			path:     "App.java",
			content:  "public class App {\n    public void run() {\n        System.out.println(\"java\");\n    }\n}\n",
			wantLang: "java",
		},
		{
			path:     "util.c",
			content:  "int add(int a, int b) {\n    return a + b;\n}\n",
			wantLang: "c",
		},
		{
			path:     "util.cpp",
			content:  "int add(int a, int b) {\n    return a + b;\n}\n",
			wantLang: "cpp",
		},
		{
			path:     "config.json",
			content:  `{"key": "value"}`,
			wantLang: "json",
		},
		{
			path:     "config.yaml",
			content:  "key: value\nnested:\n  a: 1\n",
			wantLang: "yaml",
		},
		{
			path:     "config.toml",
			content:  "[section]\nkey = \"value\"\n",
			wantLang: "toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			files := []*CompressibleFile{
				{Path: tt.path, Content: tt.content},
			}

			stats, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)

			f := files[0]
			assert.True(t, f.IsCompressed,
				"file %s should be compressed", tt.path)
			assert.Equal(t, tt.wantLang, f.Language)
			assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
				"file %s content should start with marker", tt.path)
			assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
		})
	}
}

// ---------------------------------------------------------------------------
// Orchestrator registers all 11 built-in compressors
// ---------------------------------------------------------------------------

func TestOrchestratorRegistersAllCompressors(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	expectedLangs := []string{
		"typescript", "javascript", "go", "python", "rust",
		"java", "c", "cpp", "json", "yaml", "toml",
	}

	langs := orch.registry.Languages()
	for _, expected := range expectedLangs {
		assert.Contains(t, langs, expected,
			"registry should contain compressor for %s", expected)
	}
	assert.Len(t, langs, len(expectedLangs),
		"registry should have exactly %d compressors", len(expectedLangs))
}

// ---------------------------------------------------------------------------
// Context cancellation mid-flight
// ---------------------------------------------------------------------------

func TestOrchestratorContextCancellationMidFlight(t *testing.T) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 10 * time.Second,
		Concurrency:    1, // Serialize to make cancellation predictable.
	}
	orch := NewOrchestrator(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after the first file is processed via progress callback.
	orch.SetProgressFunc(func(current, total int) {
		if current == 1 {
			cancel()
		}
	})

	files := make([]*CompressibleFile, 10)
	for i := range files {
		files[i] = &CompressibleFile{
			Path:    fmt.Sprintf("f%d.go", i),
			Content: fmt.Sprintf("package f\n\nfunc F%d() {\n\tfmt.Println(%d)\n}\n", i, i),
		}
	}

	_, err := orch.Compress(ctx, files)
	// After cancellation, we expect an error.
	assert.Error(t, err, "should propagate cancellation error")
}

// ---------------------------------------------------------------------------
// Single file with no content
// ---------------------------------------------------------------------------

func TestOrchestratorEmptyContentFile(t *testing.T) {
	orch := NewOrchestrator(testConfig())

	files := []*CompressibleFile{
		{
			Path:    "empty.go",
			Content: "",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	// The Go compressor handles empty input -- it returns an empty
	// CompressedOutput. An empty render produces "", which IsFallback
	// checks Language=="unknown" but Go returns "go". The orchestrator
	// should handle this gracefully.
	assert.Equal(t, int64(1), stats.TotalFiles())
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkOrchestratorCompress(b *testing.B) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    runtime.NumCPU(),
	}
	orch := NewOrchestrator(cfg)

	// Build 100 realistic Go files and store original content for reset.
	const fileCount = 100
	origContent := make([]string, fileCount)
	files := make([]*CompressibleFile, fileCount)
	for i := 0; i < fileCount; i++ {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("package pkg%d\n\n", i))
		sb.WriteString("import (\n\t\"context\"\n\t\"fmt\"\n)\n\n")
		sb.WriteString(fmt.Sprintf("// Config%d holds configuration.\n", i))
		sb.WriteString(fmt.Sprintf("type Config%d struct {\n", i))
		sb.WriteString("\tHost string\n\tPort int\n}\n\n")
		for j := 0; j < 5; j++ {
			sb.WriteString(fmt.Sprintf("// Func%d_%d does something.\n", i, j))
			sb.WriteString(fmt.Sprintf("func Func%d_%d(ctx context.Context, input string) (string, error) {\n", i, j))
			sb.WriteString(fmt.Sprintf("\tresult := fmt.Sprintf(\"%%s-%%d\", input, %d)\n", j))
			sb.WriteString("\tif ctx.Err() != nil {\n\t\treturn \"\", ctx.Err()\n\t}\n")
			sb.WriteString("\treturn result, nil\n}\n\n")
		}
		origContent[i] = sb.String()
		files[i] = &CompressibleFile{
			Path:    fmt.Sprintf("pkg%d/file.go", i),
			Content: origContent[i],
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// Reset files for each iteration.
		for i, f := range files {
			f.IsCompressed = false
			f.Language = ""
			f.Content = origContent[i]
		}
		_, err := orch.Compress(ctx, files)
		if err != nil {
			b.Fatal(err)
		}
	}
}
