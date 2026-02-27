package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// JavaScript golden tests
// ---------------------------------------------------------------------------

func TestJavaScriptCompressor_GoldenTests(t *testing.T) {
	compressor := NewJavaScriptCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "express handler",
			fixture:  "javascript/express-handler.js",
			expected: "javascript/express-handler.js.expected",
		},
		{
			name:     "class component",
			fixture:  "javascript/class-component.js",
			expected: "javascript/class-component.js.expected",
		},
		{
			name:     "module exports",
			fixture:  "javascript/module-exports.js",
			expected: "javascript/module-exports.js.expected",
		},
		{
			name:     "arrow functions",
			fixture:  "javascript/arrow-functions.js",
			expected: "javascript/arrow-functions.js.expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := readFixture(t, tt.fixture)
			expected := readExpected(t, tt.expected)

			output, err := compressor.Compress(ctx, source)
			require.NoError(t, err)

			rendered := strings.TrimSpace(output.Render())
			assert.Equal(t, expected, rendered,
				"golden test mismatch for %s", tt.fixture)
		})
	}
}

// ---------------------------------------------------------------------------
// JavaScript unit tests
// ---------------------------------------------------------------------------

func TestJavaScriptCompressor_Language(t *testing.T) {
	c := NewJavaScriptCompressor()
	assert.Equal(t, "javascript", c.Language())
}

func TestJavaScriptCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewJavaScriptCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "function_declaration")
	assert.Contains(t, types, "class_declaration")
	assert.Contains(t, types, "import_statement")
	assert.Contains(t, types, "export_statement")
	assert.NotContains(t, types, "interface_declaration")
	assert.NotContains(t, types, "type_alias_declaration")
	assert.NotContains(t, types, "enum_declaration")
}

func TestJavaScriptCompressor_EmptyInput(t *testing.T) {
	c := NewJavaScriptCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "javascript", output.Language)
}

func TestJavaScriptCompressor_EmptyFile(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := readFixture(t, "javascript/empty-file.js")
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
}

func TestJavaScriptCompressor_NoInterfaces(t *testing.T) {
	c := NewJavaScriptCompressor()
	// JS should not extract interface declarations (TS-only).
	source := `interface Foo {
  bar: string;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	// Since JS doesn't recognize interfaces, it won't extract them.
	for _, sig := range output.Signatures {
		assert.NotEqual(t, KindInterface, sig.Kind)
	}
}

func TestJavaScriptCompressor_NoEnums(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `enum Color {
  Red,
  Blue,
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	for _, sig := range output.Signatures {
		assert.NotEqual(t, KindType, sig.Kind, "JS compressor should not extract enums")
	}
}

func TestJavaScriptCompressor_FunctionDeclaration(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `function greet(name) {
  return "Hello, " + name;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "greet", sig.Name)
	assert.NotContains(t, sig.Source, "return")
}

func TestJavaScriptCompressor_ClassDeclaration(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `class Animal {
  constructor(name) {
    this.name = name;
  }

  speak() {
    return this.name + " makes a noise.";
  }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindClass, sig.Kind)
	assert.Equal(t, "Animal", sig.Name)
	assert.Contains(t, sig.Source, "constructor(name) { ... }")
	assert.Contains(t, sig.Source, "speak() { ... }")
}

func TestJavaScriptCompressor_ImportStatements(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `import express from 'express';
import { Router } from 'express';`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
	assert.Equal(t, KindImport, output.Signatures[1].Kind)
}

func TestJavaScriptCompressor_ExportDefault(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `export default function process(data) {
  return data;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Contains(t, sig.Source, "export default function process(data)")
}

func TestJavaScriptCompressor_DocCommentAttachment(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `/** Adds two numbers */
function add(a, b) {
  return a + b;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Contains(t, sig.Source, "/** Adds two numbers */")
	assert.Contains(t, sig.Source, "function add(a, b)")
}

func TestJavaScriptCompressor_ArrowFunction(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `const handler = async (req) => {
  return new Response("ok");
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "handler", sig.Name)
	assert.Contains(t, sig.Source, "=>")
}

func TestJavaScriptCompressor_ConstDeclaration(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := `const API_URL = "https://api.example.com";`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "API_URL", sig.Name)
}

func TestJavaScriptCompressor_ContextCancellation(t *testing.T) {
	c := NewJavaScriptCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("const x = 1;\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestJavaScriptCompressor_CompressionRatio(t *testing.T) {
	c := NewJavaScriptCompressor()
	source := readFixture(t, "javascript/express-handler.js")
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.1, "compression ratio should be > 0.1")
	assert.Less(t, ratio, 0.85, "compression ratio should be < 0.85")
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkJavaScriptCompressor_ExpressHandler(b *testing.B) {
	c := NewJavaScriptCompressor()
	source := readFixtureB(b, "javascript/express-handler.js")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}
