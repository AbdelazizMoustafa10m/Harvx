package output

import (
	"fmt"
	"path/filepath"
	"strings"
)

// languageFromExt maps file extensions to Markdown code fence language identifiers.
// Returns an empty string for unknown extensions.
func languageFromExt(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	lang, ok := extToLanguage[ext]
	if !ok {
		return ""
	}
	return lang
}

// extToLanguage maps lowercase file extensions to Markdown code fence language identifiers.
var extToLanguage = map[string]string{
	// Go
	".go": "go",
	// TypeScript
	".ts":  "typescript",
	".tsx": "typescript",
	".mts": "typescript",
	".cts": "typescript",
	// JavaScript
	".js":  "javascript",
	".jsx": "javascript",
	".mjs": "javascript",
	".cjs": "javascript",
	// Python
	".py":  "python",
	".pyi": "python",
	// Rust
	".rs": "rust",
	// Java
	".java": "java",
	// C
	".c": "c",
	".h": "c",
	// C++
	".cpp": "cpp",
	".cc":  "cpp",
	".cxx": "cpp",
	".hpp": "cpp",
	".hxx": "cpp",
	// Ruby
	".rb": "ruby",
	// PHP
	".php": "php",
	// Swift
	".swift": "swift",
	// Kotlin
	".kt":  "kotlin",
	".kts": "kotlin",
	// Scala
	".scala": "scala",
	// Shell
	".sh":   "bash",
	".bash": "bash",
	".zsh":  "zsh",
	".fish": "fish",
	// Web
	".html": "html",
	".htm":  "html",
	".css":  "css",
	".scss": "scss",
	".sass": "sass",
	".less": "less",
	// Config/Data
	".json":  "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".toml":  "toml",
	".xml":   "xml",
	".ini":   "ini",
	".cfg":   "ini",
	".conf":  "conf",
	// Markdown
	".md":       "markdown",
	".markdown": "markdown",
	// SQL
	".sql": "sql",
	// Protobuf
	".proto": "protobuf",
	// GraphQL
	".graphql": "graphql",
	".gql":     "graphql",
	// Terraform
	".tf":     "hcl",
	".tfvars": "hcl",
	// Lua
	".lua": "lua",
	// R
	".r": "r",
	".R": "r",
	// Dart
	".dart": "dart",
	// Elixir
	".ex":  "elixir",
	".exs": "elixir",
	// Erlang
	".erl": "erlang",
	// Haskell
	".hs": "haskell",
	// OCaml
	".ml":  "ocaml",
	".mli": "ocaml",
	// Zig
	".zig": "zig",
	// Nim
	".nim": "nim",
}

// formatBytes formats a byte count into a human-readable string.
// Uses binary units: 1 KB = 1024 bytes. Values 1 KB and above use one
// decimal place.
func formatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatNumber formats an integer with comma separators for readability.
// For example, 1234567 becomes "1,234,567".
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas from right to left.
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

// addLineNumbers prefixes each line of content with a right-aligned line number
// and " | " separator. Example output:
//
//	  1 | package main
//	  2 |
//	  3 | func main() {
func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")

	// Calculate width needed for line numbers.
	width := len(fmt.Sprintf("%d", len(lines)))

	var sb strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&sb, "%*d | %s", width, i+1, line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// repeatString repeats a string n times. Used in template formatting.
func repeatString(s string, n int) string {
	return strings.Repeat(s, n)
}

// tierLabel returns the human-readable label for a tier number.
func tierLabel(tier int) string {
	labels := map[int]string{
		0: "critical",
		1: "primary",
		2: "secondary",
		3: "tests",
		4: "docs",
		5: "low",
	}
	if label, ok := labels[tier]; ok {
		return label
	}
	return fmt.Sprintf("tier%d", tier)
}

// escapeTripleBackticks escapes triple backticks within file content to prevent
// breaking Markdown fenced code blocks. Replaces ``` with `` ` (two backticks,
// space, one backtick).
func escapeTripleBackticks(content string) string {
	return strings.ReplaceAll(content, "```", "`` `")
}
