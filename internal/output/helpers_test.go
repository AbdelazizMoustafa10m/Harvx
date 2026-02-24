package output

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// TestLanguageFromExt
// ---------------------------------------------------------------------------

func TestLanguageFromExt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		// Go
		{name: "go", filePath: "main.go", want: "go"},
		// TypeScript variants
		{name: "ts", filePath: "app.ts", want: "typescript"},
		{name: "tsx", filePath: "component.tsx", want: "typescript"},
		{name: "mts", filePath: "module.mts", want: "typescript"},
		{name: "cts", filePath: "common.cts", want: "typescript"},
		// JavaScript variants
		{name: "js", filePath: "index.js", want: "javascript"},
		{name: "jsx", filePath: "app.jsx", want: "javascript"},
		{name: "mjs", filePath: "module.mjs", want: "javascript"},
		{name: "cjs", filePath: "common.cjs", want: "javascript"},
		// Python
		{name: "py", filePath: "script.py", want: "python"},
		{name: "pyi", filePath: "types.pyi", want: "python"},
		// Rust
		{name: "rs", filePath: "lib.rs", want: "rust"},
		// Java
		{name: "java", filePath: "Main.java", want: "java"},
		// C
		{name: "c", filePath: "main.c", want: "c"},
		{name: "h", filePath: "header.h", want: "c"},
		// C++
		{name: "cpp", filePath: "main.cpp", want: "cpp"},
		{name: "cc", filePath: "main.cc", want: "cpp"},
		{name: "cxx", filePath: "main.cxx", want: "cpp"},
		{name: "hpp", filePath: "header.hpp", want: "cpp"},
		{name: "hxx", filePath: "header.hxx", want: "cpp"},
		// Ruby
		{name: "rb", filePath: "app.rb", want: "ruby"},
		// PHP
		{name: "php", filePath: "index.php", want: "php"},
		// Swift
		{name: "swift", filePath: "main.swift", want: "swift"},
		// Kotlin
		{name: "kt", filePath: "Main.kt", want: "kotlin"},
		{name: "kts", filePath: "build.kts", want: "kotlin"},
		// Scala
		{name: "scala", filePath: "App.scala", want: "scala"},
		// Shell
		{name: "sh", filePath: "deploy.sh", want: "bash"},
		{name: "bash", filePath: "init.bash", want: "bash"},
		{name: "zsh", filePath: "config.zsh", want: "zsh"},
		{name: "fish", filePath: "setup.fish", want: "fish"},
		// Web
		{name: "html", filePath: "index.html", want: "html"},
		{name: "htm", filePath: "page.htm", want: "html"},
		{name: "css", filePath: "style.css", want: "css"},
		{name: "scss", filePath: "theme.scss", want: "scss"},
		{name: "sass", filePath: "theme.sass", want: "sass"},
		{name: "less", filePath: "vars.less", want: "less"},
		// Config/Data
		{name: "json", filePath: "config.json", want: "json"},
		{name: "yaml", filePath: "deploy.yaml", want: "yaml"},
		{name: "yml", filePath: "ci.yml", want: "yaml"},
		{name: "toml", filePath: "config.toml", want: "toml"},
		{name: "xml", filePath: "pom.xml", want: "xml"},
		{name: "ini", filePath: "settings.ini", want: "ini"},
		{name: "cfg", filePath: "app.cfg", want: "ini"},
		{name: "conf", filePath: "nginx.conf", want: "conf"},
		// Markdown
		{name: "md", filePath: "README.md", want: "markdown"},
		{name: "markdown", filePath: "CHANGELOG.markdown", want: "markdown"},
		// SQL
		{name: "sql", filePath: "schema.sql", want: "sql"},
		// Protobuf
		{name: "proto", filePath: "api.proto", want: "protobuf"},
		// GraphQL
		{name: "graphql", filePath: "schema.graphql", want: "graphql"},
		{name: "gql", filePath: "query.gql", want: "graphql"},
		// Terraform
		{name: "tf", filePath: "main.tf", want: "hcl"},
		{name: "tfvars", filePath: "prod.tfvars", want: "hcl"},
		// Lua
		{name: "lua", filePath: "init.lua", want: "lua"},
		// R
		{name: "r lowercase", filePath: "analysis.r", want: "r"},
		{name: "R uppercase", filePath: "analysis.R", want: "r"},
		// Dart
		{name: "dart", filePath: "main.dart", want: "dart"},
		// Elixir
		{name: "ex", filePath: "app.ex", want: "elixir"},
		{name: "exs", filePath: "test.exs", want: "elixir"},
		// Erlang
		{name: "erl", filePath: "server.erl", want: "erlang"},
		// Haskell
		{name: "hs", filePath: "Main.hs", want: "haskell"},
		// OCaml
		{name: "ml", filePath: "main.ml", want: "ocaml"},
		{name: "mli", filePath: "types.mli", want: "ocaml"},
		// Zig
		{name: "zig", filePath: "main.zig", want: "zig"},
		// Nim
		{name: "nim", filePath: "app.nim", want: "nim"},
		// Unknown extensions
		{name: "unknown extension", filePath: "data.xyz", want: ""},
		{name: "no extension", filePath: "Makefile", want: ""},
		{name: "dot only", filePath: ".gitignore", want: ""},
		// Case insensitivity
		{name: "uppercase GO", filePath: "main.GO", want: "go"},
		{name: "mixed case Py", filePath: "script.Py", want: "python"},
		// Path with directories
		{name: "nested path", filePath: "internal/cli/root.go", want: "go"},
		{name: "deep nested path", filePath: "a/b/c/d/file.ts", want: "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := languageFromExt(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestFormatBytes
// ---------------------------------------------------------------------------

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "zero bytes", bytes: 0, want: "0 B"},
		{name: "one byte", bytes: 1, want: "1 B"},
		{name: "small bytes", bytes: 512, want: "512 B"},
		{name: "just under 1 KB", bytes: 1023, want: "1023 B"},
		{name: "exactly 1 KB", bytes: 1024, want: "1.0 KB"},
		{name: "1.5 KB", bytes: 1536, want: "1.5 KB"},
		{name: "just under 1 MB", bytes: 1024*1024 - 1, want: "1024.0 KB"},
		{name: "exactly 1 MB", bytes: 1024 * 1024, want: "1.0 MB"},
		{name: "2.5 MB", bytes: 2621440, want: "2.5 MB"},
		{name: "just under 1 GB", bytes: 1024*1024*1024 - 1, want: "1024.0 MB"},
		{name: "exactly 1 GB", bytes: 1024 * 1024 * 1024, want: "1.0 GB"},
		{name: "large GB", bytes: 5 * 1024 * 1024 * 1024, want: "5.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatBytes(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestFormatNumber
// ---------------------------------------------------------------------------

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: "0"},
		{name: "single digit", n: 5, want: "5"},
		{name: "two digits", n: 42, want: "42"},
		{name: "three digits", n: 999, want: "999"},
		{name: "four digits", n: 1000, want: "1,000"},
		{name: "thousands", n: 12345, want: "12,345"},
		{name: "millions", n: 1234567, want: "1,234,567"},
		{name: "billions", n: 1234567890, want: "1,234,567,890"},
		{name: "negative single", n: -5, want: "-5"},
		{name: "negative thousands", n: -12345, want: "-12,345"},
		{name: "negative millions", n: -1234567, want: "-1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatNumber(tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAddLineNumbers
// ---------------------------------------------------------------------------

func TestAddLineNumbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single line",
			content: "package main",
			want:    "1 | package main",
		},
		{
			name:    "multiple lines",
			content: "package main\n\nfunc main() {",
			want:    "1 | package main\n2 | \n3 | func main() {",
		},
		{
			name:    "double digit line numbers",
			content: strings.Join(makeLines(12), "\n"),
			want: " 1 | line 1\n 2 | line 2\n 3 | line 3\n 4 | line 4\n" +
				" 5 | line 5\n 6 | line 6\n 7 | line 7\n 8 | line 8\n" +
				" 9 | line 9\n10 | line 10\n11 | line 11\n12 | line 12",
		},
		{
			name:    "empty content",
			content: "",
			want:    "1 | ",
		},
		{
			name:    "trailing newline",
			content: "line1\nline2\n",
			want:    "1 | line1\n2 | line2\n3 | ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := addLineNumbers(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// makeLines generates n lines with content "line 1", "line 2", etc.
func makeLines(n int) []string {
	lines := make([]string, n)
	for i := range n {
		lines[i] = "line " + strconv.Itoa(i+1)
	}
	return lines
}

// ---------------------------------------------------------------------------
// TestRepeatString
// ---------------------------------------------------------------------------

func TestRepeatString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{name: "repeat dash 3 times", s: "-", n: 3, want: "---"},
		{name: "repeat hash 5 times", s: "#", n: 5, want: "#####"},
		{name: "repeat zero times", s: "x", n: 0, want: ""},
		{name: "repeat empty string", s: "", n: 5, want: ""},
		{name: "repeat multi-char", s: "ab", n: 3, want: "ababab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := repeatString(tt.s, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestTierLabel
// ---------------------------------------------------------------------------

func TestTierLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tier int
		want string
	}{
		{name: "tier 0 critical", tier: 0, want: "critical"},
		{name: "tier 1 primary", tier: 1, want: "primary"},
		{name: "tier 2 secondary", tier: 2, want: "secondary"},
		{name: "tier 3 tests", tier: 3, want: "tests"},
		{name: "tier 4 docs", tier: 4, want: "docs"},
		{name: "tier 5 low", tier: 5, want: "low"},
		{name: "unknown tier 6", tier: 6, want: "tier6"},
		{name: "unknown tier 99", tier: 99, want: "tier99"},
		{name: "negative tier", tier: -1, want: "tier-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tierLabel(tt.tier)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestEscapeTripleBackticks
// ---------------------------------------------------------------------------

func TestEscapeTripleBackticks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "no backticks",
			content: "func main() {}",
			want:    "func main() {}",
		},
		{
			name:    "single backtick unchanged",
			content: "use `code` here",
			want:    "use `code` here",
		},
		{
			name:    "double backtick unchanged",
			content: "use ``code`` here",
			want:    "use ``code`` here",
		},
		{
			name:    "triple backtick escaped",
			content: "```go\nfunc main() {}\n```",
			want:    "`` `go\nfunc main() {}\n`` `",
		},
		{
			name:    "multiple triple backticks",
			content: "start\n```\nmiddle\n```\nend",
			want:    "start\n`` `\nmiddle\n`` `\nend",
		},
		{
			name:    "quadruple backticks partial escape",
			content: "````",
			want:    "`` ``",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := escapeTripleBackticks(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}
