package compression

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TOMLCompressor metadata tests
// ---------------------------------------------------------------------------

func TestTOMLCompressor_Language(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()
	assert.Equal(t, "toml", c.Language())
}

func TestTOMLCompressor_SupportedNodeTypes(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()
	types := c.SupportedNodeTypes()
	assert.Equal(t, []string{"table", "array_of_tables", "key_value", "comment"}, types)
}

// ---------------------------------------------------------------------------
// Empty / nil input
// ---------------------------------------------------------------------------

func TestTOMLCompressor_EmptyInput(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty byte slice", input: []byte{}},
		{name: "nil byte slice", input: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), tt.input)
			require.NoError(t, err)
			assert.Empty(t, output.Signatures)
			assert.Equal(t, 0, output.OriginalSize)
			assert.Equal(t, "toml", output.Language)
			assert.Equal(t, 0, output.NodeCount)
		})
	}
}

// ---------------------------------------------------------------------------
// Section headers
// ---------------------------------------------------------------------------

func TestTOMLCompressor_SectionHeaders(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name   string
		input  string
		expect string // substring that must appear in output
	}{
		{
			name:   "simple section",
			input:  "[database]\nhost = \"localhost\"",
			expect: "[database]",
		},
		{
			name:   "nested section",
			input:  "[tool.ruff.lint]\nselect = [\"E\", \"W\"]",
			expect: "[tool.ruff.lint]",
		},
		{
			name:   "section with whitespace before",
			input:  "  [indented]\nkey = \"val\"",
			expect: "[indented]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)
			assert.Contains(t, output.Signatures[0].Source, tt.expect)
		})
	}
}

// ---------------------------------------------------------------------------
// Array-of-tables headers
// ---------------------------------------------------------------------------

func TestTOMLCompressor_ArrayOfTables(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `[[bin]]
name = "server"
path = "src/main.rs"

[[example]]
name = "basic"
path = "examples/basic.rs"`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	assert.Contains(t, src, "[[bin]]")
	assert.Contains(t, src, "[[example]]")
}

// ---------------------------------------------------------------------------
// Comment preservation
// ---------------------------------------------------------------------------

func TestTOMLCompressor_CommentPreservation(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name    string
		input   string
		comment string // comment that must appear in output
	}{
		{
			name:    "standalone comment",
			input:   "# Build configuration\n[profile.release]",
			comment: "# Build configuration",
		},
		{
			name:    "indented comment",
			input:   "  # indented comment\nkey = 1",
			comment: "# indented comment",
		},
		{
			name:    "comment only file",
			input:   "# just a comment",
			comment: "# just a comment",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)
			assert.Contains(t, output.Signatures[0].Source, tt.comment)
		})
	}
}

// ---------------------------------------------------------------------------
// String truncation
// ---------------------------------------------------------------------------

func TestTOMLCompressor_StringTruncation(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name      string
		input     string
		truncated bool // whether we expect truncation ("..." in value)
	}{
		{
			name:      "short string preserved",
			input:     `title = "My Application"`,
			truncated: false,
		},
		{
			name:      "exactly 80 chars not truncated",
			input:     `desc = "` + strings.Repeat("a", 80) + `"`,
			truncated: false,
		},
		{
			name:      "81 chars truncated",
			input:     `desc = "` + strings.Repeat("b", 81) + `"`,
			truncated: true,
		},
		{
			name:      "long string truncated with ellipsis",
			input:     `description = "` + strings.Repeat("x", 200) + `"`,
			truncated: true,
		},
		{
			name:      "literal string truncated",
			input:     `path = '` + strings.Repeat("p", 100) + `'`,
			truncated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			if tt.truncated {
				assert.Contains(t, src, "...", "expected truncation marker")
			} else {
				// Non-truncated strings should not have "..." added.
				// (The original might have "..." in it, but these test inputs don't.)
				assert.NotContains(t, src, "...", "should not be truncated")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Inline array collapsing
// ---------------------------------------------------------------------------

func TestTOMLCompressor_InlineArrayCollapsing(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name      string
		input     string
		collapsed bool
		itemCount int // expected item count when collapsed
	}{
		{
			name:      "small array preserved",
			input:     `select = ["E", "W", "F"]`,
			collapsed: false,
		},
		{
			name:      "exactly 5 items preserved",
			input:     `items = ["a", "b", "c", "d", "e"]`,
			collapsed: false,
		},
		{
			name:      "6 items collapsed",
			input:     `items = ["a", "b", "c", "d", "e", "f"]`,
			collapsed: true,
			itemCount: 6,
		},
		{
			name:      "9 items collapsed",
			input:     `select = ["E", "W", "F", "I", "N", "UP", "S", "B", "A"]`,
			collapsed: true,
			itemCount: 9,
		},
		{
			name:      "empty array preserved",
			input:     `empty = []`,
			collapsed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			if tt.collapsed {
				assert.Contains(t, src, "/* ", "expected collapsed array marker")
				assert.Contains(t, src, " items */", "expected item count in collapsed array")
				// Verify exact item count.
				assert.Contains(t, src, fmtItemCount(tt.itemCount))
			} else {
				assert.NotContains(t, src, "/* ", "should not be collapsed")
			}
		})
	}
}

// fmtItemCount returns the expected collapsed array marker for the given count.
func fmtItemCount(n int) string {
	return fmt.Sprintf("/* %d items */", n)
}

// ---------------------------------------------------------------------------
// Multi-line array collapsing
// ---------------------------------------------------------------------------

func TestTOMLCompressor_MultiLineArray(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name      string
		input     string
		itemCount int
	}{
		{
			name: "multi-line array with 3 items",
			input: `deps = [
    "fastapi",
    "uvicorn",
    "pydantic",
]`,
			itemCount: 3,
		},
		{
			name: "multi-line array with 11 items",
			input: `dependencies = [
    "fastapi>=0.109.0",
    "uvicorn>=0.27.0",
    "pydantic>=2.5.0",
    "sqlalchemy>=2.0.0",
    "alembic>=1.13.0",
    "httpx>=0.26.0",
    "python-jose>=3.3.0",
    "passlib>=1.7.4",
    "python-multipart>=0.0.6",
    "redis>=5.0.0",
    "celery>=5.3.0",
]`,
			itemCount: 11,
		},
		{
			name: "multi-line array with comments inside",
			input: `items = [
    # group 1
    "alpha",
    "beta",
    # group 2
    "gamma",
]`,
			itemCount: 3,
		},
		{
			name: "multi-line array with items on first line",
			input: `items = ["first",
    "second",
    "third",
]`,
			itemCount: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			// Multi-line arrays should always be collapsed.
			assert.Contains(t, src, "items */]", "expected collapsed multi-line array")
			// Verify it's a single line now (key = [/* N items */]).
			nonEmpty := filterNonEmpty(strings.Split(src, "\n"))
			assert.Len(t, nonEmpty, 1, "collapsed multi-line array should be a single line")
		})
	}
}

// filterNonEmpty returns lines that are not empty after trimming.
func filterNonEmpty(lines []string) []string {
	var result []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			result = append(result, l)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Numbers, booleans, dates
// ---------------------------------------------------------------------------

func TestTOMLCompressor_NumbersAndBooleans(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name  string
		input string
	}{
		{name: "integer", input: "port = 5432"},
		{name: "float", input: "ratio = 3.14"},
		{name: "boolean true", input: "debug = true"},
		{name: "boolean false", input: "verbose = false"},
		{name: "date", input: `created = 2024-01-15`},
		{name: "datetime", input: `updated = 2024-01-15T10:30:00Z`},
		{name: "negative number", input: "offset = -42"},
		{name: "scientific notation", input: "epsilon = 1e-10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)
			// Numbers, booleans, dates should be preserved exactly.
			assert.Equal(t, tt.input, output.Signatures[0].Source)
		})
	}
}

// ---------------------------------------------------------------------------
// Inline tables
// ---------------------------------------------------------------------------

func TestTOMLCompressor_InlineTables(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple inline table",
			input: `serde = { version = "1.0", features = ["derive"] }`,
		},
		{
			name:  "complex inline table",
			input: `sqlx = { version = "0.7", features = ["runtime-tokio-rustls", "postgres", "migrate", "uuid", "chrono"] }`,
		},
		{
			name:  "inline table with single key",
			input: `author = { name = "Test" }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)
			// Inline tables should be preserved as-is.
			assert.Equal(t, tt.input, output.Signatures[0].Source)
		})
	}
}

// ---------------------------------------------------------------------------
// Multi-line strings (""" and ''')
// ---------------------------------------------------------------------------

func TestTOMLCompressor_MultiLineStrings(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	tests := []struct {
		name           string
		input          string
		expectContains string
		expectTrunc    bool
	}{
		{
			name: "triple double-quoted multi-line",
			input: `desc = """
This is a long
multi-line description
that spans several lines.
"""`,
			expectContains: `desc = "`,
			expectTrunc:    true,
		},
		{
			name: "triple single-quoted multi-line",
			input: `regex = '''
\d+\.\d+\.\d+
'''`,
			expectContains: `regex = '`,
			expectTrunc:    true,
		},
		{
			name:           "single-line triple double-quoted",
			input:          `desc = """short text"""`,
			expectContains: `desc = """short text"""`,
			expectTrunc:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			assert.Contains(t, src, tt.expectContains)
			if tt.expectTrunc {
				assert.Contains(t, src, "...")
				// Multi-line should be collapsed to a single line.
				nonEmpty := filterNonEmpty(strings.Split(src, "\n"))
				assert.Len(t, nonEmpty, 1, "multi-line string should collapse to one line")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Blank line preservation
// ---------------------------------------------------------------------------

func TestTOMLCompressor_BlankLinePreservation(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `[section1]
key1 = "val1"

[section2]
key2 = "val2"`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	// The blank line between sections should be preserved.
	assert.Contains(t, src, "\n\n")
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestTOMLCompressor_ContextCancellation(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	t.Run("already cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Compress(ctx, []byte("[section]\nkey = \"value\""))
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("large input with cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Generate large source to exercise the per-1000-line check.
		var b strings.Builder
		for i := 0; i < 5000; i++ {
			b.WriteString("key" + strings.Repeat("x", 10) + " = \"value\"\n")
		}

		_, err := c.Compress(ctx, []byte(b.String()))
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// ---------------------------------------------------------------------------
// Output structure
// ---------------------------------------------------------------------------

func TestTOMLCompressor_OutputStructure(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `# Config
[server]
host = "localhost"
port = 8080`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	// Single signature.
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]

	assert.Equal(t, KindDocComment, sig.Kind)
	assert.Equal(t, "toml-skeleton", sig.Name)
	assert.Equal(t, 1, sig.StartLine)
	assert.Greater(t, sig.EndLine, 0)

	// Metadata.
	assert.Equal(t, "toml", output.Language)
	assert.Equal(t, len(input), output.OriginalSize)
	assert.Greater(t, output.OutputSize, 0)
	assert.Equal(t, 1, output.NodeCount)
}

// ---------------------------------------------------------------------------
// Golden tests with fixtures
// ---------------------------------------------------------------------------

func testdataPath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	// Navigate from internal/compression/ to repo root, then into testdata.
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	elems := append([]string{repoRoot, "testdata"}, parts...)
	return filepath.Join(elems...)
}

func TestTOMLCompressor_GoldenCargo(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	fixturePath := testdataPath(t, "compression", "toml", "cargo.toml")
	source, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "failed to read fixture %s", fixturePath)

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	src := output.Signatures[0].Source

	// Structural assertions -- section headers preserved.
	assert.Contains(t, src, "[package]")
	assert.Contains(t, src, "[profile.release]")
	assert.Contains(t, src, "[dependencies]")
	assert.Contains(t, src, "[dev-dependencies]")
	assert.Contains(t, src, "[[bin]]")
	assert.Contains(t, src, "[[example]]")

	// Comment preserved.
	assert.Contains(t, src, "# Build configuration")

	// Key names preserved.
	assert.Contains(t, src, "name =")
	assert.Contains(t, src, "version =")
	assert.Contains(t, src, "opt-level =")

	// Inline tables preserved.
	assert.Contains(t, src, "serde = {")

	// Booleans preserved.
	assert.Contains(t, src, "lto = true")
	assert.Contains(t, src, "strip = true")

	// Numbers preserved.
	assert.Contains(t, src, "opt-level = 3")

	// Metadata.
	assert.Equal(t, len(source), output.OriginalSize)
	assert.Equal(t, "toml", output.Language)
	assert.Equal(t, 1, output.NodeCount)
}

func TestTOMLCompressor_GoldenPyproject(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	fixturePath := testdataPath(t, "compression", "toml", "pyproject.toml")
	source, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "failed to read fixture %s", fixturePath)

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	src := output.Signatures[0].Source

	// Section headers preserved.
	assert.Contains(t, src, "[build-system]")
	assert.Contains(t, src, "[project]")
	assert.Contains(t, src, "[project.optional-dependencies]")
	assert.Contains(t, src, "[tool.ruff]")
	assert.Contains(t, src, "[tool.ruff.lint]")
	assert.Contains(t, src, "[tool.mypy]")

	// Multi-line arrays (dependencies, authors, dev, select) should be collapsed.
	// dependencies has 11 items.
	assert.Contains(t, src, "dependencies = [/* 11 items */]")
	// dev has 6 items.
	assert.Contains(t, src, "dev = [/* 6 items */]")
	// authors has 1 item -- it's a multi-line array so it gets collapsed regardless.
	assert.Contains(t, src, "authors = [/*")

	// The select array is inline with 9 items -- should be collapsed.
	assert.Contains(t, src, "select = [/* 9 items */]")

	// Booleans preserved.
	assert.Contains(t, src, "strict = true")

	// Numbers preserved.
	assert.Contains(t, src, "line-length = 88")
}

func TestTOMLCompressor_GoldenSimple(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	fixturePath := testdataPath(t, "compression", "toml", "simple.toml")
	source, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "failed to read fixture %s", fixturePath)

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	src := output.Signatures[0].Source

	// Simple file should be mostly preserved as-is.
	assert.Contains(t, src, "# Simple TOML config")
	assert.Contains(t, src, `title = "My Application"`)
	assert.Contains(t, src, "version = 1")
	assert.Contains(t, src, "debug = true")
	assert.Contains(t, src, "[database]")
	assert.Contains(t, src, `host = "localhost"`)
	assert.Contains(t, src, "port = 5432")
	assert.Contains(t, src, `name = "mydb"`)
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestTOMLCompressor_CompressionRatio(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	// Use the pyproject fixture which has multi-line arrays that get collapsed.
	fixturePath := testdataPath(t, "compression", "toml", "pyproject.toml")
	source, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	assert.Less(t, output.OutputSize, output.OriginalSize,
		"compressed output should be smaller than original for pyproject.toml")
	assert.Greater(t, output.CompressionRatio(), 0.0,
		"compression ratio should be positive")
	assert.Less(t, output.CompressionRatio(), 1.0,
		"compression ratio should be less than 1.0 for a file with collapsible arrays")
}

// ---------------------------------------------------------------------------
// Dotted keys
// ---------------------------------------------------------------------------

func TestTOMLCompressor_DottedKeys(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `physical.color = "orange"
physical.shape = "round"`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	assert.Contains(t, src, `physical.color = "orange"`)
	assert.Contains(t, src, `physical.shape = "round"`)
}

// ---------------------------------------------------------------------------
// Quoted keys
// ---------------------------------------------------------------------------

func TestTOMLCompressor_QuotedKeys(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `"quoted.key" = "value"
'literal.key' = "value2"`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	assert.Contains(t, src, `"quoted.key" =`)
	assert.Contains(t, src, `'literal.key' =`)
}

// ---------------------------------------------------------------------------
// Unterminated multi-line constructs at EOF
// ---------------------------------------------------------------------------

func TestTOMLCompressor_UnterminatedMultiLineArray(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `items = [
    "one",
    "two",
    "three"`
	// No closing bracket.

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	// Should still produce a collapsed array.
	assert.Contains(t, src, "items = [/*")
	assert.Contains(t, src, "items */]")
}

func TestTOMLCompressor_UnterminatedMultiLineString(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `desc = """
This is an unterminated
multi-line string`
	// No closing """.

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	// Should emit a truncated version.
	assert.Contains(t, src, "desc = \"")
	assert.Contains(t, src, "...\"")
}

// ---------------------------------------------------------------------------
// Mixed content end-to-end
// ---------------------------------------------------------------------------

func TestTOMLCompressor_MixedContent(t *testing.T) {
	t.Parallel()
	c := NewTOMLCompressor()

	input := `# Application config
[app]
name = "my-service"
version = "1.2.3"
debug = false
port = 8080

[app.limits]
max_connections = 100
timeout = 30.5

# Long description that should be truncated
[metadata]
description = "` + strings.Repeat("a", 200) + `"
tags = ["web", "api", "rest", "microservice", "docker", "kubernetes"]
authors = [
    "Alice",
    "Bob",
    "Charlie",
]`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source

	// Comment preserved.
	assert.Contains(t, src, "# Application config")
	assert.Contains(t, src, "# Long description")

	// Section headers preserved.
	assert.Contains(t, src, "[app]")
	assert.Contains(t, src, "[app.limits]")
	assert.Contains(t, src, "[metadata]")

	// Short strings preserved.
	assert.Contains(t, src, `name = "my-service"`)

	// Long string truncated.
	assert.Contains(t, src, "description = \""+strings.Repeat("a", 80)+"...\"")

	// Inline array with 6 items collapsed.
	assert.Contains(t, src, "tags = [/* 6 items */]")

	// Multi-line array collapsed.
	assert.Contains(t, src, "authors = [/* 3 items */]")

	// Numbers and booleans preserved.
	assert.Contains(t, src, "debug = false")
	assert.Contains(t, src, "port = 8080")
	assert.Contains(t, src, "max_connections = 100")
	assert.Contains(t, src, "timeout = 30.5")
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkTOMLCompress(b *testing.B) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("failed to get caller info")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	fixturePath := filepath.Join(repoRoot, "testdata", "compression", "toml", "cargo.toml")

	source, err := os.ReadFile(fixturePath)
	if err != nil {
		b.Fatalf("failed to read fixture: %v", err)
	}

	c := NewTOMLCompressor()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTOMLCompress_Large(b *testing.B) {
	// Generate a large TOML file: 10,000 key-value pairs across 100 sections.
	var sb strings.Builder
	for s := 0; s < 100; s++ {
		sb.WriteString("[section" + strings.Repeat("x", 3) + "]\n")
		for k := 0; k < 100; k++ {
			sb.WriteString("key_" + strings.Repeat("y", 5) + " = \"" + strings.Repeat("v", 50) + "\"\n")
		}
		sb.WriteString("\n")
	}
	source := []byte(sb.String())

	c := NewTOMLCompressor()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}