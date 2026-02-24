package compression

import "path/filepath"

// LanguageDetector maps file extensions to language identifiers.
type LanguageDetector struct {
	extMap map[string]string
}

// NewLanguageDetector creates a LanguageDetector with all built-in extension mappings.
func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{
		extMap: map[string]string{
			// Tier 1: Primary languages
			".ts":  "typescript",
			".tsx": "typescript",
			".mts": "typescript",
			".cts": "typescript",
			".js":  "javascript",
			".jsx": "javascript",
			".mjs": "javascript",
			".cjs": "javascript",
			".go":  "go",
			".py":  "python",
			".pyi": "python",
			".rs":  "rust",

			// Tier 2: Secondary languages
			".java": "java",
			".c":    "c",
			".cpp":  "cpp",
			".cc":   "cpp",
			".cxx":  "cpp",
			".hpp":  "cpp",
			".hxx":  "cpp",
			".h":    "c", // Default to C for ambiguous .h

			// Tier 2: Data formats
			".json": "json",
			".yaml": "yaml",
			".yml":  "yaml",
			".toml": "toml",
		},
	}
}

// DetectLanguage returns the language identifier for a file path.
// Returns empty string if the language is not recognized.
// Extension matching is case-sensitive (Go convention: lowercase extensions).
func (d *LanguageDetector) DetectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	return d.extMap[ext]
}

// SupportedExtensions returns a copy of all registered extension mappings.
func (d *LanguageDetector) SupportedExtensions() map[string]string {
	result := make(map[string]string, len(d.extMap))
	for k, v := range d.extMap {
		result[k] = v
	}
	return result
}