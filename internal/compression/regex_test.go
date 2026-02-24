package compression

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// RegexCompressor metadata tests
// ---------------------------------------------------------------------------

func TestRegexCompressor_Language(t *testing.T) {
	tests := []struct {
		language string
	}{
		{"go"},
		{"typescript"},
		{"javascript"},
		{"python"},
		{"rust"},
		{"java"},
		{"c"},
		{"cpp"},
	}
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			c := NewRegexCompressor(tt.language)
			assert.Equal(t, tt.language, c.Language())
		})
	}
}

func TestRegexCompressor_SupportedNodeTypes(t *testing.T) {
	c := NewRegexCompressor("go")
	types := c.SupportedNodeTypes()
	assert.NotEmpty(t, types)
	// All types should be prefixed with "regex_".
	for _, typ := range types {
		assert.True(t, strings.HasPrefix(typ, "regex_"),
			"node type %q should have regex_ prefix", typ)
	}
}

func TestRegexCompressor_InterfaceCompliance(t *testing.T) {
	// Compile-time check is already in regex.go, but verify at runtime too.
	var _ LanguageCompressor = NewRegexCompressor("go")
}

// ---------------------------------------------------------------------------
// Go regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_Go(t *testing.T) {
	source := `package main

import (
	"fmt"
	"strings"
)

type Handler struct {
	name string
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func main() {
	fmt.Println("hello")
}

const MaxRetries = 3
`
	c := NewRegexCompressor("go")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	assert.Equal(t, "go", output.Language)
	assert.Greater(t, len(output.Signatures), 0, "should extract signatures")
	assert.Equal(t, len(source), output.OriginalSize)

	// Verify expected signature kinds are present.
	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract import")
	assert.Contains(t, kinds, KindFunction, "should extract functions")
	assert.Contains(t, kinds, KindType, "should extract type declarations")
	assert.Contains(t, kinds, KindConstant, "should extract constants")

	// Verify function names.
	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "Serve", "should extract method name")
	assert.Contains(t, names, "main", "should extract function name")
	assert.Contains(t, names, "MaxRetries", "should extract constant name")

	// Function bodies should NOT be in compressed output.
	rendered := output.Render()
	assert.NotContains(t, rendered, `fmt.Fprintf(w, "hello")`, "function body should not be in output")
	assert.NotContains(t, rendered, `fmt.Println("hello")`, "function body should not be in output")
}

func TestRegexCompressor_Go_Methods(t *testing.T) {
	source := `package pkg

func (s *Server) Start(ctx context.Context) error {
	return s.listen()
}

func (s *Server) Stop() {
	s.close()
}

func NewServer(addr string) *Server {
	return &Server{addr: addr}
}
`
	c := NewRegexCompressor("go")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "Start")
	assert.Contains(t, names, "Stop")
	assert.Contains(t, names, "NewServer")
}

// ---------------------------------------------------------------------------
// TypeScript regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_TypeScript(t *testing.T) {
	source := `import { Request, Response } from 'express';
import axios from 'axios';

interface UserConfig {
  name: string;
  email: string;
}

type Status = 'active' | 'inactive' | 'pending';

export const enum Direction {
  Up,
  Down,
}

export class UserService {
  constructor(private db: Database) {}

  async findUser(id: string): Promise<User> {
    return this.db.find(id);
  }
}

export async function handleRequest(req: Request, res: Response): Promise<void> {
  const user = await findUser(req.params.id);
  res.json(user);
}

export function syncHandler(req: Request): Response {
  return new Response();
}

export const API_KEY = 'secret';

export { UserService as default };
`
	c := NewRegexCompressor("typescript")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract imports")
	assert.Contains(t, kinds, KindInterface, "should extract interfaces")
	assert.Contains(t, kinds, KindType, "should extract type aliases")
	assert.Contains(t, kinds, KindClass, "should extract classes")
	assert.Contains(t, kinds, KindFunction, "should extract functions")
	assert.Contains(t, kinds, KindConstant, "should extract constants")
	assert.Contains(t, kinds, KindExport, "should extract re-exports")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "UserConfig")
	assert.Contains(t, names, "Status")
	assert.Contains(t, names, "Direction")
	assert.Contains(t, names, "UserService")
	assert.Contains(t, names, "handleRequest")
	assert.Contains(t, names, "syncHandler")
	assert.Contains(t, names, "API_KEY")

	rendered := output.Render()
	assert.NotContains(t, rendered, "return this.db.find(id)")
	assert.NotContains(t, rendered, "res.json(user)")
}

// ---------------------------------------------------------------------------
// JavaScript regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_JavaScript(t *testing.T) {
	source := `import express from 'express';
import { readFile } from 'fs/promises';

class Router {
  constructor() {
    this.routes = [];
  }

  addRoute(path, handler) {
    this.routes.push({ path, handler });
  }
}

export function createApp() {
  const app = express();
  return app;
}

export async function loadConfig(path) {
  const data = await readFile(path, 'utf-8');
  return JSON.parse(data);
}

export const PORT = 3000;

export { Router };
`
	c := NewRegexCompressor("javascript")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport)
	assert.Contains(t, kinds, KindClass)
	assert.Contains(t, kinds, KindFunction)
	assert.Contains(t, kinds, KindConstant)
	assert.Contains(t, kinds, KindExport)

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "Router")
	assert.Contains(t, names, "createApp")
	assert.Contains(t, names, "loadConfig")
	assert.Contains(t, names, "PORT")
}

// ---------------------------------------------------------------------------
// Python regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_Python(t *testing.T) {
	source := `from fastapi import APIRouter, Depends
import os
from typing import Optional, List

MAX_RETRIES = 5
DEFAULT_TIMEOUT = 30

class UserService:
    def __init__(self, db):
        self.db = db

    def get_user(self, user_id: int):
        return self.db.query(User).get(user_id)

async def create_user(name: str, email: str) -> User:
    user = User(name=name, email=email)
    db.add(user)
    return user

def sync_handler(request):
    return {"status": "ok"}

class Config:
    DEBUG = True
    SECRET_KEY = "secret"
`
	c := NewRegexCompressor("python")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract imports")
	assert.Contains(t, kinds, KindClass, "should extract classes")
	assert.Contains(t, kinds, KindFunction, "should extract functions/defs")
	assert.Contains(t, kinds, KindConstant, "should extract constants")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "MAX_RETRIES")
	assert.Contains(t, names, "DEFAULT_TIMEOUT")
	assert.Contains(t, names, "UserService")
	assert.Contains(t, names, "create_user")
	assert.Contains(t, names, "sync_handler")
	assert.Contains(t, names, "Config")
}

// ---------------------------------------------------------------------------
// Rust regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_Rust(t *testing.T) {
	source := `use std::collections::HashMap;
use std::fmt;

pub struct Config {
    pub host: String,
    pub port: u16,
}

pub enum Status {
    Active,
    Inactive,
}

pub trait Service {
    fn start(&self) -> Result<(), Error>;
    fn stop(&self);
}

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

pub fn create_config(host: &str, port: u16) -> Config {
    Config {
        host: host.to_string(),
        port,
    }
}

pub async fn fetch_data(url: &str) -> Result<String> {
    let resp = reqwest::get(url).await?;
    Ok(resp.text().await?)
}

pub const MAX_CONNECTIONS: usize = 100;
`
	c := NewRegexCompressor("rust")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract use statements")
	assert.Contains(t, kinds, KindStruct, "should extract structs")
	assert.Contains(t, kinds, KindType, "should extract enums and type aliases")
	assert.Contains(t, kinds, KindInterface, "should extract traits")
	assert.Contains(t, kinds, KindFunction, "should extract functions")
	assert.Contains(t, kinds, KindConstant, "should extract constants")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "Config")
	assert.Contains(t, names, "Status")
	assert.Contains(t, names, "Service")
	assert.Contains(t, names, "create_config")
	assert.Contains(t, names, "fetch_data")
	assert.Contains(t, names, "MAX_CONNECTIONS")
}

// ---------------------------------------------------------------------------
// Java regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_Java(t *testing.T) {
	source := `package com.example.api;

import java.util.List;
import java.util.Optional;

public class UserController {
    private final UserService service;

    public UserController(UserService service) {
        this.service = service;
    }

    public List<User> listUsers() {
        return service.findAll();
    }

    public Optional<User> getUser(String id) {
        return service.findById(id);
    }

    private void validateInput(String input) {
        if (input == null) throw new IllegalArgumentException("null input");
    }
}
`
	c := NewRegexCompressor("java")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract package and imports")
	assert.Contains(t, kinds, KindClass, "should extract class declaration")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "UserController")

	// Note: regex compressor only captures top-level (non-indented) declarations.
	// Java methods inside class bodies are indented and intentionally skipped
	// to avoid capturing function body code. The AST compressor handles nested
	// method extraction.
	assert.GreaterOrEqual(t, len(output.Signatures), 2,
		"should extract at least package/import and class")
}

// ---------------------------------------------------------------------------
// C regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_C(t *testing.T) {
	source := `#include <stdio.h>
#include <stdlib.h>

#define MAX_BUFFER_SIZE 1024
#define MIN(a, b) ((a) < (b) ? (a) : (b))

typedef struct {
    char *name;
    int age;
} Person;

struct Node {
    int value;
    struct Node *next;
};

enum Color {
    RED,
    GREEN,
    BLUE
};

int add(int a, int b) {
    return a + b;
}

void *allocate_buffer(size_t size) {
    return malloc(size);
}
`
	c := NewRegexCompressor("c")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract #include")
	assert.Contains(t, kinds, KindConstant, "should extract #define")
	assert.Contains(t, kinds, KindStruct, "should extract structs")
	assert.Contains(t, kinds, KindType, "should extract enums")
	assert.Contains(t, kinds, KindFunction, "should extract functions")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "MAX_BUFFER_SIZE")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "allocate_buffer")
}

// ---------------------------------------------------------------------------
// C++ regex extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_Cpp(t *testing.T) {
	source := `#include <iostream>
#include <vector>
#include <memory>

#define VERSION "1.0.0"

template<typename T>
class Container {
public:
    void add(const T& item) {
        items.push_back(item);
    }

    size_t size() const {
        return items.size();
    }

private:
    std::vector<T> items;
};

namespace utils {

int helper(int x) {
    return x * 2;
}

} // namespace utils

enum class Status {
    Active,
    Inactive,
};

std::string format_name(const std::string& first, const std::string& last) {
    return first + " " + last;
}

void print_all(const std::vector<int>& values) {
    for (auto v : values) {
        std::cout << v << "\n";
    }
}
`
	c := NewRegexCompressor("cpp")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	kinds := signatureKinds(output.Signatures)
	assert.Contains(t, kinds, KindImport, "should extract #include")
	assert.Contains(t, kinds, KindConstant, "should extract #define")
	assert.Contains(t, kinds, KindClass, "should extract template class")
	assert.Contains(t, kinds, KindType, "should extract namespaces and enums")
	assert.Contains(t, kinds, KindFunction, "should extract functions")

	names := signatureNames(output.Signatures)
	assert.Contains(t, names, "Container")
	assert.Contains(t, names, "format_name")
	assert.Contains(t, names, "print_all")
}

// ---------------------------------------------------------------------------
// Multi-line signature extraction
// ---------------------------------------------------------------------------

func TestRegexCompressor_MultiLine(t *testing.T) {
	tests := []struct {
		name     string
		language string
		source   string
		wantSigs int
		wantName string
	}{
		{
			name:     "Go function spanning multiple lines",
			language: "go",
			source: `func ProcessData(
	ctx context.Context,
	input *DataInput,
	options ...Option,
) (*Result, error) {
	return nil, nil
}`,
			wantSigs: 1,
			wantName: "ProcessData",
		},
		{
			name:     "Python function with long params",
			language: "python",
			source: `def create_user(
    name: str,
    email: str,
    role: str = "viewer",
) -> User:
    pass`,
			wantSigs: 1,
			wantName: "create_user",
		},
		{
			name:     "Rust function with generics",
			language: "rust",
			source: `pub fn merge_maps(
    left: HashMap<String, Value>,
    right: HashMap<String, Value>,
) -> HashMap<String, Value> {
    left.into_iter().chain(right).collect()
}`,
			wantSigs: 1,
			wantName: "merge_maps",
		},
		{
			name:     "TypeScript function with long signature",
			language: "typescript",
			source: `export async function processRequest(
    req: Request,
    res: Response,
    next: NextFunction,
): Promise<void> {
    const data = await req.json();
    res.json(data);
}`,
			wantSigs: 1,
			wantName: "processRequest",
		},
		{
			name:     "Java method with annotations",
			language: "java",
			source: `public ResponseEntity<List<UserDTO>> listUsers(
        int page,
        int size,
        String sortBy) {
    return ResponseEntity.ok(service.list(page, size, sortBy));
}`,
			wantSigs: 1,
			wantName: "listUsers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRegexCompressor(tt.language)
			output, err := c.Compress(context.Background(), []byte(tt.source))
			require.NoError(t, err)

			assert.Len(t, output.Signatures, tt.wantSigs,
				"expected %d signatures, got %d", tt.wantSigs, len(output.Signatures))

			if tt.wantSigs > 0 && len(output.Signatures) > 0 {
				assert.Equal(t, tt.wantName, output.Signatures[0].Name,
					"expected name %q, got %q", tt.wantName, output.Signatures[0].Name)
				assert.Equal(t, KindFunction, output.Signatures[0].Kind)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Empty and edge cases
// ---------------------------------------------------------------------------

func TestRegexCompressor_EmptySource(t *testing.T) {
	for _, lang := range []string{"go", "typescript", "javascript", "python", "rust", "java", "c", "cpp"} {
		t.Run(lang, func(t *testing.T) {
			c := NewRegexCompressor(lang)
			output, err := c.Compress(context.Background(), []byte{})
			require.NoError(t, err)
			assert.Empty(t, output.Signatures)
			assert.Equal(t, 0, output.OriginalSize)
			assert.Equal(t, lang, output.Language)
		})
	}
}

func TestRegexCompressor_NoMatchingPatterns(t *testing.T) {
	// Source with no recognizable patterns for Go.
	source := `// just a comment
// another comment
// no actual code here
`
	c := NewRegexCompressor("go")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)
	assert.Empty(t, output.Signatures, "comments-only source should produce no signatures")
}

func TestRegexCompressor_UnknownLanguage(t *testing.T) {
	c := NewRegexCompressor("haskell")
	output, err := c.Compress(context.Background(), []byte("module Main where\nmain = putStrLn \"hello\""))
	require.NoError(t, err)
	assert.Empty(t, output.Signatures, "unknown language should produce no signatures")
	assert.Equal(t, "haskell", output.Language)
}

func TestRegexCompressor_ContextCancellation(t *testing.T) {
	c := NewRegexCompressor("go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Generate enough lines to trigger the cancellation check.
	var b strings.Builder
	for i := 0; i < 2000; i++ {
		b.WriteString("// line\n")
	}

	_, err := c.Compress(ctx, []byte(b.String()))
	assert.Error(t, err, "should return error on cancelled context")
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Name extraction tests
// ---------------------------------------------------------------------------

func TestExtractRegexName(t *testing.T) {
	tests := []struct {
		name     string
		trimmed  string
		kind     SignatureKind
		language string
		want     string
	}{
		// Go functions
		{"go func", "func Hello()", KindFunction, "go", "Hello"},
		{"go method", "func (s *Server) Start(ctx context.Context) error", KindFunction, "go", "Start"},
		{"go func no parens", "func init()", KindFunction, "go", "init"},

		// Python functions
		{"python def", "def process(data):", KindFunction, "python", "process"},
		{"python async def", "async def fetch(url: str):", KindFunction, "python", "fetch"},

		// Rust functions
		{"rust fn", "pub fn create(name: &str) -> Self", KindFunction, "rust", "create"},
		{"rust async fn", "pub async fn fetch(url: &str) -> Result<String>", KindFunction, "rust", "fetch"},

		// Java methods
		{"java method", "public void run()", KindFunction, "java", "run"},
		{"java generic return", "public List<String> getAll()", KindFunction, "java", "getAll"},

		// TypeScript functions
		{"ts function", "export async function handler(req: Request)", KindFunction, "typescript", "handler"},
		{"ts function*", "function* generator()", KindFunction, "typescript", "generator"},

		// C functions
		{"c function", "int main(int argc, char *argv[])", KindFunction, "c", "main"},
		{"c pointer return", "void *malloc(size_t size)", KindFunction, "c", "malloc"},

		// C++ functions
		{"cpp method", "int MyClass::method(int x)", KindFunction, "cpp", "method"},

		// Class names
		{"class simple", "class UserService", KindClass, "typescript", "UserService"},
		{"class export", "export class Router", KindClass, "typescript", "Router"},
		{"class abstract", "export abstract class Base", KindClass, "typescript", "Base"},

		// Struct names
		{"rust struct", "pub struct Config", KindStruct, "rust", "Config"},
		{"rust pub(crate) struct", "pub(crate) struct InternalConfig", KindStruct, "rust", "InternalConfig"},

		// Interface names
		{"ts interface", "interface UserConfig", KindInterface, "typescript", "UserConfig"},
		{"ts export interface", "export interface ApiResponse", KindInterface, "typescript", "ApiResponse"},
		{"rust trait", "pub trait Handler", KindInterface, "rust", "Handler"},

		// Type names
		{"go type", "type ID string", KindType, "go", "ID"},
		{"ts type alias", "type Status = 'active' | 'inactive'", KindType, "typescript", "Status"},
		{"ts enum", "enum Direction", KindType, "typescript", "Direction"},
		{"rust enum", "pub enum Color", KindType, "rust", "Color"},

		// Constant names
		{"go const", "const MaxRetries = 3", KindConstant, "go", "MaxRetries"},
		{"python const", "MAX_RETRIES = 5", KindConstant, "python", "MAX_RETRIES"},
		{"rust const", "pub const MAX_SIZE: usize = 1024", KindConstant, "rust", "MAX_SIZE"},
		{"c define", "#define BUFFER_SIZE 1024", KindConstant, "c", "BUFFER_SIZE"},
		{"ts const", "export const API_KEY = 'key'", KindConstant, "typescript", "API_KEY"},

		// Import/export: should return empty name
		{"import", "import \"fmt\"", KindImport, "go", ""},
		{"export", "export { Router }", KindExport, "typescript", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRegexName(tt.trimmed, tt.kind, tt.language)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// extractWordBefore tests
// ---------------------------------------------------------------------------

func TestExtractWordBefore(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		stopChars string
		want      string
	}{
		{"simple", "Hello(world)", "(", "Hello"},
		{"with space", "Hello world", " ", "Hello"},
		{"empty input", "", "(", ""},
		{"stop at first", "(hello)", "(", ""},
		{"no stop char", "hello", "xyz", "hello"},
		{"strip asterisk", "*ptr", " (", "ptr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWordBefore(tt.input, tt.stopChars)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// CompressEngine parsing
// ---------------------------------------------------------------------------

func TestParseCompressEngine(t *testing.T) {
	tests := []struct {
		input   string
		want    CompressEngine
		wantErr bool
	}{
		{"ast", EngineAST, false},
		{"wasm", EngineAST, false},
		{"regex", EngineRegex, false},
		{"auto", EngineAuto, false},
		{"unknown", "", true},
		{"", "", true},
		{"AST", "", true},
		{"WASM", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCompressEngine(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCompressEngineString(t *testing.T) {
	assert.Equal(t, "ast", EngineAST.String())
	assert.Equal(t, "regex", EngineRegex.String())
	assert.Equal(t, "auto", EngineAuto.String())
}

func TestValidEngines(t *testing.T) {
	assert.Len(t, ValidEngines, 3)
	assert.Contains(t, ValidEngines, EngineAST)
	assert.Contains(t, ValidEngines, EngineRegex)
	assert.Contains(t, ValidEngines, EngineAuto)
}

// ---------------------------------------------------------------------------
// Multi-line signature helper tests
// ---------------------------------------------------------------------------

func TestExtractMultiLineSignature(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		startIdx    int
		wantSig     string
		wantEndLine int
	}{
		{
			name:        "single line with brace",
			lines:       []string{"func Hello() {"},
			startIdx:    0,
			wantSig:     "func Hello()",
			wantEndLine: 1,
		},
		{
			name: "multi-line Go func",
			lines: []string{
				"func Process(",
				"\tctx context.Context,",
				"\tinput string,",
				") error {",
			},
			startIdx:    0,
			wantEndLine: 4,
		},
		{
			name:        "balanced on first line",
			lines:       []string{"func Simple() error {"},
			startIdx:    0,
			wantSig:     "func Simple() error",
			wantEndLine: 1,
		},
		{
			name: "no closing brace",
			lines: []string{
				"func NoBody(",
				"\tx int,",
				"\ty int,",
				")",
			},
			startIdx:    0,
			wantEndLine: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, endLine := extractMultiLineSignature(tt.lines, tt.startIdx)
			assert.Equal(t, tt.wantEndLine, endLine, "end line mismatch")
			if tt.wantSig != "" {
				assert.Equal(t, tt.wantSig, sig, "signature mismatch")
			}
			assert.NotEmpty(t, sig)
		})
	}
}

// ---------------------------------------------------------------------------
// False positive tests: patterns inside strings/comments
// ---------------------------------------------------------------------------

func TestRegexCompressor_FalsePositives(t *testing.T) {
	tests := []struct {
		name     string
		language string
		source   string
		desc     string
	}{
		{
			name:     "Go comment with func keyword",
			language: "go",
			source: `package main

// This comment mentions func but is not a function
// func fakeName() should not be extracted
`,
			desc: "comment lines should be skipped (they start with //)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRegexCompressor(tt.language)
			output, err := c.Compress(context.Background(), []byte(tt.source))
			require.NoError(t, err)

			// Filter out imports (package declarations match as import in regex).
			funcSigs := filterSignatures(output.Signatures, KindFunction)
			assert.Empty(t, funcSigs, "%s: %s", tt.name, tt.desc)
		})
	}
}

// ---------------------------------------------------------------------------
// Signature source order verification
// ---------------------------------------------------------------------------

func TestRegexCompressor_SignatureOrder(t *testing.T) {
	source := `package main

import "fmt"

const Version = "1.0"

func First() {
}

func Second() {
}

func Third() {
}
`
	c := NewRegexCompressor("go")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	// Verify signatures are in source order (StartLine increases).
	for i := 1; i < len(output.Signatures); i++ {
		assert.GreaterOrEqual(t, output.Signatures[i].StartLine, output.Signatures[i-1].StartLine,
			"signatures should be in source order")
	}
}

// ---------------------------------------------------------------------------
// Compression ratio sanity check
// ---------------------------------------------------------------------------

func TestRegexCompressor_CompressionRatio(t *testing.T) {
	source := `package main

import (
	"fmt"
	"net/http"
)

type Server struct {
	addr string
	port int
}

func NewServer(addr string, port int) *Server {
	return &Server{
		addr: addr,
		port: port,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", r.URL.Path)
	})
	return http.ListenAndServe(fmt.Sprintf("%s:%d", s.addr, s.port), nil)
}

func (s *Server) Stop() {
	fmt.Println("stopping server")
}

func main() {
	s := NewServer("localhost", 8080)
	s.Start()
}
`
	c := NewRegexCompressor("go")
	output, err := c.Compress(context.Background(), []byte(source))
	require.NoError(t, err)

	rendered := output.Render()
	output.OutputSize = len(rendered)
	ratio := output.CompressionRatio()
	assert.Greater(t, ratio, 0.0, "ratio should be positive")
	assert.Less(t, ratio, 1.0, "compression should reduce size")
}

// ---------------------------------------------------------------------------
// Orchestrator engine selection tests
// ---------------------------------------------------------------------------

func TestOrchestratorRegexEngine(t *testing.T) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second, // 5s
		Concurrency:    1,
		Engine:         EngineRegex,
	}
	orch := NewOrchestrator(cfg)

	files := []*CompressibleFile{
		{
			Path:    "main.go",
			Content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.True(t, files[0].IsCompressed, "file should be compressed with regex engine")
	assert.Equal(t, "go", files[0].Language)
	assert.True(t, strings.HasPrefix(files[0].Content, CompressedMarker))
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
}

func TestOrchestratorAutoEngine(t *testing.T) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    1,
		Engine:         EngineAuto,
	}
	orch := NewOrchestrator(cfg)

	files := []*CompressibleFile{
		{
			Path:    "handler.ts",
			Content: "export function handler(): void {\n  console.log('hello');\n}\n",
		},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.True(t, files[0].IsCompressed, "auto engine should compress TypeScript")
	assert.Equal(t, "typescript", files[0].Language)
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesCompressed))
}

func TestOrchestratorRegexEngine_UnsupportedLanguage(t *testing.T) {
	cfg := CompressionConfig{
		Enabled:        true,
		TimeoutPerFile: 5 * time.Second,
		Concurrency:    1,
		Engine:         EngineRegex,
	}
	orch := NewOrchestrator(cfg)

	originalContent := "# README\n\nSome text.\n"
	files := []*CompressibleFile{
		{Path: "README.md", Content: originalContent},
	}

	stats, err := orch.Compress(context.Background(), files)
	require.NoError(t, err)

	assert.False(t, files[0].IsCompressed, "markdown should not be compressed")
	assert.Equal(t, originalContent, files[0].Content)
	assert.Equal(t, int64(1), atomic.LoadInt64(&stats.FilesSkipped))
}

func TestOrchestratorDefaultEngine(t *testing.T) {
	cfg := DefaultCompressionConfig()
	assert.Equal(t, EngineAuto, cfg.Engine, "default engine should be auto")
}

// ---------------------------------------------------------------------------
// Per-language pattern registration
// ---------------------------------------------------------------------------

func TestGetRegexPatterns_AllRegistered(t *testing.T) {
	expected := []string{"go", "typescript", "javascript", "python", "rust", "java", "c", "cpp"}
	for _, lang := range expected {
		t.Run(lang, func(t *testing.T) {
			patterns := getRegexPatterns(lang)
			assert.NotNil(t, patterns, "patterns should be registered for %s", lang)
			assert.NotEmpty(t, patterns, "patterns should not be empty for %s", lang)
		})
	}
}

func TestGetRegexPatterns_UnregisteredLanguage(t *testing.T) {
	patterns := getRegexPatterns("cobol")
	assert.Nil(t, patterns, "unregistered language should return nil")
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// signatureKinds extracts the set of unique SignatureKind values from a slice.
func signatureKinds(sigs []Signature) []SignatureKind {
	seen := make(map[SignatureKind]bool)
	var kinds []SignatureKind
	for _, s := range sigs {
		if !seen[s.Kind] {
			seen[s.Kind] = true
			kinds = append(kinds, s.Kind)
		}
	}
	return kinds
}

// signatureNames extracts the set of non-empty names from signatures.
func signatureNames(sigs []Signature) []string {
	var names []string
	for _, s := range sigs {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return names
}

// filterSignatures returns only signatures of the given kind.
func filterSignatures(sigs []Signature, kind SignatureKind) []Signature {
	var filtered []Signature
	for _, s := range sigs {
		if s.Kind == kind {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
