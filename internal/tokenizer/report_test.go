package tokenizer

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// updateGolden controls whether golden files are regenerated rather than
// compared. Pass -update on the command line to regenerate:
//
//	go test ./internal/tokenizer/... -update
var updateGolden = flag.Bool("update", false, "regenerate golden files")

// makeFile is a test helper that creates a FileDescriptor with the given fields.
func makeFile(t *testing.T, path string, tokenCount, tier int) *pipeline.FileDescriptor {
	t.Helper()
	return &pipeline.FileDescriptor{
		Path:       path,
		TokenCount: tokenCount,
		Tier:       tier,
	}
}

// --- FormatInt ---

func TestFormatInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "0"},
		{name: "single digit", n: 7, want: "7"},
		{name: "three digits", n: 999, want: "999"},
		{name: "four digits", n: 1000, want: "1,000"},
		{name: "five digits", n: 12345, want: "12,345"},
		{name: "six digits", n: 100000, want: "100,000"},
		{name: "seven digits", n: 1234567, want: "1,234,567"},
		{name: "negative", n: -1234, want: "-1,234"},
		{name: "large number", n: 89420, want: "89,420"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatInt(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- TierLabel ---

func TestTierLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Config", TierLabel[0])
	assert.Equal(t, "Source", TierLabel[1])
	assert.Equal(t, "Secondary", TierLabel[2])
	assert.Equal(t, "Tests", TierLabel[3])
	assert.Equal(t, "Docs", TierLabel[4])
	assert.Equal(t, "CI/Lock", TierLabel[5])
}

func TestTierLabelFor_UnknownTier(t *testing.T) {
	t.Parallel()
	// Unknown tier falls back to a generated label.
	assert.Equal(t, "Tier99", tierLabelFor(99))
}

// --- NewTokenReport ---

func TestNewTokenReport_Empty(t *testing.T) {
	t.Parallel()

	r := NewTokenReport(nil, "cl100k_base", 0)
	require.NotNil(t, r)
	assert.Equal(t, "cl100k_base", r.TokenizerName)
	assert.Equal(t, 0, r.TotalFiles)
	assert.Equal(t, 0, r.TotalTokens)
	assert.Empty(t, r.TierStats)
}

func TestNewTokenReport_NilFilesEntry(t *testing.T) {
	t.Parallel()

	// A nil pointer in the slice should be skipped gracefully.
	files := []*pipeline.FileDescriptor{nil, makeFile(t, "a.go", 100, 1)}
	r := NewTokenReport(files, "none", 0)
	assert.Equal(t, 1, r.TotalFiles)
	assert.Equal(t, 100, r.TotalTokens)
}

func TestNewTokenReport_AggregateStats(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "config.toml", 200, 0),
		makeFile(t, "main.go", 500, 1),
		makeFile(t, "util.go", 300, 1),
		makeFile(t, "README.md", 150, 4),
	}

	r := NewTokenReport(files, "cl100k_base", 2000)
	require.NotNil(t, r)
	assert.Equal(t, 4, r.TotalFiles)
	assert.Equal(t, 1150, r.TotalTokens)
	assert.Equal(t, 2000, r.Budget)

	require.Contains(t, r.TierStats, 0)
	assert.Equal(t, 1, r.TierStats[0].FileCount)
	assert.Equal(t, 200, r.TierStats[0].TokenCount)

	require.Contains(t, r.TierStats, 1)
	assert.Equal(t, 2, r.TierStats[1].FileCount)
	assert.Equal(t, 800, r.TierStats[1].TokenCount)

	require.Contains(t, r.TierStats, 4)
	assert.Equal(t, 1, r.TierStats[4].FileCount)
	assert.Equal(t, 150, r.TierStats[4].TokenCount)
}

// --- TokenReport.Format ---

func TestTokenReport_Format_UnlimitedBudget(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "main.go", 1000, 1),
	}
	r := NewTokenReport(files, "cl100k_base", 0)
	out := r.Format()

	assert.Contains(t, out, "Token Report (cl100k_base)")
	assert.Contains(t, out, "─")
	assert.Contains(t, out, "Total files:  1")
	assert.Contains(t, out, "Total tokens: 1,000")
	assert.Contains(t, out, "Budget:       unlimited")
	assert.Contains(t, out, "Tier 1 (Source):")
}

func TestTokenReport_Format_WithBudget(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "main.go", 500, 1),
	}
	r := NewTokenReport(files, "o200k_base", 1000)
	out := r.Format()

	assert.Contains(t, out, "Token Report (o200k_base)")
	assert.Contains(t, out, "Budget:       1,000 (50% used)")
}

func TestTokenReport_Format_NoFiles(t *testing.T) {
	t.Parallel()

	r := NewTokenReport(nil, "none", 0)
	out := r.Format()

	assert.Contains(t, out, "Token Report (none)")
	assert.Contains(t, out, "Total files:  0")
	assert.Contains(t, out, "Total tokens: 0")
	// No "By Tier" section for empty report.
	assert.NotContains(t, out, "By Tier:")
}

// --- NewTopFilesReport ---

func TestNewTopFilesReport_SortedDescending(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "small.go", 100, 1),
		makeFile(t, "large.go", 5000, 1),
		makeFile(t, "medium.go", 1000, 2),
	}

	r := NewTopFilesReport(files, 10)
	require.Len(t, r.Files, 3)
	assert.Equal(t, "large.go", r.Files[0].Path)
	assert.Equal(t, "medium.go", r.Files[1].Path)
	assert.Equal(t, "small.go", r.Files[2].Path)
}

func TestNewTopFilesReport_LimitN(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "a.go", 300, 1),
		makeFile(t, "b.go", 200, 1),
		makeFile(t, "c.go", 100, 1),
	}

	r := NewTopFilesReport(files, 2)
	assert.Equal(t, 2, r.N)
	require.Len(t, r.Files, 2)
	assert.Equal(t, "a.go", r.Files[0].Path)
	assert.Equal(t, "b.go", r.Files[1].Path)
}

func TestNewTopFilesReport_NZeroIncludesAll(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "a.go", 300, 1),
		makeFile(t, "b.go", 200, 1),
	}

	r := NewTopFilesReport(files, 0)
	assert.Equal(t, 0, r.N)
	assert.Len(t, r.Files, 2)
}

func TestNewTopFilesReport_NilEntry(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{nil, makeFile(t, "a.go", 100, 1)}
	r := NewTopFilesReport(files, 10)
	assert.Len(t, r.Files, 1)
}

// --- TopFilesReport.Format ---

func TestTopFilesReport_Format_WithFiles(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "prisma/schema.prisma", 4210, 0),
		makeFile(t, "src/main.go", 800, 1),
	}
	r := NewTopFilesReport(files, 10)
	out := r.Format()

	assert.Contains(t, out, "Top 10 Files by Token Count:")
	assert.Contains(t, out, "─")
	assert.Contains(t, out, "4,210")
	assert.Contains(t, out, "Tier 0: Config")
	assert.Contains(t, out, " 1.")
	assert.Contains(t, out, " 2.")
}

func TestTopFilesReport_Format_Empty(t *testing.T) {
	t.Parallel()

	r := NewTopFilesReport(nil, 10)
	out := r.Format()

	assert.Contains(t, out, "Top 10 Files by Token Count:")
	assert.Contains(t, out, "(no files)")
}

func TestTopFilesReport_Format_AllFiles(t *testing.T) {
	t.Parallel()

	r := NewTopFilesReport(nil, 0)
	out := r.Format()

	assert.Contains(t, out, "All Files by Token Count:")
}

// --- NewHeatmapReport ---

func TestNewHeatmapReport_DensityCalculation(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "dense.json", 1000, 0),
		makeFile(t, "sparse.go", 100, 1),
	}
	lineCounts := map[string]int{
		"dense.json": 10,  // density = 100.0
		"sparse.go":  100, // density = 1.0
	}

	r := NewHeatmapReport(files, lineCounts)
	require.Len(t, r.Files, 2)
	// Sorted descending by density.
	assert.Equal(t, "dense.json", r.Files[0].Path)
	assert.InDelta(t, 100.0, r.Files[0].Density, 0.001)
	assert.Equal(t, "sparse.go", r.Files[1].Path)
	assert.InDelta(t, 1.0, r.Files[1].Density, 0.001)
}

func TestNewHeatmapReport_ZeroLines_NoDivisionByZero(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "empty.go", 0, 1),
	}
	lineCounts := map[string]int{"empty.go": 0}

	r := NewHeatmapReport(files, lineCounts)
	require.Len(t, r.Files, 1)
	assert.Equal(t, 0.0, r.Files[0].Density)
}

func TestNewHeatmapReport_NilLineCounts(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "main.go", 500, 1),
	}

	r := NewHeatmapReport(files, nil)
	require.Len(t, r.Files, 1)
	assert.Equal(t, 0.0, r.Files[0].Density) // no line count available
}

func TestNewHeatmapReport_NilFiles(t *testing.T) {
	t.Parallel()

	r := NewHeatmapReport(nil, nil)
	require.NotNil(t, r)
	assert.Empty(t, r.Files)
}

func TestNewHeatmapReport_NilFileEntry(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{nil, makeFile(t, "a.go", 100, 1)}
	r := NewHeatmapReport(files, map[string]int{"a.go": 50})
	assert.Len(t, r.Files, 1)
}

// --- HeatmapReport.Format ---

func TestHeatmapReport_Format_WithFiles(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "data/fixtures.json", 111000, 0),
	}
	lineCounts := map[string]int{"data/fixtures.json": 780}

	r := NewHeatmapReport(files, lineCounts)
	out := r.Format()

	assert.Contains(t, out, "Token Heatmap (tokens per line):")
	assert.Contains(t, out, "─")
	assert.Contains(t, out, "tok/line")
	assert.Contains(t, out, "780")
	assert.Contains(t, out, "111,000")
	assert.True(t, strings.Contains(out, " 1."))
}

func TestHeatmapReport_Format_Empty(t *testing.T) {
	t.Parallel()

	r := NewHeatmapReport(nil, nil)
	out := r.Format()

	assert.Contains(t, out, "Token Heatmap (tokens per line):")
	assert.Contains(t, out, "(no files)")
}

// --- HeatmapReport: density-sort correctness ---

// TestNewHeatmapReport_SortedByDensityDescending verifies that files are
// ordered by density (tokens/line) descending -- the most token-dense file
// must appear first.
func TestNewHeatmapReport_SortedByDensityDescending(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "sparse.go", 10, 1),     // 10 tokens / 100 lines = 0.10 tok/line
		makeFile(t, "dense.json", 500, 0),   // 500 tokens / 5 lines  = 100.0 tok/line
		makeFile(t, "medium.ts", 200, 2),    // 200 tokens / 20 lines = 10.0 tok/line
	}
	lineCounts := map[string]int{
		"sparse.go":  100,
		"dense.json": 5,
		"medium.ts":  20,
	}

	r := NewHeatmapReport(files, lineCounts)
	require.Len(t, r.Files, 3)

	assert.Equal(t, "dense.json", r.Files[0].Path, "highest density must be first")
	assert.InDelta(t, 100.0, r.Files[0].Density, 0.001)

	assert.Equal(t, "medium.ts", r.Files[1].Path)
	assert.InDelta(t, 10.0, r.Files[1].Density, 0.001)

	assert.Equal(t, "sparse.go", r.Files[2].Path, "lowest density must be last")
	assert.InDelta(t, 0.1, r.Files[2].Density, 0.001)
}

// TestNewHeatmapReport_ZeroLines_GuardDivision verifies that files with exactly
// 0 lines receive density 0 (not a division-by-zero panic or +Inf).
func TestNewHeatmapReport_ZeroLines_GuardDivision(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "empty.bin", 999, 0), // many tokens but 0 lines
		makeFile(t, "normal.go", 100, 1), // 100 tokens / 10 lines = 10.0
	}
	lineCounts := map[string]int{
		"empty.bin": 0,
		"normal.go": 10,
	}

	r := NewHeatmapReport(files, lineCounts)
	require.Len(t, r.Files, 2)

	// normal.go has density 10.0; empty.bin has density 0.0 --
	// normal.go should sort first despite empty.bin having more tokens.
	assert.Equal(t, "normal.go", r.Files[0].Path, "non-zero density should rank above zero-density")
	assert.InDelta(t, 10.0, r.Files[0].Density, 0.001)

	assert.Equal(t, "empty.bin", r.Files[1].Path)
	assert.Equal(t, 0.0, r.Files[1].Density)
}

// --- TopFilesReport: exact N boundary ---

// TestNewTopFilesReport_ExactlyFive verifies that when --top-files 5 is used
// with more than 5 files, exactly 5 entries are returned sorted descending.
func TestNewTopFilesReport_ExactlyFive(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "a.go", 100, 1),
		makeFile(t, "b.go", 600, 1),
		makeFile(t, "c.go", 300, 2),
		makeFile(t, "d.go", 900, 1),
		makeFile(t, "e.go", 50, 3),
		makeFile(t, "f.go", 750, 1),
		makeFile(t, "g.go", 400, 2),
	}

	r := NewTopFilesReport(files, 5)
	assert.Equal(t, 5, r.N)
	require.Len(t, r.Files, 5, "--top-files 5 must return exactly 5 files")

	// Verify descending order of the returned slice.
	for i := 1; i < len(r.Files); i++ {
		assert.GreaterOrEqual(t, r.Files[i-1].TokenCount, r.Files[i].TokenCount,
			"files must be sorted descending by token count")
	}

	// d.go (900) must be first; e.go (50) and a.go (100) must be excluded.
	assert.Equal(t, "d.go", r.Files[0].Path)
	paths := make([]string, len(r.Files))
	for i, f := range r.Files {
		paths[i] = f.Path
	}
	assert.NotContains(t, paths, "e.go", "e.go (50 tokens) must be excluded from top-5")
	assert.NotContains(t, paths, "a.go", "a.go (100 tokens) must be excluded from top-5")
}

// TestNewTopFilesReport_FewerThanN verifies that when the pool has fewer files
// than N, all files are returned without error.
func TestNewTopFilesReport_FewerThanN(t *testing.T) {
	t.Parallel()

	files := []*pipeline.FileDescriptor{
		makeFile(t, "x.go", 200, 1),
		makeFile(t, "y.go", 100, 1),
	}

	r := NewTopFilesReport(files, 5)
	assert.Equal(t, 5, r.N)
	// Only 2 files exist -- we get 2, not a panic.
	require.Len(t, r.Files, 2, "fewer than N files returns all available files")
	assert.Equal(t, "x.go", r.Files[0].Path)
	assert.Equal(t, "y.go", r.Files[1].Path)
}

// --- Golden test ---

// goldenPath returns the path to a golden file in the package testdata/golden
// directory (relative to the test binary working directory).
func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".golden")
}

// checkOrUpdateGolden compares actual against the named golden file. If the
// -update flag is set OR the golden file does not yet exist, the file is
// written (and the test passes). Otherwise the file is read and compared
// byte-for-byte; any mismatch fails the test.
func checkOrUpdateGolden(t *testing.T, name string, actual []byte) {
	t.Helper()

	golden := goldenPath(name)

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(t, os.WriteFile(golden, actual, 0o644))
		return
	}

	expected, err := os.ReadFile(golden)
	if os.IsNotExist(err) {
		// Auto-create on first run so the test suite is self-bootstrapping.
		require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0o755))
		require.NoError(t, os.WriteFile(golden, actual, 0o644))
		return
	}
	require.NoError(t, err, "golden file read error: %s", golden)
	assert.Equal(t, string(expected), string(actual),
		"golden mismatch for %s; run with -update to regenerate", name)
}

// TestTokenReport_Golden verifies that a fixed set of files produces an exact,
// stable token report. Run with -update to regenerate the golden file.
func TestTokenReport_Golden(t *testing.T) {
	// Not parallelized: golden file writes from multiple goroutines could race
	// on the first run (auto-create path). Simpler to keep sequential.
	files := []*pipeline.FileDescriptor{
		makeFile(t, "prisma/schema.prisma", 4210, 0),
		makeFile(t, "lib/services/transaction.ts", 3890, 1),
		makeFile(t, "app/api/transactions/route.ts", 2340, 1),
		makeFile(t, "README.md", 450, 4),
		makeFile(t, "jest.config.ts", 120, 5),
	}

	r := NewTokenReport(files, "cl100k_base", 200000)
	checkOrUpdateGolden(t, "token_report", []byte(r.Format()))
}

// TestTopFilesReport_Golden verifies that a fixed set of files produces an
// exact, stable top-files report. Run with -update to regenerate.
func TestTopFilesReport_Golden(t *testing.T) {
	files := []*pipeline.FileDescriptor{
		makeFile(t, "prisma/schema.prisma", 4210, 0),
		makeFile(t, "lib/services/transaction.ts", 3890, 1),
		makeFile(t, "app/api/transactions/route.ts", 2340, 1),
		makeFile(t, "README.md", 450, 4),
		makeFile(t, "jest.config.ts", 120, 5),
	}

	r := NewTopFilesReport(files, 3)
	checkOrUpdateGolden(t, "top_files_report", []byte(r.Format()))
}

// TestHeatmapReport_Golden verifies that a fixed set of files with known line
// counts produces an exact, stable heatmap report. Run with -update to regenerate.
func TestHeatmapReport_Golden(t *testing.T) {
	files := []*pipeline.FileDescriptor{
		makeFile(t, "data/fixtures.json", 111000, 0),
		makeFile(t, "prisma/schema.prisma", 4210, 0),
		makeFile(t, "package-lock.json", 101000, 5),
	}
	lineCounts := map[string]int{
		"data/fixtures.json":   780,
		"prisma/schema.prisma": 348,
		"package-lock.json":    12000,
	}

	r := NewHeatmapReport(files, lineCounts)
	checkOrUpdateGolden(t, "heatmap_report", []byte(r.Format()))
}
