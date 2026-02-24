package compression

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdataDir returns the absolute path to the testdata directory.
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata", "compression")
}

// readFixture reads a test fixture file.
func readFixture(t *testing.T, relPath string) []byte {
	t.Helper()
	path := filepath.Join(testdataDir(), relPath)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read fixture %s", relPath)
	return data
}

// readExpected reads the expected output file and trims trailing whitespace.
func readExpected(t *testing.T, relPath string) string {
	t.Helper()
	path := filepath.Join(testdataDir(), relPath)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read expected %s", relPath)
	return strings.TrimSpace(string(data))
}

// ---------------------------------------------------------------------------
// TypeScript golden tests
// ---------------------------------------------------------------------------

func TestTypeScriptCompressor_GoldenTests(t *testing.T) {
	compressor := NewTypeScriptCompressor()
	ctx := context.Background()

	tests := []struct {
		name     string
		fixture  string
		expected string
	}{
		{
			name:     "api route",
			fixture:  "typescript/api-route.ts",
			expected: "typescript/api-route.ts.expected",
		},
		{
			name:     "react component",
			fixture:  "typescript/react-component.tsx",
			expected: "typescript/react-component.tsx.expected",
		},
		{
			name:     "service class",
			fixture:  "typescript/service-class.ts",
			expected: "typescript/service-class.ts.expected",
		},
		{
			name:     "barrel file",
			fixture:  "typescript/barrel-file.ts",
			expected: "typescript/barrel-file.ts.expected",
		},
		{
			name:     "types and enums",
			fixture:  "typescript/types-and-enums.ts",
			expected: "typescript/types-and-enums.ts.expected",
		},
		{
			name:     "arrow functions",
			fixture:  "typescript/arrow-functions.ts",
			expected: "typescript/arrow-functions.ts.expected",
		},
		{
			name:     "class inheritance",
			fixture:  "typescript/class-inheritance.ts",
			expected: "typescript/class-inheritance.ts.expected",
		},
		{
			name:     "default exports",
			fixture:  "typescript/default-exports.ts",
			expected: "typescript/default-exports.ts.expected",
		},
		{
			name:     "doc comments",
			fixture:  "typescript/doc-comments.ts",
			expected: "typescript/doc-comments.ts.expected",
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
// TypeScript unit tests
// ---------------------------------------------------------------------------

func TestTypeScriptCompressor_Language(t *testing.T) {
	c := NewTypeScriptCompressor()
	assert.Equal(t, "typescript", c.Language())
}

func TestTypeScriptCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewTypeScriptCompressor()
	types := c.SupportedNodeTypes()
	assert.Contains(t, types, "function_declaration")
	assert.Contains(t, types, "class_declaration")
	assert.Contains(t, types, "interface_declaration")
	assert.Contains(t, types, "type_alias_declaration")
	assert.Contains(t, types, "enum_declaration")
	assert.Contains(t, types, "import_statement")
	assert.Contains(t, types, "export_statement")
}

func TestTypeScriptCompressor_EmptyInput(t *testing.T) {
	c := NewTypeScriptCompressor()
	output, err := c.Compress(context.Background(), []byte{})
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
	assert.Equal(t, 0, output.OriginalSize)
	assert.Equal(t, "typescript", output.Language)
}

func TestTypeScriptCompressor_EmptyFile(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := readFixture(t, "typescript/empty-file.ts")
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)
	assert.Empty(t, output.Signatures)
}

func TestTypeScriptCompressor_ContextCancellation(t *testing.T) {
	c := NewTypeScriptCompressor()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Generate a large source to ensure cancellation is checked.
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("const x = 1;\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTypeScriptCompressor_InterfaceExtraction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `interface User {
  id: string;
  name: string;
  email: string;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindInterface, output.Signatures[0].Kind)
	assert.Equal(t, "User", output.Signatures[0].Name)
	assert.Contains(t, output.Signatures[0].Source, "id: string;")
}

func TestTypeScriptCompressor_TypeAliasExtraction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `type Result<T> = Success<T> | Failure;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindType, output.Signatures[0].Kind)
	assert.Equal(t, "Result", output.Signatures[0].Name)
}

func TestTypeScriptCompressor_EnumExtraction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `enum Direction {
  Up = 'UP',
  Down = 'DOWN',
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	assert.Equal(t, KindType, output.Signatures[0].Kind)
	assert.Equal(t, "Direction", output.Signatures[0].Name)
	assert.Contains(t, output.Signatures[0].Source, "Up = 'UP'")
}

func TestTypeScriptCompressor_FunctionDeclaration(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `function greet(name: string): string {
  return "Hello, " + name;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "greet", sig.Name)
	assert.NotContains(t, sig.Source, "return")
	assert.Contains(t, sig.Source, "function greet(name: string): string")
}

func TestTypeScriptCompressor_ArrowFunction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `const handler = async (req: Request): Promise<Response> => {
  const body = await req.json();
  return new Response(JSON.stringify(body));
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "handler", sig.Name)
	assert.Contains(t, sig.Source, "=>")
	assert.Contains(t, sig.Source, "{ ... }")
}

func TestTypeScriptCompressor_ArrowFunctionExpression(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `const add = (a: number, b: number): number => a + b;`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "add", sig.Name)
	assert.Contains(t, sig.Source, "=> a + b")
}

func TestTypeScriptCompressor_ClassDeclaration(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `class MyService {
  private db: Database;

  constructor(db: Database) {
    this.db = db;
  }

  async find(id: string): Promise<Item> {
    return this.db.get(id);
  }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindClass, sig.Kind)
	assert.Equal(t, "MyService", sig.Name)
	assert.Contains(t, sig.Source, "private db: Database;")
	assert.Contains(t, sig.Source, "constructor(db: Database) { ... }")
	assert.Contains(t, sig.Source, "async find(id: string): Promise<Item> { ... }")
	assert.NotContains(t, sig.Source, "this.db = db")
}

func TestTypeScriptCompressor_ImportStatements(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `import { useState } from 'react';
import type { FC } from 'react';`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 2)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
	assert.Equal(t, KindImport, output.Signatures[1].Kind)
}

func TestTypeScriptCompressor_ExportStatements(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `export * from './utils';
export { foo, bar } from './module';
export type { Config } from './types';`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 3)
	for _, sig := range output.Signatures {
		assert.Equal(t, KindExport, sig.Kind)
	}
}

func TestTypeScriptCompressor_DocCommentAttachment(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `/** Does something important */
function doStuff(): void {
  console.log("stuff");
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Contains(t, sig.Source, "/** Does something important */")
	assert.Contains(t, sig.Source, "function doStuff(): void")
}

func TestTypeScriptCompressor_DecoratorAttachment(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `@Injectable()
class MyService {
  doWork(): void {
    return;
  }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Contains(t, sig.Source, "@Injectable()")
	assert.Equal(t, KindClass, sig.Kind)
}

func TestTypeScriptCompressor_TopLevelConst(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `const API_URL: string = "https://api.example.com";`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindConstant, sig.Kind)
	assert.Equal(t, "API_URL", sig.Name)
	assert.Contains(t, sig.Source, "API_URL: string")
	assert.NotContains(t, sig.Source, "https://")
}

func TestTypeScriptCompressor_SourceOrderPreserved(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `import { x } from 'y';

interface Foo {
  bar: string;
}

function baz(): void {
  return;
}

export { Foo, baz };`

	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 4)
	assert.Equal(t, KindImport, output.Signatures[0].Kind)
	assert.Equal(t, KindInterface, output.Signatures[1].Kind)
	assert.Equal(t, KindFunction, output.Signatures[2].Kind)
	assert.Equal(t, KindExport, output.Signatures[3].Kind)
}

func TestTypeScriptCompressor_CompressionRatio(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := readFixture(t, "typescript/api-route.ts")
	output, err := c.Compress(context.Background(), source)
	require.NoError(t, err)

	output.Render()
	ratio := output.CompressionRatio()
	// Expect 30-70% of original size (50-70% reduction).
	assert.Greater(t, ratio, 0.1, "compression ratio should be > 0.1")
	assert.Less(t, ratio, 0.75, "compression ratio should be < 0.75 (at least 25%% reduction)")
}

func TestTypeScriptCompressor_ExportedFunction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `export function hello(name: string): string {
  return "Hello " + name;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "hello", sig.Name)
	assert.Contains(t, sig.Source, "export function hello(name: string): string")
}

func TestTypeScriptCompressor_ExportedArrowFunction(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `export const handler = async (req: Request): Promise<Response> => {
  return new Response("ok");
};`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindFunction, sig.Kind)
	assert.Equal(t, "handler", sig.Name)
	assert.Contains(t, sig.Source, "export const handler")
	assert.Contains(t, sig.Source, "{ ... }")
}

func TestTypeScriptCompressor_AbstractClass(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `abstract class Base {
  abstract doWork(): void;

  protected helper(): string {
    return "help";
  }
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindClass, sig.Kind)
	assert.Equal(t, "Base", sig.Name)
	assert.Contains(t, sig.Source, "abstract class Base")
	assert.Contains(t, sig.Source, "abstract doWork(): void")
}

func TestTypeScriptCompressor_MultiLineDocComment(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `/**
 * Processes input data.
 * @param data - The raw data
 * @returns The processed result
 */
function process(data: string): string {
  return data.trim();
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Contains(t, sig.Source, "/**")
	assert.Contains(t, sig.Source, "@param data")
	assert.Contains(t, sig.Source, "function process(data: string): string")
}

func TestTypeScriptCompressor_ExportedInterface(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `export interface Config {
  host: string;
  port: number;
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindInterface, sig.Kind)
	assert.Equal(t, "Config", sig.Name)
	assert.Contains(t, sig.Source, "export interface Config")
}

func TestTypeScriptCompressor_ConstEnum(t *testing.T) {
	c := NewTypeScriptCompressor()
	source := `const enum Color {
  Red = 'red',
  Blue = 'blue',
}`
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	require.Len(t, output.Signatures, 1)
	sig := output.Signatures[0]
	assert.Equal(t, KindType, sig.Kind)
	assert.Equal(t, "Color", sig.Name)
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkTypeScriptCompressor_APIRoute(b *testing.B) {
	c := NewTypeScriptCompressor()
	source := readFixtureB(b, "typescript/api-route.ts")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTypeScriptCompressor_ServiceClass(b *testing.B) {
	c := NewTypeScriptCompressor()
	source := readFixtureB(b, "typescript/service-class.ts")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Compress(ctx, source)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func readFixtureB(b *testing.B, relPath string) []byte {
	b.Helper()
	path := filepath.Join(testdataDir(), relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("failed to read fixture %s: %v", relPath, err)
	}
	return data
}
