package compression

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Per-language regex compression benchmarks
// ---------------------------------------------------------------------------

func BenchmarkRegexCompressor_Go(b *testing.B) {
	source := buildLargeGoSource(500)
	c := NewRegexCompressor("go")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_TypeScript(b *testing.B) {
	source := buildLargeTypeScriptSource(500)
	c := NewRegexCompressor("typescript")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_Python(b *testing.B) {
	source := buildLargePythonSource(500)
	c := NewRegexCompressor("python")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_Rust(b *testing.B) {
	source := buildLargeRustSource(500)
	c := NewRegexCompressor("rust")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_Java(b *testing.B) {
	source := buildLargeJavaSource(500)
	c := NewRegexCompressor("java")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_C(b *testing.B) {
	source := buildLargeCSource(500)
	c := NewRegexCompressor("c")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkRegexCompressor_Cpp(b *testing.B) {
	source := buildLargeCppSource(500)
	c := NewRegexCompressor("cpp")
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

// ---------------------------------------------------------------------------
// Regex vs AST comparison benchmarks
// ---------------------------------------------------------------------------

func BenchmarkRegexVsAST_Go(b *testing.B) {
	source := buildLargeGoSource(500)
	ctx := context.Background()

	b.Run("ast", func(b *testing.B) {
		c := NewGoCompressor()
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})

	b.Run("regex", func(b *testing.B) {
		c := NewRegexCompressor("go")
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})
}

func BenchmarkRegexVsAST_TypeScript(b *testing.B) {
	source := buildLargeTypeScriptSource(500)
	ctx := context.Background()

	b.Run("ast", func(b *testing.B) {
		c := NewTypeScriptCompressor()
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})

	b.Run("regex", func(b *testing.B) {
		c := NewRegexCompressor("typescript")
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})
}

func BenchmarkRegexVsAST_Python(b *testing.B) {
	source := buildLargePythonSource(500)
	ctx := context.Background()

	b.Run("ast", func(b *testing.B) {
		c := NewPythonCompressor()
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})

	b.Run("regex", func(b *testing.B) {
		c := NewRegexCompressor("python")
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})
}

func BenchmarkRegexVsAST_Rust(b *testing.B) {
	source := buildLargeRustSource(500)
	ctx := context.Background()

	b.Run("ast", func(b *testing.B) {
		c := NewRustCompressor()
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})

	b.Run("regex", func(b *testing.B) {
		c := NewRegexCompressor("rust")
		b.SetBytes(int64(len(source)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.Compress(ctx, source)
		}
	})
}

// ---------------------------------------------------------------------------
// Orchestrator batch compression benchmark
// ---------------------------------------------------------------------------

func BenchmarkOrchestrator_BatchCompression(b *testing.B) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    runtime.NumCPU(),
		Engine:         EngineAuto,
	}
	orch := NewOrchestrator(cfg)

	// Create 50 files across multiple languages.
	const fileCount = 50
	origContent := make([]string, fileCount)
	files := make([]*CompressibleFile, fileCount)

	languages := []struct {
		ext     string
		builder func(int) []byte
	}{
		{".go", buildLargeGoSource},
		{".ts", buildLargeTypeScriptSource},
		{".py", buildLargePythonSource},
		{".rs", buildLargeRustSource},
		{".java", buildLargeJavaSource},
	}

	for i := 0; i < fileCount; i++ {
		lang := languages[i%len(languages)]
		// Each file has ~100 lines of code.
		content := lang.builder(100)
		origContent[i] = string(content)
		files[i] = &CompressibleFile{
			Path:    fmt.Sprintf("pkg/file_%d%s", i, lang.ext),
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

func BenchmarkOrchestrator_BatchRegex(b *testing.B) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    runtime.NumCPU(),
		Engine:         EngineRegex,
	}
	orch := NewOrchestrator(cfg)

	const fileCount = 50
	origContent := make([]string, fileCount)
	files := make([]*CompressibleFile, fileCount)

	languages := []struct {
		ext     string
		builder func(int) []byte
	}{
		{".go", buildLargeGoSource},
		{".ts", buildLargeTypeScriptSource},
		{".py", buildLargePythonSource},
		{".rs", buildLargeRustSource},
		{".java", buildLargeJavaSource},
	}

	for i := 0; i < fileCount; i++ {
		lang := languages[i%len(languages)]
		content := lang.builder(100)
		origContent[i] = string(content)
		files[i] = &CompressibleFile{
			Path:    fmt.Sprintf("pkg/file_%d%s", i, lang.ext),
			Content: origContent[i],
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
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

// ---------------------------------------------------------------------------
// Per-file compression time sanity benchmarks
// ---------------------------------------------------------------------------

func BenchmarkPerFile_ASTCompressor_Go(b *testing.B) {
	source := buildLargeGoSource(200)
	c := NewGoCompressor()
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkPerFile_ASTCompressor_TypeScript(b *testing.B) {
	source := buildLargeTypeScriptSource(200)
	c := NewTypeScriptCompressor()
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkPerFile_ASTCompressor_Python(b *testing.B) {
	source := buildLargePythonSource(200)
	c := NewPythonCompressor()
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

func BenchmarkPerFile_ASTCompressor_Rust(b *testing.B) {
	source := buildLargeRustSource(200)
	c := NewRustCompressor()
	ctx := context.Background()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(ctx, source)
	}
}

// ---------------------------------------------------------------------------
// Source generators for benchmarks
// ---------------------------------------------------------------------------

// buildLargeGoSource generates a Go source file with approximately `lines` lines.
func buildLargeGoSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("package benchmark\n\n")
	b.WriteString("import (\n\t\"context\"\n\t\"fmt\"\n)\n\n")
	b.WriteString("// Config holds configuration.\ntype Config struct {\n\tHost string\n\tPort int\n}\n\n")

	funcsNeeded := lines / 6 // Each function is ~6 lines.
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("// Func%d does something.\n", i))
		b.WriteString(fmt.Sprintf("func Func%d(ctx context.Context, input string) (string, error) {\n", i))
		b.WriteString(fmt.Sprintf("\tresult := fmt.Sprintf(\"%%s-%%d\", input, %d)\n", i))
		b.WriteString("\tif ctx.Err() != nil {\n\t\treturn \"\", ctx.Err()\n\t}\n")
		b.WriteString("\treturn result, nil\n}\n\n")
	}
	return []byte(b.String())
}

// buildLargeTypeScriptSource generates a TypeScript source file.
func buildLargeTypeScriptSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("import { Request, Response } from 'express';\n\n")
	b.WriteString("interface Config {\n  host: string;\n  port: number;\n}\n\n")
	b.WriteString("type Status = 'active' | 'inactive';\n\n")

	funcsNeeded := lines / 5
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("export async function handler%d(req: Request): Promise<Response> {\n", i))
		b.WriteString(fmt.Sprintf("  const data = await fetch('/api/%d');\n", i))
		b.WriteString("  return new Response(JSON.stringify(data));\n")
		b.WriteString("}\n\n")
	}
	b.WriteString("export const VERSION = '1.0.0';\n")
	return []byte(b.String())
}

// buildLargePythonSource generates a Python source file.
func buildLargePythonSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("from typing import Optional, List\nimport os\n\n")
	b.WriteString("MAX_SIZE = 1000\n\n")
	b.WriteString("class DataProcessor:\n    def __init__(self, name: str):\n        self.name = name\n\n")

	funcsNeeded := lines / 4
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("async def process_%d(data: str) -> str:\n", i))
		b.WriteString(fmt.Sprintf("    result = data + '_%d'\n", i))
		b.WriteString("    return result\n\n")
	}
	return []byte(b.String())
}

// buildLargeRustSource generates a Rust source file.
func buildLargeRustSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("use std::collections::HashMap;\nuse std::fmt;\n\n")
	b.WriteString("pub struct Config {\n    pub host: String,\n    pub port: u16,\n}\n\n")
	b.WriteString("pub const MAX_SIZE: usize = 1000;\n\n")

	funcsNeeded := lines / 5
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("pub fn process_%d(input: &str) -> String {\n", i))
		b.WriteString(fmt.Sprintf("    format!(\"{{}}-%d\", input)\n", i))
		b.WriteString("}\n\n")
	}
	return []byte(b.String())
}

// buildLargeJavaSource generates a Java source file.
func buildLargeJavaSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("package com.example.bench;\n\n")
	b.WriteString("import java.util.List;\nimport java.util.Optional;\n\n")
	b.WriteString("public class BenchService {\n\n")

	funcsNeeded := lines / 5
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("    public String process%d(String input) {\n", i))
		b.WriteString(fmt.Sprintf("        return input + \"-%d\";\n", i))
		b.WriteString("    }\n\n")
	}
	b.WriteString("}\n")
	return []byte(b.String())
}

// buildLargeCSource generates a C source file.
func buildLargeCSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("#include <stdio.h>\n#include <stdlib.h>\n\n")
	b.WriteString("#define MAX_SIZE 1024\n\n")
	b.WriteString("struct Config {\n    char *host;\n    int port;\n};\n\n")

	funcsNeeded := lines / 4
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("int process_%d(int input) {\n", i))
		b.WriteString(fmt.Sprintf("    return input + %d;\n", i))
		b.WriteString("}\n\n")
	}
	return []byte(b.String())
}

// buildLargeCppSource generates a C++ source file.
func buildLargeCppSource(lines int) []byte {
	var b strings.Builder
	b.WriteString("#include <iostream>\n#include <vector>\n#include <string>\n\n")
	b.WriteString("#define VERSION \"1.0.0\"\n\n")
	b.WriteString("namespace bench {\n\n")
	b.WriteString("class Processor {\npublic:\n")

	funcsNeeded := lines / 4
	if funcsNeeded < 1 {
		funcsNeeded = 1
	}
	for i := 0; i < funcsNeeded; i++ {
		b.WriteString(fmt.Sprintf("    std::string process_%d(const std::string& input) {\n", i))
		b.WriteString(fmt.Sprintf("        return input + \"-%d\";\n", i))
		b.WriteString("    }\n\n")
	}
	b.WriteString("};\n\n} // namespace bench\n")
	return []byte(b.String())
}
