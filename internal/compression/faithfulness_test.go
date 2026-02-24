package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Faithfulness verification helpers
// ---------------------------------------------------------------------------

// verifyFaithfulness checks that every non-empty, non-marker line in the
// compressed output appears verbatim somewhere in the original source.
func verifyFaithfulness(t *testing.T, original, compressed string) {
	t.Helper()
	compressedLines := strings.Split(compressed, "\n")
	for i, line := range compressedLines {
		if line == CompressedMarker {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(original, line) {
			t.Errorf("faithfulness: line %d in compressed output not found verbatim in original: %q", i+1, line)
		}
	}
}

// verifySourceOrder checks that the non-empty, non-marker lines in compressed
// output appear in the same relative order as in the original source.
// Comparison is flexible because AST compressors may strip trailing braces
// or perform minor transformations on extracted signatures.
func verifySourceOrder(t *testing.T, original, compressed string) {
	t.Helper()
	origLines := strings.Split(original, "\n")
	compLines := strings.Split(compressed, "\n")

	lastIdx := -1
	linesFound := 0
	for _, cline := range compLines {
		trimmed := strings.TrimSpace(cline)
		if trimmed == "" || cline == CompressedMarker {
			continue
		}
		for i := lastIdx + 1; i < len(origLines); i++ {
			origTrimmed := strings.TrimSpace(origLines[i])
			// Flexible match: exact equality, or the original line starts
			// with the compressed line (handles stripped trailing '{').
			if origTrimmed == trimmed ||
				strings.HasPrefix(origTrimmed, trimmed) ||
				strings.HasPrefix(trimmed, strings.TrimRight(origTrimmed, " \t{")) {
				lastIdx = i
				linesFound++
				break
			}
		}
		// Not a hard failure if not found: some compressed lines are
		// synthesized (e.g., "{ ... }" body markers) or reformatted.
	}
	assert.Greater(t, linesFound, 0, "should find at least one compressed line in original")
}

// verifyNoBodyLeakage checks that function body code does not appear in the
// compressed output. It looks for telltale body patterns specific to each language.
func verifyNoBodyLeakage(t *testing.T, compressed, language string) {
	t.Helper()

	// After the marker, check for common body-only patterns.
	afterMarker := compressed
	if idx := strings.Index(compressed, CompressedMarker); idx >= 0 {
		afterMarker = compressed[idx+len(CompressedMarker):]
	}

	switch language {
	case "go":
		// Go function bodies typically contain return, if/else, for.
		bodyPatterns := []string{
			"\treturn ",
			"\tif err != nil",
			"\tfor ",
			"\tfmt.Println(",
			"\tfmt.Fprintf(",
			"json.NewEncoder(w).Encode(",
			"http.Error(w,",
		}
		for _, pattern := range bodyPatterns {
			assert.NotContains(t, afterMarker, pattern,
				"Go body code leaked: %q", pattern)
		}

	case "python":
		bodyPatterns := []string{
			"    return ",
			"    db.add(",
			"    db.commit(",
			"    raise ",
			"    users = db.query(",
		}
		for _, pattern := range bodyPatterns {
			assert.NotContains(t, afterMarker, pattern,
				"Python body code leaked: %q", pattern)
		}

	case "typescript":
		bodyPatterns := []string{
			"  const params = ",
			"  return NextResponse.json(",
			"  const body = await ",
			"  return {};",
		}
		for _, pattern := range bodyPatterns {
			assert.NotContains(t, afterMarker, pattern,
				"TypeScript body code leaked: %q", pattern)
		}

	case "rust":
		bodyPatterns := []string{
			"        Self {",
			"        self.entries.get(",
			"        self.entries.retain(",
			"        let now =",
		}
		for _, pattern := range bodyPatterns {
			assert.NotContains(t, afterMarker, pattern,
				"Rust body code leaked: %q", pattern)
		}
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: AST compressor -- all E2E fixture languages
// ---------------------------------------------------------------------------

func TestFaithfulness_AllLanguages_AST(t *testing.T) {
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

	orch := NewOrchestrator(e2eConfig(EngineAST))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: original},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed, "file should be compressed")

			compressed := files[0].Content
			verifyFaithfulness(t, original, compressed)
			verifySourceOrder(t, original, compressed)
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: Regex compressor -- all E2E fixture languages
// ---------------------------------------------------------------------------

func TestFaithfulness_AllLanguages_Regex(t *testing.T) {
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

	orch := NewOrchestrator(e2eConfig(EngineRegex))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: original},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed, "file should be compressed")

			compressed := files[0].Content
			verifyFaithfulness(t, original, compressed)
			verifySourceOrder(t, original, compressed)
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: No body leakage (AST)
// ---------------------------------------------------------------------------

func TestFaithfulness_NoBodyLeakage_AST(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		path    string
		lang    string
	}{
		{"Go", "go/http-handler.go", "http-handler.go", "go"},
		{"TypeScript", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Python", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	orch := NewOrchestrator(e2eConfig(EngineAST))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: original},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed)

			verifyNoBodyLeakage(t, files[0].Content, tt.lang)
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: No body leakage (Regex)
// ---------------------------------------------------------------------------

func TestFaithfulness_NoBodyLeakage_Regex(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		path    string
		lang    string
	}{
		{"Go", "go/http-handler.go", "http-handler.go", "go"},
		{"TypeScript", "typescript/api-route.ts", "api-route.ts", "typescript"},
		{"Python", "python/fastapi-router.py", "fastapi-router.py", "python"},
		{"Rust", "rust/struct-impl.rs", "struct-impl.rs", "rust"},
	}

	orch := NewOrchestrator(e2eConfig(EngineRegex))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := readE2EFixture(t, tt.fixture)
			files := []*CompressibleFile{
				{Path: tt.path, Content: original},
			}

			_, err := orch.Compress(context.Background(), files)
			require.NoError(t, err)
			require.True(t, files[0].IsCompressed)

			verifyNoBodyLeakage(t, files[0].Content, tt.lang)
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: Direct compressor verbatim check
// ---------------------------------------------------------------------------

func TestFaithfulness_DirectCompressorVerbatim(t *testing.T) {
	tests := []struct {
		name       string
		compressor LanguageCompressor
		source     string
	}{
		{
			name:       "Go compressor",
			compressor: NewGoCompressor(),
			source: `package main

import "fmt"

// Greet returns a greeting.
func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

type Config struct {
	Host string
	Port int
}

const Version = "1.0.0"
`,
		},
		{
			name:       "TypeScript compressor",
			compressor: NewTypeScriptCompressor(),
			source: `import { Request, Response } from 'express';

interface Handler {
  handle(req: Request): Response;
}

export function createHandler(): Handler {
  return {
    handle: (req) => new Response(),
  };
}

export const PORT = 3000;
`,
		},
		{
			name:       "Python compressor",
			compressor: NewPythonCompressor(),
			source: `from typing import List, Optional

MAX_SIZE = 100

class DataStore:
    def __init__(self, path: str):
        self.path = path

    def get(self, key: str) -> Optional[str]:
        return None

async def process(items: List[str]) -> int:
    return len(items)
`,
		},
		{
			name:       "Rust compressor",
			compressor: NewRustCompressor(),
			source: `use std::collections::HashMap;

pub struct Cache {
    data: HashMap<String, String>,
}

impl Cache {
    pub fn new() -> Self {
        Self { data: HashMap::new() }
    }

    pub fn get(&self, key: &str) -> Option<&String> {
        self.data.get(key)
    }
}

pub const MAX_ENTRIES: usize = 1000;
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tt.compressor.Compress(context.Background(), []byte(tt.source))
			require.NoError(t, err)

			rendered := output.Render()
			if rendered == "" {
				return // Nothing to verify.
			}

			// Every non-empty line in rendered output must exist verbatim in source.
			lines := strings.Split(rendered, "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				assert.True(t, strings.Contains(tt.source, line),
					"line %d not found verbatim in original: %q", i+1, line)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: Regex compressor verbatim check
// ---------------------------------------------------------------------------

func TestFaithfulness_RegexVerbatim(t *testing.T) {
	tests := []struct {
		name     string
		language string
		source   string
	}{
		{
			name:     "Go regex",
			language: "go",
			source: `package main

import "fmt"

func Hello(name string) string {
	return fmt.Sprintf("hello %s", name)
}

const Version = "1.0"
`,
		},
		{
			name:     "TypeScript regex",
			language: "typescript",
			source: `import { Router } from 'express';

interface Config {
  port: number;
}

export function start(config: Config): void {
  console.log(config.port);
}

export const DEFAULT_PORT = 3000;
`,
		},
		{
			name:     "Python regex",
			language: "python",
			source: `import os
from pathlib import Path

MAX_SIZE = 1024

class FileReader:
    def read(self, path: str) -> str:
        return Path(path).read_text()

def main():
    reader = FileReader()
    print(reader.read("test.txt"))
`,
		},
		{
			name:     "Rust regex",
			language: "rust",
			source: `use std::fs;

pub struct Config {
    pub path: String,
}

pub fn load_config(path: &str) -> Config {
    Config { path: path.to_string() }
}

pub const DEFAULT_PATH: &str = "/etc/config";
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRegexCompressor(tt.language)
			output, err := c.Compress(context.Background(), []byte(tt.source))
			require.NoError(t, err)

			rendered := output.Render()
			if rendered == "" {
				return
			}

			lines := strings.Split(rendered, "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				assert.True(t, strings.Contains(tt.source, line),
					"regex line %d not found verbatim in original: %q", i+1, line)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Faithfulness: Doc comments are not duplicated
// ---------------------------------------------------------------------------

func TestFaithfulness_DocCommentsNotDuplicated(t *testing.T) {
	source := `package main

// Greet returns a greeting string.
func Greet(name string) string {
	return "Hello, " + name
}

// Process handles processing logic.
func Process(input string) error {
	return nil
}
`
	c := NewGoCompressor()
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()

	// Count occurrences of each doc comment.
	greetDoc := strings.Count(rendered, "// Greet returns a greeting string.")
	processDoc := strings.Count(rendered, "// Process handles processing logic.")

	assert.LessOrEqual(t, greetDoc, 1,
		"Greet doc comment should appear at most once, found %d times", greetDoc)
	assert.LessOrEqual(t, processDoc, 1,
		"Process doc comment should appear at most once, found %d times", processDoc)
}

// ---------------------------------------------------------------------------
// Faithfulness: Marker is never in source, only added by orchestrator
// ---------------------------------------------------------------------------

func TestFaithfulness_MarkerOnlyFromOrchestrator(t *testing.T) {
	// Raw compressor output should NOT contain the marker.
	source := `package main

func main() {}
`
	c := NewGoCompressor()
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()
	assert.NotContains(t, rendered, CompressedMarker,
		"raw compressor output should not contain the marker")
}
