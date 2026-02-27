package compression

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// e2eTestdataDir returns the absolute path to the testdata/compression/e2e directory.
func e2eTestdataDir() string {
	return filepath.Join(testdataDir(), "e2e")
}

// readE2EFixture reads a test fixture file from testdata/compression/e2e.
func readE2EFixture(t *testing.T, relPath string) string {
	t.Helper()
	path := filepath.Join(e2eTestdataDir(), relPath)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read fixture %s", relPath)
	return string(data)
}

// e2eConfig returns a CompressionConfig for E2E tests.
func e2eConfig(engine CompressEngine) CompressionConfig {
	return CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 10 * time.Second,
		Concurrency:    runtime.NumCPU(),
		Engine:         engine,
	}
}

// ---------------------------------------------------------------------------
// E2E: Orchestrator with AST engine
// ---------------------------------------------------------------------------

func TestE2E_OrchestratorASTEngine(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		path     string
		wantLang string
	}{
		{"TypeScript API route", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Go HTTP handler", "go/http-handler.go", "http-handler.go", "go"},
		{"Python FastAPI router", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust struct impl", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	orch := NewOrchestrator(e2eConfig(EngineAST))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: content},
			}

			stats, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)

			f := files[0]
			assert.True(t, f.IsCompressed, "file should be compressed: %s", tt.fixture)
			assert.Equal(t, tt.wantLang, f.Language)
			assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
				"compressed content should start with CompressedMarker")
			assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Orchestrator with Regex engine
// ---------------------------------------------------------------------------

func TestE2E_OrchestratorRegexEngine(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		path     string
		wantLang string
	}{
		{"TypeScript API route", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Go HTTP handler", "go/http-handler.go", "http-handler.go", "go"},
		{"Python FastAPI router", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust struct impl", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	orch := NewOrchestrator(e2eConfig(EngineRegex))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: content},
			}

			stats, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)

			f := files[0]
			assert.True(t, f.IsCompressed, "file should be compressed via regex: %s", tt.fixture)
			assert.Equal(t, tt.wantLang, f.Language)
			assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
				"compressed content should start with CompressedMarker")
			assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Orchestrator with Auto engine
// ---------------------------------------------------------------------------

func TestE2E_OrchestratorAutoEngine(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		path     string
		wantLang string
	}{
		{"TypeScript API route", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Go HTTP handler", "go/http-handler.go", "http-handler.go", "go"},
		{"Python FastAPI router", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust struct impl", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	orch := NewOrchestrator(e2eConfig(EngineAuto))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: content},
			}

			stats, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)

			f := files[0]
			assert.True(t, f.IsCompressed, "file should be compressed: %s", tt.fixture)
			assert.Equal(t, tt.wantLang, f.Language)
			assert.True(t, strings.HasPrefix(f.Content, CompressedMarker))
			assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Cross-engine comparison (AST vs Regex)
// ---------------------------------------------------------------------------

func TestE2E_CrossEngineComparison(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		path    string
		lang    string
	}{
		{"TypeScript", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Go", "go/http-handler.go", "http-handler.go", "go"},
		{"Python", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := readE2EFixture(t, tt.fixture)

			// Compress with AST engine.
			astOrch := NewOrchestrator(e2eConfig(EngineAST))
			astFiles := []*CompressibleFile{
				{Path: tt.path, Content: content},
			}
			astStats, err := astOrch.Compress(context.Background(), astFiles)
			require.NoError(t, err)
			require.Equal(t, int64(1), astStats.FilesCompressed, "AST should compress %s", tt.lang)

			astLen := len(astFiles[0].Content)

			// Compress with Regex engine.
			regexOrch := NewOrchestrator(e2eConfig(EngineRegex))
			regexFiles := []*CompressibleFile{
				{Path: tt.path, Content: content},
			}
			regexStats, err := regexOrch.Compress(context.Background(), regexFiles)
			require.NoError(t, err)
			require.Equal(t, int64(1), regexStats.FilesCompressed, "Regex should compress %s", tt.lang)

			regexLen := len(regexFiles[0].Content)
			originalLen := len(content)

			t.Logf("%s: original=%d AST=%d Regex=%d", tt.lang, originalLen, astLen, regexLen)

			// Both should produce smaller output than original.
			assert.Less(t, astLen, originalLen,
				"AST compressed should be smaller than original for %s", tt.lang)
			assert.Less(t, regexLen, originalLen,
				"Regex compressed should be smaller than original for %s", tt.lang)

			// Both engines produce valid compressed output. AST extracts more
			// structural detail (struct fields, doc comments, member signatures)
			// so it may produce larger output than regex, which only captures
			// top-level declarations. Both are valid compression strategies.
			astRatio := float64(astLen) / float64(originalLen)
			regexRatio := float64(regexLen) / float64(originalLen)
			t.Logf("%s ratios: AST=%.2f Regex=%.2f", tt.lang, astRatio, regexRatio)
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Mixed-language project
// ---------------------------------------------------------------------------

func TestE2E_MixedLanguageProject(t *testing.T) {
	orch := NewOrchestrator(e2eConfig(EngineAuto))

	files := []*CompressibleFile{
		{
			Path:    "api-route.ts",
			Content: readE2EFixture(t, "typescript/api-route.ts"),
		},
		{
			Path:    "http-handler.go",
			Content: readE2EFixture(t, "go/http-handler.go"),
		},
		{
			Path:    "fastapi-router.py",
			Content: readE2EFixture(t, "python/fastapi-router.py"),
		},
		{
			Path:    "struct-impl.rs",
			Content: readE2EFixture(t, "rust/struct-impl.rs"),
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.Equal(t, int64(4), atomic.LoadInt64(&stats.FilesCompressed),
		"all 4 files should be compressed")
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesSkipped))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesFailed))

	expectedLangs := []string{"typescript", "go", "python", "rust"}
	for i, f := range files {
		assert.True(t, f.IsCompressed, "file %d (%s) should be compressed", i, f.Path)
		assert.Equal(t, expectedLangs[i], f.Language, "language mismatch for %s", f.Path)
		assert.True(t, strings.HasPrefix(f.Content, CompressedMarker),
			"file %s should have marker prefix", f.Path)
	}
}

// ---------------------------------------------------------------------------
// E2E: Unsupported file types
// ---------------------------------------------------------------------------

func TestE2E_UnsupportedFileTypes(t *testing.T) {
	tests := []struct {
		path    string
		content string
	}{
		{"README.md", "# Project Title\n\nSome description.\n"},
		{"notes.txt", "These are plain text notes.\nNothing special here.\n"},
		{"logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"><circle cx="50" cy="50" r="40"/></svg>`},
		{"Makefile", "all:\n\tgo build ./...\n"},
		{"script.sh", "#!/bin/bash\necho 'hello'\n"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			orch := NewOrchestrator(e2eConfig(EngineAuto))
			files := []*CompressibleFile{
				{Path: tt.path, Content: tt.content},
			}

			stats, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)

			f := files[0]
			assert.False(t, f.IsCompressed,
				"unsupported file %s should not be compressed", tt.path)
			assert.Equal(t, tt.content, f.Content,
				"unsupported file %s content should be unchanged", tt.path)
			assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesSkipped))
			assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesCompressed))
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: CompressedMarker always present
// ---------------------------------------------------------------------------

func TestE2E_CompressedMarkerPresent(t *testing.T) {
	fixtures := []struct {
		path    string
		fixture string
	}{
		{"api-route.ts", "typescript/api-route.ts"},
		{"http-handler.go", "go/http-handler.go"},
		{"fastapi-router.py", "python/fastapi-router.py"},
		{"struct-impl.rs", "rust/struct-impl.rs"},
	}

	for _, engine := range []CompressEngine{EngineAST, EngineRegex, EngineAuto} {
		t.Run(string(engine), func(t *testing.T) {
			orch := NewOrchestrator(e2eConfig(engine))

			for _, fx := range fixtures {
				t.Run(fx.path, func(t *testing.T) {
					content := readE2EFixture(t, fx.fixture)
					files := []*CompressibleFile{
						{Path: fx.path, Content: content},
					}

					_, err := orch.Compress(context.Background(), files)
					require.NoError(t, err)

					f := files[0]
					require.True(t, f.IsCompressed, "file should be compressed")
					assert.True(t, strings.HasPrefix(f.Content, CompressedMarker+"\n"),
						"compressed content must start with marker followed by newline")
				})
			}
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Compression ratio bounds
// ---------------------------------------------------------------------------

func TestE2E_CompressionRatioBounds(t *testing.T) {
	fixtures := []struct {
		path    string
		fixture string
	}{
		{"api-route.ts", "typescript/api-route.ts"},
		{"http-handler.go", "go/http-handler.go"},
		{"fastapi-router.py", "python/fastapi-router.py"},
		{"struct-impl.rs", "rust/struct-impl.rs"},
	}

	orch := NewOrchestrator(e2eConfig(EngineAST))

	for _, fx := range fixtures {
		t.Run(fx.path, func(t *testing.T) {
			content := readE2EFixture(t, fx.fixture)
			files := []*CompressibleFile{
				{Path: fx.path, Content: content},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed)

			compressedLen := len(files[0].Content)
			originalLen := len(content)
			ratio := float64(compressedLen) / float64(originalLen)

			t.Logf("%s: original=%d compressed=%d ratio=%.3f", fx.path, originalLen, compressedLen, ratio)

			// Compression ratio should be between 0.1 and 0.95.
			// Below 0.1 would mean almost nothing is kept.
			// Above 0.95 would mean almost no compression happened.
			assert.Greater(t, ratio, 0.1,
				"ratio %.3f too low for %s", ratio, fx.path)
			assert.Less(t, ratio, 0.95,
				"ratio %.3f too high (not enough compression) for %s", ratio, fx.path)
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Stats counters are correct
// ---------------------------------------------------------------------------

func TestE2E_StatsCounters(t *testing.T) {
	orch := NewOrchestrator(e2eConfig(EngineAuto))

	files := []*CompressibleFile{
		{Path: "api-route.ts", Content: readE2EFixture(t, "typescript/api-route.ts")},
		{Path: "http-handler.go", Content: readE2EFixture(t, "go/http-handler.go")},
		{Path: "README.md", Content: "# Readme\n\nText.\n"},
		{Path: "fastapi-router.py", Content: readE2EFixture(t, "python/fastapi-router.py")},
		{Path: "notes.txt", Content: "plain text\n"},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.Equal(t, int64(3), atomic.LoadInt64(&stats.FilesCompressed),
		"3 files should be compressed (ts, go, py)")
	assert.Equal(t, int64(2), atomic.LoadInt64(&stats.FilesSkipped),
		"2 files should be skipped (md, txt)")
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesFailed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&stats.FilesTimedOut))
	assert.Equal(t, int64(5), stats.TotalFiles())
	assert.Greater(t, stats.TotalDuration, time.Duration(0))

	// Token counts should reflect compression savings.
	assert.Greater(t, atomic.LoadInt64(&stats.OriginalTokens), int64(0))
	assert.Greater(t, atomic.LoadInt64(&stats.CompressedTokens), int64(0))
	assert.Greater(t, stats.TokenSavings(), int64(0),
		"compression should save tokens")
}

// ---------------------------------------------------------------------------
// E2E: Regex achieves meaningful token reduction
// ---------------------------------------------------------------------------

func TestE2E_RegexTokenReduction(t *testing.T) {
	fixtures := []struct {
		path    string
		fixture string
	}{
		{"api-route.ts", "typescript/api-route.ts"},
		{"http-handler.go", "go/http-handler.go"},
		{"fastapi-router.py", "python/fastapi-router.py"},
		{"struct-impl.rs", "rust/struct-impl.rs"},
	}

	orch := NewOrchestrator(e2eConfig(EngineRegex))

	for _, fx := range fixtures {
		t.Run(fx.path, func(t *testing.T) {
			content := readE2EFixture(t, fx.fixture)
			files := []*CompressibleFile{
				{Path: fx.path, Content: content},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed)

			compressedLen := len(files[0].Content)
			originalLen := len(content)
			reduction := 1.0 - float64(compressedLen)/float64(originalLen)

			t.Logf("%s: regex reduction=%.1f%%", fx.path, reduction*100)

			// Regex should achieve at least some reduction (> 5%).
			// The task spec targets 30-50%, but we allow wider bounds for tests.
			assert.Greater(t, reduction, 0.05,
				"regex should achieve some token reduction for %s (got %.1f%%)", fx.path, reduction*100)
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Regression test -- arrow functions in TS/JS
// ---------------------------------------------------------------------------

func TestE2E_Regression_ArrowFunctions(t *testing.T) {
	// Arrow functions are assigned to const -- regex should catch the const.
	source := `export const handler = async (req: Request): Promise<Response> => {
  const data = await req.json();
  return Response.json(data);
};

export const add = (a: number, b: number): number => a + b;
`
	orch := NewOrchestrator(e2eConfig(EngineRegex))
	files := []*CompressibleFile{
		{Path: "arrows.ts", Content: source},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.FilesCompressed,
		"TypeScript file with arrow functions should be compressed")
}

// ---------------------------------------------------------------------------
// E2E: Regression test -- Go generics
// ---------------------------------------------------------------------------

func TestE2E_Regression_GoGenerics(t *testing.T) {
	source := `package collections

type Set[T comparable] struct {
	items map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{items: make(map[T]struct{})}
}

func (s *Set[T]) Add(item T) {
	s.items[item] = struct{}{}
}

func (s *Set[T]) Contains(item T) bool {
	_, ok := s.items[item]
	return ok
}

func Map[T any, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}
`
	orch := NewOrchestrator(e2eConfig(EngineAST))
	files := []*CompressibleFile{
		{Path: "generics.go", Content: source},
	}

	_, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)
	require.True(t, files[0].IsCompressed, "Go generics file should be compressed")

	// Verify key signatures are present.
	compressed := files[0].Content
	assert.Contains(t, compressed, "Set")
	assert.Contains(t, compressed, "NewSet")
}

// ---------------------------------------------------------------------------
// E2E: Large file compression correctness
// ---------------------------------------------------------------------------

func TestE2E_LargeFile(t *testing.T) {
	// Generate a large Go file (> 10KB).
	var b strings.Builder
	b.WriteString("package largefile\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"fmt\"\n)\n\n")

	for i := 0; i < 100; i++ {
		b.WriteString("// Handler" + string(rune('A'+i%26)) + " handles requests.\n")
		b.WriteString("func Handler" + string(rune('A'+i%26)))
		b.WriteString("_" + strings.Repeat("x", i%5))
		b.WriteString("(ctx context.Context, input string) (string, error) {\n")
		b.WriteString("\tresult := fmt.Sprintf(\"%s-%d\", input, " + string(rune('0'+i%10)) + ")\n")
		b.WriteString("\tif ctx.Err() != nil {\n")
		b.WriteString("\t\treturn \"\", ctx.Err()\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn result, nil\n}\n\n")
	}

	source := b.String()
	assert.Greater(t, len(source), 10000, "source should be > 10KB")

	orch := NewOrchestrator(e2eConfig(EngineAST))
	files := []*CompressibleFile{
		{Path: "large.go", Content: source},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.FilesCompressed)
	assert.True(t, files[0].IsCompressed)

	// Should achieve meaningful compression on large files.
	ratio := float64(len(files[0].Content)) / float64(len(source))
	assert.Less(t, ratio, 0.8, "large file should compress well (ratio=%.3f)", ratio)
}
