// Package grammars provides embedded tree-sitter grammar WASM files for use
// by the compression subsystem. Grammar files are downloaded from the Sourcegraph
// tree-sitter-wasms npm package and embedded at compile time.
//
// To download or update grammar files, run:
//
//	./scripts/fetch-grammars.sh
//
// The WASM files must be present in this directory before compilation.
package grammars

import "embed"

// FS embeds all tree-sitter grammar WASM files in this directory.
// The embedded filesystem maps filenames like "tree-sitter-go.wasm"
// to their binary content. Use FS.ReadFile to access individual grammars.
//
//go:embed *.wasm
var FS embed.FS

// GrammarFiles maps language names to their embedded WASM filenames.
// This serves as the canonical registry of supported languages for
// tree-sitter compression.
var GrammarFiles = map[string]string{
	"typescript": "tree-sitter-typescript.wasm",
	"javascript": "tree-sitter-javascript.wasm",
	"go":         "tree-sitter-go.wasm",
	"python":     "tree-sitter-python.wasm",
	"rust":       "tree-sitter-rust.wasm",
	"java":       "tree-sitter-java.wasm",
	"c":          "tree-sitter-c.wasm",
	"cpp":        "tree-sitter-cpp.wasm",
}