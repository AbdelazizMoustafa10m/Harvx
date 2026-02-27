package compression

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Metadata tests
// ---------------------------------------------------------------------------

func TestJSONCompressor_Language(t *testing.T) {
	c := NewJSONCompressor()
	assert.Equal(t, "json", c.Language())
}

func TestJSONCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewJSONCompressor()
	types := c.SupportedNodeTypes()
	assert.ElementsMatch(t, []string{"object", "array", "key_value"}, types)
}

// ---------------------------------------------------------------------------
// Empty and invalid input
// ---------------------------------------------------------------------------

func TestJSONCompressor_EmptyInput(t *testing.T) {
	c := NewJSONCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)

	assert.Equal(t, "json", output.Language)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OutputSize)
	assert.Equal(t, 0, output.NodeCount)
}

func TestJSONCompressor_InvalidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "truncated object",
			input: `{"name": "test"`,
		},
		{
			name:  "trailing comma",
			input: `{"a": 1, "b": 2,}`,
		},
		{
			name:  "bare words",
			input: `not json at all`,
		},
		{
			name:  "single bracket",
			input: `{`,
		},
		{
			name:  "yaml-like",
			input: "name: test\nversion: 1",
		},
	}

	c := NewJSONCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err, "invalid JSON should not return an error")

			// Should fall back to full content unchanged.
			require.Len(t, output.Signatures, 1)
			sig := output.Signatures[0]
			assert.Equal(t, KindDocComment, sig.Kind)
			assert.Equal(t, "json-raw", sig.Name)
			assert.Equal(t, tt.input, sig.Source, "fallback should preserve input verbatim")
			assert.Equal(t, "json", output.Language)
			assert.Equal(t, len(tt.input), output.OriginalSize)
			assert.Equal(t, len(tt.input), output.OutputSize)
		})
	}
}

// ---------------------------------------------------------------------------
// Simple object (small enough to be preserved intact)
// ---------------------------------------------------------------------------

func TestJSONCompressor_SimpleObject(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"name": "test", "version": 1, "enabled": true, "items": [1, 2, 3]}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindDocComment, sig.Kind)
	assert.Equal(t, "json-skeleton", sig.Name)

	// The output should be valid JSON.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(sig.Source), &parsed)
	require.NoError(t, err, "compressed output should be valid JSON")

	// All keys should be preserved at depth 0.
	assert.Contains(t, sig.Source, `"name"`)
	assert.Contains(t, sig.Source, `"version"`)
	assert.Contains(t, sig.Source, `"enabled"`)
	assert.Contains(t, sig.Source, `"items"`)

	// Small array [1,2,3] should be preserved (3 < 5).
	assert.Contains(t, sig.Source, "1")
	assert.Contains(t, sig.Source, "2")
	assert.Contains(t, sig.Source, "3")
}

// ---------------------------------------------------------------------------
// String truncation
// ---------------------------------------------------------------------------

func TestJSONCompressor_StringTruncation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrunc   bool
		wantContain string
	}{
		{
			name:        "short string preserved",
			input:       `{"key": "short"}`,
			wantTrunc:   false,
			wantContain: `"short"`,
		},
		{
			name:        "exactly 50 chars preserved",
			input:       `{"key": "` + strings.Repeat("a", 50) + `"}`,
			wantTrunc:   false,
			wantContain: strings.Repeat("a", 50),
		},
		{
			name:        "51 chars truncated",
			input:       `{"key": "` + strings.Repeat("b", 51) + `"}`,
			wantTrunc:   true,
			wantContain: strings.Repeat("b", 50) + "...",
		},
		{
			name:        "long string truncated with ellipsis",
			input:       `{"description": "` + strings.Repeat("x", 100) + `"}`,
			wantTrunc:   true,
			wantContain: strings.Repeat("x", 50) + "...",
		},
	}

	c := NewJSONCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			assert.Contains(t, src, tt.wantContain)

			if tt.wantTrunc {
				assert.Contains(t, src, "...")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Array collapsing
// ---------------------------------------------------------------------------

func TestJSONCompressor_ArrayCollapsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantExpand   bool   // true if array should be expanded, false if collapsed
		wantCollapse string // expected collapse text when wantExpand is false
	}{
		{
			name:       "3 items preserved",
			input:      `{"arr": [1, 2, 3]}`,
			wantExpand: true,
		},
		{
			name:       "5 items preserved (at boundary)",
			input:      `{"arr": [1, 2, 3, 4, 5]}`,
			wantExpand: true,
		},
		{
			name:         "6 items collapsed",
			input:        `{"arr": [1, 2, 3, 4, 5, 6]}`,
			wantExpand:   false,
			wantCollapse: "/* 6 items */",
		},
		{
			name:         "10 items collapsed",
			input:        `{"arr": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]}`,
			wantExpand:   false,
			wantCollapse: "/* 10 items */",
		},
		{
			name:       "empty array preserved",
			input:      `{"arr": []}`,
			wantExpand: true,
		},
	}

	c := NewJSONCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source

			if tt.wantExpand {
				assert.NotContains(t, src, "/* ", "array should be expanded")
			} else {
				assert.Contains(t, src, tt.wantCollapse,
					"array should be collapsed with correct count")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Depth limiting
// ---------------------------------------------------------------------------

func TestJSONCompressor_DepthLimit(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantCollapse bool
		collapseText string
	}{
		{
			name:         "depth 0 object preserved",
			input:        `{"a": 1}`,
			wantCollapse: false,
		},
		{
			name:         "depth 1 nested object preserved",
			input:        `{"outer": {"inner": "value"}}`,
			wantCollapse: false,
		},
		{
			name:         "depth 2 nested object collapsed",
			input:        `{"l0": {"l1": {"l2": "deep"}}}`,
			wantCollapse: true,
			collapseText: "/* object with 1 keys */",
		},
		{
			name:         "depth 2 nested object with multiple keys",
			input:        `{"l0": {"l1": {"a": 1, "b": 2, "c": 3}}}`,
			wantCollapse: true,
			collapseText: "/* object with 3 keys */",
		},
		{
			name:         "depth 2 nested array collapsed",
			input:        `{"l0": {"l1": [1, 2, 3]}}`,
			wantCollapse: true,
			collapseText: "/* 3 items */",
		},
	}

	c := NewJSONCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source

			if tt.wantCollapse {
				assert.Contains(t, src, tt.collapseText,
					"expected depth-limited collapse text in output")
			} else {
				assert.NotContains(t, src, "/* object with",
					"should not collapse at this depth")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestJSONCompressor_ContextCancellation(t *testing.T) {
	c := NewJSONCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.Compress(ctx, []byte(`{"key": "value"}`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestJSONCompressor_ContextCancellationOnEmptyInput(t *testing.T) {
	c := NewJSONCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Compress(ctx, []byte{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Primitive type preservation
// ---------------------------------------------------------------------------

func TestJSONCompressor_NumbersAndBooleans(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "integer",
			input: `{"count": 42}`,
			want:  "42",
		},
		{
			name:  "float",
			input: `{"ratio": 3.14}`,
			want:  "3.14",
		},
		{
			name:  "negative number",
			input: `{"offset": -10}`,
			want:  "-10",
		},
		{
			name:  "zero",
			input: `{"zero": 0}`,
			want:  ": 0",
		},
		{
			name:  "boolean true",
			input: `{"enabled": true}`,
			want:  "true",
		},
		{
			name:  "boolean false",
			input: `{"disabled": false}`,
			want:  "false",
		},
		{
			name:  "scientific notation",
			input: `{"big": 1e10}`,
			want:  "1",
		},
	}

	c := NewJSONCompressor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := c.Compress(context.Background(), []byte(tt.input))
			require.NoError(t, err)
			require.Len(t, output.Signatures, 1)

			src := output.Signatures[0].Source
			assert.Contains(t, src, tt.want, "primitive value should be preserved")

			// Verify the output is valid JSON.
			var parsed interface{}
			err = json.Unmarshal([]byte(src), &parsed)
			assert.NoError(t, err, "compressed output should be valid JSON")
		})
	}
}

func TestJSONCompressor_NullValue(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"value": null, "other": "text"}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)

	src := output.Signatures[0].Source
	assert.Contains(t, src, "null", "null value should be preserved")

	// Verify the output is valid JSON with null preserved.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(src), &parsed)
	require.NoError(t, err)
	assert.Nil(t, parsed["value"], "null should remain null in parsed output")
}

// ---------------------------------------------------------------------------
// Golden tests using fixture files
// ---------------------------------------------------------------------------

func TestJSONCompressor_GoldenPackageJSON(t *testing.T) {
	c := NewJSONCompressor()
	source := readFixture(t, "json/package.json")

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindDocComment, sig.Kind)
	assert.Equal(t, "json-skeleton", sig.Name)
	assert.Equal(t, "json", output.Language)

	// Verify the output is valid JSON.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(sig.Source), &parsed)
	require.NoError(t, err, "golden output should be valid JSON")

	// Top-level keys should all be present.
	for _, key := range []string{"name", "version", "description", "main", "scripts", "dependencies", "devDependencies", "keywords", "author", "license"} {
		assert.Contains(t, parsed, key, "top-level key %q should be present", key)
	}

	// "scripts" is at depth 1, so its values should be present.
	scripts, ok := parsed["scripts"].(map[string]interface{})
	require.True(t, ok, "scripts should be an object")
	assert.Contains(t, scripts, "dev")
	assert.Contains(t, scripts, "build")

	// "dependencies" is at depth 1, values are strings at depth 1 - should be preserved.
	deps, ok := parsed["dependencies"].(map[string]interface{})
	require.True(t, ok, "dependencies should be an object")
	assert.Contains(t, deps, "next")
	assert.Contains(t, deps, "react")

	// "keywords" has 3 items, should be expanded (3 <= 5).
	keywords, ok := parsed["keywords"].([]interface{})
	require.True(t, ok, "keywords should be an array")
	assert.Len(t, keywords, 3)

	// "description" is a long string (>50 chars) that should be truncated.
	desc, ok := parsed["description"].(string)
	require.True(t, ok, "description should be a string")
	assert.True(t, strings.HasSuffix(desc, "..."),
		"long description should be truncated with '...' suffix, got: %s", desc)
}

func TestJSONCompressor_GoldenTsconfig(t *testing.T) {
	c := NewJSONCompressor()
	source := readFixture(t, "json/tsconfig.json")

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, "json-skeleton", sig.Name)

	// Verify the output is valid JSON.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(sig.Source), &parsed)
	require.NoError(t, err, "golden output should be valid JSON")

	// Top-level keys.
	assert.Contains(t, parsed, "compilerOptions")
	assert.Contains(t, parsed, "include")
	assert.Contains(t, parsed, "exclude")

	// "compilerOptions" at depth 1 should be an object.
	compilerOpts, ok := parsed["compilerOptions"].(map[string]interface{})
	require.True(t, ok, "compilerOptions should be an object")

	// Boolean values inside compilerOptions (depth 1) should be preserved.
	assert.Equal(t, true, compilerOpts["allowJs"])
	assert.Equal(t, true, compilerOpts["strict"])
	assert.Equal(t, true, compilerOpts["noEmit"])

	// String values inside compilerOptions should be preserved.
	assert.Equal(t, "ES2022", compilerOpts["target"])

	// "lib" is an array at depth 2 (inside compilerOptions), should be collapsed.
	lib, ok := compilerOpts["lib"].(string)
	require.True(t, ok, "lib at depth 2 should be collapsed to a string summary")
	assert.Contains(t, lib, "/* 3 items */")

	// "paths" at depth 2 should be collapsed to "/* object with N keys */".
	paths, ok := compilerOpts["paths"].(string)
	require.True(t, ok, "paths at depth 2 should be collapsed to a string summary")
	assert.Contains(t, paths, "/* object with 3 keys */")

	// "include" is an array at depth 0 with 3 items, should be expanded.
	include, ok := parsed["include"].([]interface{})
	require.True(t, ok, "include should be an array")
	assert.Len(t, include, 3)

	// "exclude" is an array at depth 0 with 1 item, should be expanded.
	exclude, ok := parsed["exclude"].([]interface{})
	require.True(t, ok, "exclude should be an array")
	assert.Len(t, exclude, 1)
	assert.Equal(t, "node_modules", exclude[0])
}

func TestJSONCompressor_GoldenSimple(t *testing.T) {
	c := NewJSONCompressor()
	source := readFixture(t, "json/simple.json")

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]

	// Verify the output is valid JSON.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(sig.Source), &parsed)
	require.NoError(t, err)

	// All values should be fully preserved (everything is small).
	assert.Equal(t, "test", parsed["name"])
	assert.Equal(t, float64(1), parsed["version"]) // JSON numbers are float64
	assert.Equal(t, true, parsed["enabled"])

	items, ok := parsed["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
	assert.Equal(t, float64(1), items[0])
	assert.Equal(t, float64(2), items[1])
	assert.Equal(t, float64(3), items[2])
}

// ---------------------------------------------------------------------------
// Compression ratio
// ---------------------------------------------------------------------------

func TestJSONCompressor_CompressionRatio(t *testing.T) {
	c := NewJSONCompressor()
	source := readFixture(t, "json/package.json")

	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	ratio := output.CompressionRatio()
	// The package.json fixture is moderately sized. The ratio should be
	// positive (non-zero output) and reasonable (not exploding in size).
	assert.Greater(t, ratio, 0.0, "ratio should be > 0")
	assert.Less(t, ratio, 2.0, "output should not be excessively larger than input")

	// Both sizes should be positive.
	assert.Greater(t, output.OriginalSize, 0)
	assert.Greater(t, output.OutputSize, 0)
}

func TestJSONCompressor_CompressionRatioDeepNesting(t *testing.T) {
	// Build a deeply nested JSON structure that should compress significantly.
	input := `{
  "level0": {
    "level1": {
      "level2a": {"a": 1, "b": 2, "c": 3, "d": 4},
      "level2b": {"x": "` + strings.Repeat("y", 100) + `"},
      "level2c": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    }
  }
}`

	c := NewJSONCompressor()
	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	ratio := output.CompressionRatio()
	// Deep nesting + long string + large array should yield good compression.
	assert.Less(t, ratio, 0.8, "deeply nested JSON should compress well")
}

// ---------------------------------------------------------------------------
// Output structure validation
// ---------------------------------------------------------------------------

func TestJSONCompressor_OutputStructure(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"name": "test", "count": 42}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	assert.Equal(t, "json", output.Language)
	assert.Equal(t, len(input), output.OriginalSize)
	assert.Equal(t, 1, output.NodeCount)
	assert.Greater(t, output.OutputSize, 0)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindDocComment, sig.Kind)
	assert.Equal(t, "json-skeleton", sig.Name)
	assert.Equal(t, 1, sig.StartLine)
	assert.Greater(t, sig.EndLine, 0)
}

func TestJSONCompressor_StartLineEndLine(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"a": 1, "b": 2, "c": 3}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, 1, sig.StartLine)
	// json.MarshalIndent will produce multiple lines.
	assert.GreaterOrEqual(t, sig.EndLine, 1)

	// Count lines in the source to verify EndLine.
	lines := strings.Count(sig.Source, "\n") + 1
	assert.Equal(t, lines, sig.EndLine)
}

// ---------------------------------------------------------------------------
// Interface compliance
// ---------------------------------------------------------------------------

func TestJSONCompressor_InterfaceCompliance(t *testing.T) {
	// Verify that JSONCompressor satisfies LanguageCompressor at compile time.
	var _ LanguageCompressor = (*JSONCompressor)(nil)
	var _ LanguageCompressor = NewJSONCompressor()
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestJSONCompressor_TopLevelArray(t *testing.T) {
	c := NewJSONCompressor()
	input := `[1, 2, 3, 4, 5]`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, "json-skeleton", sig.Name)

	// 5 items, should be expanded.
	var parsed []interface{}
	err = json.Unmarshal([]byte(sig.Source), &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 5)
}

func TestJSONCompressor_TopLevelLargeArray(t *testing.T) {
	c := NewJSONCompressor()
	input := `[1, 2, 3, 4, 5, 6, 7]`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	// 7 items > 5, should be collapsed.
	assert.Contains(t, sig.Source, "/* 7 items */")
}

func TestJSONCompressor_TopLevelString(t *testing.T) {
	c := NewJSONCompressor()
	input := `"hello world"`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	assert.Contains(t, output.Signatures[0].Source, "hello world")
}

func TestJSONCompressor_TopLevelNumber(t *testing.T) {
	c := NewJSONCompressor()
	input := `42`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	assert.Contains(t, output.Signatures[0].Source, "42")
}

func TestJSONCompressor_TopLevelBoolean(t *testing.T) {
	c := NewJSONCompressor()
	input := `true`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	assert.Contains(t, output.Signatures[0].Source, "true")
}

func TestJSONCompressor_TopLevelNull(t *testing.T) {
	c := NewJSONCompressor()
	input := `null`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	assert.Contains(t, output.Signatures[0].Source, "null")
}

func TestJSONCompressor_UnicodeStrings(t *testing.T) {
	c := NewJSONCompressor()
	// 51 Unicode characters (each > 1 byte in UTF-8 but 1 rune).
	longUnicode := strings.Repeat("\u00e9", 51) // e-acute, 51 runes
	input := `{"text": "` + longUnicode + `"}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	src := output.Signatures[0].Source
	// Should be truncated to 50 runes + "...".
	assert.Contains(t, src, strings.Repeat("\u00e9", 50)+"...")
	assert.NotContains(t, src, strings.Repeat("\u00e9", 51))
}

func TestJSONCompressor_EmptyObject(t *testing.T) {
	c := NewJSONCompressor()
	input := `{}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	// Empty object should round-trip fine.
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output.Signatures[0].Source), &parsed)
	require.NoError(t, err)
	assert.Empty(t, parsed)
}

func TestJSONCompressor_EmptyArray(t *testing.T) {
	c := NewJSONCompressor()
	input := `[]`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	var parsed []interface{}
	err = json.Unmarshal([]byte(output.Signatures[0].Source), &parsed)
	require.NoError(t, err)
	assert.Empty(t, parsed)
}

func TestJSONCompressor_NestedEmptyStructures(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"empty_obj": {}, "empty_arr": [], "nested": {"also_empty": {}}}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output.Signatures[0].Source), &parsed)
	require.NoError(t, err)

	// Empty object at depth 1 should be preserved.
	emptyObj, ok := parsed["empty_obj"].(map[string]interface{})
	require.True(t, ok)
	assert.Empty(t, emptyObj)

	// Empty array at depth 1 should be preserved.
	emptyArr, ok := parsed["empty_arr"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, emptyArr)
}

func TestJSONCompressor_MixedArrayTypes(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"mixed": [1, "two", true, null, 3.14]}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output.Signatures[0].Source), &parsed)
	require.NoError(t, err)

	mixed, ok := parsed["mixed"].([]interface{})
	require.True(t, ok)
	assert.Len(t, mixed, 5) // 5 <= maxArrayDisplay
	assert.Equal(t, float64(1), mixed[0])
	assert.Equal(t, "two", mixed[1])
	assert.Equal(t, true, mixed[2])
	assert.Nil(t, mixed[3])
	assert.Equal(t, 3.14, mixed[4])
}

func TestJSONCompressor_WhitespaceOnlyInput(t *testing.T) {
	c := NewJSONCompressor()
	// Whitespace-only is not valid JSON.
	input := "   \n\t  \n  "

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	// Should fall back to raw content.
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, "json-raw", output.Signatures[0].Name)
}

func TestJSONCompressor_DepthBoundaryExact(t *testing.T) {
	// Test the exact boundary: maxDepth=2 means depth 0, 1 are preserved,
	// depth 2 collapses.
	c := NewJSONCompressor()

	// Depth 0 -> depth 1 -> depth 2: the inner object at depth 2 collapses.
	input := `{
  "d0_key": {
    "d1_key": {
      "d2_key": "value"
    }
  }
}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	require.Len(t, output.Signatures, 1)
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(output.Signatures[0].Source), &parsed)
	require.NoError(t, err)

	d0, ok := parsed["d0_key"].(map[string]interface{})
	require.True(t, ok, "d0 should be a preserved object")

	// d1_key's value at depth 2 should be a string collapse marker.
	d1Val, ok := d0["d1_key"].(string)
	require.True(t, ok, "d1_key value should be collapsed to a string, got %T", d0["d1_key"])
	assert.Equal(t, "/* object with 1 keys */", d1Val)
}

// ---------------------------------------------------------------------------
// Render integration
// ---------------------------------------------------------------------------

func TestJSONCompressor_Render(t *testing.T) {
	c := NewJSONCompressor()
	input := `{"name": "test"}`

	output, err := c.Compress(context.Background(), []byte(input))
	require.NoError(t, err)

	rendered := output.Render()
	assert.NotEmpty(t, rendered)
	// Render should produce the same content as the single signature's Source.
	assert.Equal(t, output.Signatures[0].Source, rendered)
}

// ---------------------------------------------------------------------------
// Defaults verification
// ---------------------------------------------------------------------------

func TestJSONCompressor_Defaults(t *testing.T) {
	c := NewJSONCompressor()
	assert.Equal(t, 2, c.maxDepth)
	assert.Equal(t, 50, c.maxStringLen)
	assert.Equal(t, 5, c.maxArrayDisplay)
}

// ---------------------------------------------------------------------------
// countLines helper
// ---------------------------------------------------------------------------

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"trailing newline", "hello\n", 2},
		{"three lines", "a\nb\nc", 3},
		{"only newlines", "\n\n\n", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// truncateString helper
// ---------------------------------------------------------------------------

func TestJSONCompressor_TruncateString(t *testing.T) {
	c := NewJSONCompressor()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"short", "hello", "hello"},
		{"exactly 50", strings.Repeat("a", 50), strings.Repeat("a", 50)},
		{"51 chars", strings.Repeat("b", 51), strings.Repeat("b", 50) + "..."},
		{"100 chars", strings.Repeat("c", 100), strings.Repeat("c", 50) + "..."},
		{"unicode runes", strings.Repeat("\u00e9", 51), strings.Repeat("\u00e9", 50) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.truncateString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkJSONCompress(b *testing.B) {
	c := NewJSONCompressor()
	source := readFixtureB(b, "json/package.json")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONCompress_Small(b *testing.B) {
	c := NewJSONCompressor()
	source := []byte(`{"name": "test", "version": 1, "enabled": true}`)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONCompress_Tsconfig(b *testing.B) {
	c := NewJSONCompressor()
	source := readFixtureB(b, "json/tsconfig.json")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}