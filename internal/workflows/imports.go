package workflows

import (
	"bufio"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Package-level compiled regexes for import parsing.
var (
	// goImportRe matches both single-line `import "path"` and lines inside
	// an import block: `"path"`, `alias "path"`, `. "path"`, `_ "path"`.
	goImportRe = regexp.MustCompile(`^\s*(?:import\s+)?(?:\w+\s+|[._]\s+)?"([^"]+)"`)

	// jsImportFromRe matches `import ... from '...'` and `import ... from "..."`.
	jsImportFromRe = regexp.MustCompile(`import\s+.*?\s+from\s+['"]([^'"]+)['"]`)

	// jsRequireRe matches `require('...')` and `require("...")`.
	jsRequireRe = regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	// pyImportRe matches `from X import ...` and `import X`.
	pyImportRe = regexp.MustCompile(`^\s*(?:from\s+(\S+)\s+import|import\s+(\S+))`)

	// goRelativeMarkers are path segments that indicate a project-relative Go
	// import (as opposed to a stdlib or third-party import).
	goRelativeMarkers = []string{"/internal/", "/cmd/", "/pkg/"}
)

// ParseImports parses import statements from a file's content and returns a
// list of resolved relative file paths. The file extension is used to detect
// the language and select the appropriate parser. Returns sorted, deduplicated
// relative paths. Absolute path resolution is the caller's responsibility.
func ParseImports(filePath, content string) []string {
	ext := strings.ToLower(filepath.Ext(filePath))

	var paths []string
	switch ext {
	case ".go":
		paths = parseGoImports(content)
	case ".ts", ".tsx", ".js", ".jsx":
		paths = parseJSImports(content)
	case ".py":
		paths = parsePythonImports(content)
	default:
		// Other languages are handled by neighbors.go via same-directory heuristic.
		return nil
	}

	return deduplicateAndSort(paths)
}

// parseGoImports extracts import paths from Go source code. It handles both
// single-line imports and multi-line import blocks. Only paths that look like
// project-relative module paths are returned (containing /internal/, /cmd/,
// /pkg/, or starting with ./ or ../).
func parseGoImports(content string) []string {
	var paths []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		matches := goImportRe.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		importPath := matches[1]

		if isRelativeImport(importPath) {
			paths = append(paths, importPath)
			continue
		}

		// Check for project-relative markers and strip the module prefix.
		for _, marker := range goRelativeMarkers {
			idx := strings.Index(importPath, marker)
			if idx < 0 {
				continue
			}
			// Strip the module prefix: take everything from the marker segment onward.
			// e.g., "github.com/harvx/harvx/internal/config" -> "internal/config"
			relPath := importPath[idx+1:] // +1 to skip the leading /
			paths = append(paths, relPath)
			break
		}
	}

	return paths
}

// parseJSImports extracts relative import paths from JavaScript/TypeScript
// source code. It recognizes both ES module `import ... from '...'` syntax
// and CommonJS `require('...')` calls. Only relative imports (starting with
// ./ or ../) are returned, with the leading ./ stripped and no extension added.
func parseJSImports(content string) []string {
	var paths []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// Check ES module import.
		if matches := jsImportFromRe.FindStringSubmatch(line); len(matches) >= 2 {
			importPath := matches[1]
			if isRelativeImport(importPath) {
				paths = append(paths, normalizeJSImport(importPath))
			}
		}

		// Check CommonJS require.
		if matches := jsRequireRe.FindStringSubmatch(line); len(matches) >= 2 {
			importPath := matches[1]
			if isRelativeImport(importPath) {
				paths = append(paths, normalizeJSImport(importPath))
			}
		}
	}

	return paths
}

// parsePythonImports extracts import paths from Python source code. It handles
// both `from X import Y` and `import X` statements. Only relative imports
// (starting with .) or simple module names are processed. Dotted module paths
// are converted to file paths (e.g., foo.bar -> foo/bar).
func parsePythonImports(content string) []string {
	var paths []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		matches := pyImportRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		// matches[1] is from `from X import`, matches[2] is from `import X`.
		modulePath := matches[1]
		if modulePath == "" {
			modulePath = matches[2]
		}

		if modulePath == "" {
			continue
		}

		// Process relative imports (starting with .).
		if strings.HasPrefix(modulePath, ".") {
			relPath := convertPythonDottedPath(modulePath)
			if relPath != "" {
				paths = append(paths, relPath)
			}
			continue
		}

		// Simple module names: convert dots to path separators.
		converted := strings.ReplaceAll(modulePath, ".", "/")
		paths = append(paths, converted)
	}

	return paths
}

// normalizeJSImport returns the import path preserving the "./" prefix, which
// is needed by the import resolver to correctly resolve relative paths against
// the importing file's directory.
func normalizeJSImport(importPath string) string {
	return importPath
}

// convertPythonDottedPath converts a Python relative import path to a file
// system path. Leading dots indicate parent directory traversal:
//
//	".foo.bar"  -> "foo/bar"
//	"..foo.bar" -> "../foo/bar"
//	"...foo"    -> "../../foo"
func convertPythonDottedPath(modulePath string) string {
	// Count leading dots.
	dots := 0
	for _, ch := range modulePath {
		if ch != '.' {
			break
		}
		dots++
	}

	remainder := modulePath[dots:]
	if remainder == "" {
		return ""
	}

	// Convert the remainder's dots to path separators.
	converted := strings.ReplaceAll(remainder, ".", "/")

	// Leading dots beyond the first indicate parent directory traversal.
	if dots <= 1 {
		return converted
	}

	// Each extra dot adds a "../" prefix.
	prefix := strings.Repeat("../", dots-1)
	return prefix + converted
}

// deduplicateAndSort removes duplicate entries from a string slice and returns
// the result in sorted order.
func deduplicateAndSort(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))

	for _, p := range paths {
		if _, exists := seen[p]; exists {
			continue
		}
		seen[p] = struct{}{}
		result = append(result, p)
	}

	sort.Strings(result)
	return result
}

// isRelativeImport reports whether the import path starts with "./" or "../",
// indicating a filesystem-relative import.
func isRelativeImport(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}