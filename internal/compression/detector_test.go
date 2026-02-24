package compression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectLanguage_AllExtensions(t *testing.T) {
	d := NewLanguageDetector()

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		// TypeScript extensions
		{name: ".ts", filePath: "src/index.ts", want: "typescript"},
		{name: ".tsx", filePath: "src/App.tsx", want: "typescript"},
		{name: ".mts", filePath: "src/utils.mts", want: "typescript"},
		{name: ".cts", filePath: "src/config.cts", want: "typescript"},

		// JavaScript extensions
		{name: ".js", filePath: "src/main.js", want: "javascript"},
		{name: ".jsx", filePath: "src/Component.jsx", want: "javascript"},
		{name: ".mjs", filePath: "lib/helpers.mjs", want: "javascript"},
		{name: ".cjs", filePath: "lib/require.cjs", want: "javascript"},

		// Go
		{name: ".go", filePath: "src/main.go", want: "go"},

		// Python
		{name: ".py", filePath: "app.py", want: "python"},
		{name: ".pyi", filePath: "stubs/types.pyi", want: "python"},

		// Rust
		{name: ".rs", filePath: "src/lib.rs", want: "rust"},

		// Java
		{name: ".java", filePath: "src/Main.java", want: "java"},

		// C
		{name: ".c", filePath: "src/main.c", want: "c"},

		// C++
		{name: ".cpp", filePath: "src/engine.cpp", want: "cpp"},
		{name: ".cc", filePath: "src/parser.cc", want: "cpp"},
		{name: ".cxx", filePath: "src/driver.cxx", want: "cpp"},
		{name: ".hpp", filePath: "include/header.hpp", want: "cpp"},
		{name: ".hxx", filePath: "include/types.hxx", want: "cpp"},

		// Ambiguous .h defaults to C
		{name: ".h", filePath: "include/api.h", want: "c"},

		// Data formats
		{name: ".json", filePath: "package.json", want: "json"},
		{name: ".yaml", filePath: "config.yaml", want: "yaml"},
		{name: ".yml", filePath: "docker-compose.yml", want: "yaml"},
		{name: ".toml", filePath: "Cargo.toml", want: "toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.DetectLanguage(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectLanguage_UnknownExtensions(t *testing.T) {
	d := NewLanguageDetector()

	tests := []struct {
		name     string
		filePath string
	}{
		{name: "xyz extension", filePath: "data/file.xyz"},
		{name: "markdown", filePath: "README.md"},
		{name: "plain text", filePath: "notes.txt"},
		{name: "html", filePath: "index.html"},
		{name: "css", filePath: "styles.css"},
		{name: "ruby", filePath: "app.rb"},
		{name: "php", filePath: "index.php"},
		{name: "no extension", filePath: "Makefile"},
		{name: "dotfile no extension", filePath: ".gitignore"},
		{name: "empty string", filePath: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.DetectLanguage(tt.filePath)
			assert.Equal(t, "", got, "unknown extension should return empty string")
		})
	}
}

func TestDetectLanguage_CaseSensitivity(t *testing.T) {
	d := NewLanguageDetector()

	tests := []struct {
		name     string
		filePath string
	}{
		{name: "uppercase .Go", filePath: "main.Go"},
		{name: "uppercase .PY", filePath: "app.PY"},
		{name: "uppercase .TS", filePath: "index.TS"},
		{name: "uppercase .JS", filePath: "app.JS"},
		{name: "uppercase .RS", filePath: "lib.RS"},
		{name: "uppercase .JAVA", filePath: "Main.JAVA"},
		{name: "uppercase .CPP", filePath: "engine.CPP"},
		{name: "uppercase .JSON", filePath: "config.JSON"},
		{name: "mixed case .Go", filePath: "main.Go"},
		{name: "mixed case .Py", filePath: "app.Py"},
		{name: "mixed case .Ts", filePath: "index.Ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.DetectLanguage(tt.filePath)
			assert.Equal(t, "", got, "uppercase extension %q should return empty string", tt.filePath)
		})
	}
}

func TestDetectLanguage_AmbiguousH(t *testing.T) {
	d := NewLanguageDetector()

	// .h is ambiguous between C and C++ but defaults to "c".
	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{name: "plain header", filePath: "include/types.h", want: "c"},
		{name: "nested header", filePath: "src/internal/api.h", want: "c"},
		{name: "root header", filePath: "config.h", want: "c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.DetectLanguage(tt.filePath)
			assert.Equal(t, tt.want, got, ".h files should default to C")
		})
	}
}

func TestDetectLanguage_NestedPaths(t *testing.T) {
	d := NewLanguageDetector()

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{name: "deeply nested go", filePath: "a/b/c/d/e/main.go", want: "go"},
		{name: "relative path ts", filePath: "./relative/path/index.ts", want: "typescript"},
		{name: "absolute path py", filePath: "/usr/local/src/app.py", want: "python"},
		{name: "dots in directory", filePath: "my.project/src/lib.rs", want: "rust"},
		{name: "spaces in path", filePath: "my project/src/Main.java", want: "java"},
		{name: "unicode in path", filePath: "projet/acces/fichier.cpp", want: "cpp"},
		{name: "multiple dots in name", filePath: "src/my.config.test.json", want: "json"},
		{name: "hidden directory", filePath: ".config/settings.yaml", want: "yaml"},
		{name: "dot dot path", filePath: "../parent/config.toml", want: "toml"},
		{name: "trailing slash removed", filePath: "src/util.js", want: "javascript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.DetectLanguage(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSupportedExtensions_ReturnsCopy(t *testing.T) {
	d := NewLanguageDetector()

	ext1 := d.SupportedExtensions()
	ext2 := d.SupportedExtensions()

	// Both should have identical content.
	assert.Equal(t, ext1, ext2, "two calls should return equivalent maps")

	// Mutate the first copy by adding a new key.
	ext1[".fake"] = "fakelang"

	// The second copy and the detector itself should be unaffected.
	ext3 := d.SupportedExtensions()
	_, hasFake := ext3[".fake"]
	assert.False(t, hasFake, "modifying returned map should not affect the detector")

	// Mutate the first copy by deleting an existing key.
	delete(ext1, ".go")
	ext4 := d.SupportedExtensions()
	_, hasGo := ext4[".go"]
	assert.True(t, hasGo, "deleting from returned map should not affect the detector")
}

func TestSupportedExtensions_Count(t *testing.T) {
	d := NewLanguageDetector()

	ext := d.SupportedExtensions()

	// The built-in map has 24 extensions:
	//   TypeScript: .ts .tsx .mts .cts          (4)
	//   JavaScript: .js .jsx .mjs .cjs          (4)
	//   Go:         .go                         (1)
	//   Python:     .py .pyi                    (2)
	//   Rust:       .rs                         (1)
	//   Java:       .java                       (1)
	//   C:          .c                          (1)
	//   C++:        .cpp .cc .cxx .hpp .hxx     (5)
	//   C (header): .h                          (1)
	//   JSON:       .json                       (1)
	//   YAML:       .yaml .yml                  (2)
	//   TOML:       .toml                       (1)
	//   Total:                                  24
	assert.Len(t, ext, 24, "should have exactly 24 registered extensions")

	// Verify each expected extension is present.
	expectedExts := []string{
		".ts", ".tsx", ".mts", ".cts",
		".js", ".jsx", ".mjs", ".cjs",
		".go",
		".py", ".pyi",
		".rs",
		".java",
		".c",
		".cpp", ".cc", ".cxx", ".hpp", ".hxx",
		".h",
		".json",
		".yaml", ".yml",
		".toml",
	}

	for _, e := range expectedExts {
		_, ok := ext[e]
		assert.True(t, ok, "extension %q should be present in supported extensions", e)
	}
}
