package workflows

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NeighborResult holds the discovered neighborhood for a set of changed files.
type NeighborResult struct {
	// TestFiles are test files related to the changed modules.
	TestFiles []string
	// ImporterFiles are files that import the changed files.
	ImporterFiles []string
	// AllNeighbors is the combined, deduplicated, sorted list of all neighbor files.
	AllNeighbors []string
}

// NeighborOptions configures neighborhood discovery.
type NeighborOptions struct {
	// RootDir is the repository root directory.
	RootDir string
	// ChangedFiles is the list of changed file paths (relative to RootDir).
	ChangedFiles []string
	// Depth controls neighbor discovery depth. 0 = no neighbors, 1 = direct only.
	Depth int
	// AllFiles is the full list of files in the repository (relative paths).
	// Used for searching importers and test files.
	AllFiles []string
}

// DiscoverNeighbors finds related files (tests, importers) for a set of
// changed files. If Depth is 0, returns an empty result. Otherwise it finds
// test files for changed modules and files that import the changed files,
// combining them into a deduplicated, sorted list that excludes files already
// in the changed set.
func DiscoverNeighbors(opts NeighborOptions) *NeighborResult {
	if opts.Depth == 0 {
		slog.Debug("neighbor discovery skipped", "depth", 0)
		return &NeighborResult{}
	}

	changedSet := make(map[string]bool, len(opts.ChangedFiles))
	for _, f := range opts.ChangedFiles {
		changedSet[f] = true
	}

	allFileSet := make(map[string]bool, len(opts.AllFiles))
	for _, f := range opts.AllFiles {
		allFileSet[f] = true
	}

	slog.Debug("discovering neighbors",
		"changed_files", len(opts.ChangedFiles),
		"all_files", len(opts.AllFiles),
		"depth", opts.Depth,
	)

	// 1. Find test files for changed modules.
	testFiles := findRelatedTests(opts.RootDir, opts.ChangedFiles, allFileSet)

	// 2. Find files that import changed files.
	importerFiles := findImporters(opts.RootDir, opts.ChangedFiles, opts.AllFiles)

	// 3. Combine into AllNeighbors, excluding files already in ChangedFiles.
	seen := make(map[string]bool)
	var allNeighbors []string

	for _, f := range testFiles {
		if !changedSet[f] && !seen[f] {
			seen[f] = true
			allNeighbors = append(allNeighbors, f)
		}
	}
	for _, f := range importerFiles {
		if !changedSet[f] && !seen[f] {
			seen[f] = true
			allNeighbors = append(allNeighbors, f)
		}
	}

	sort.Strings(allNeighbors)

	// Also filter test and importer results to exclude changed files.
	testFiles = filterExcluding(testFiles, changedSet)
	importerFiles = filterExcluding(importerFiles, changedSet)

	slog.Debug("neighbor discovery complete",
		"test_files", len(testFiles),
		"importer_files", len(importerFiles),
		"all_neighbors", len(allNeighbors),
	)

	return &NeighborResult{
		TestFiles:     testFiles,
		ImporterFiles: importerFiles,
		AllNeighbors:  allNeighbors,
	}
}

// findRelatedTests discovers test files related to the changed files by
// generating candidate test file paths and checking which ones exist in the
// allFiles set.
func findRelatedTests(rootDir string, changedFiles []string, allFileSet map[string]bool) []string {
	seen := make(map[string]bool)
	var results []string

	for _, changed := range changedFiles {
		candidates := findTestCandidates(changed)
		for _, candidate := range candidates {
			if seen[candidate] {
				continue
			}
			if fileExistsInSet(candidate, allFileSet) {
				seen[candidate] = true
				results = append(results, candidate)
				slog.Debug("found related test",
					"changed", changed,
					"test", candidate,
				)
			}
		}
	}

	sort.Strings(results)
	return results
}

// findImporters discovers files that import one of the changed files. It uses
// a heuristic to limit which files are read: only files with parseable
// extensions (.go, .ts, .tsx, .js, .jsx, .py) are scanned. It builds a set
// of target basenames from the changed files and uses ParseImports from
// imports.go to extract import paths, then checks if any resolved import
// matches a changed file's directory or path.
func findImporters(rootDir string, changedFiles, allFiles []string) []string {
	// Build a set of target directories for matching resolved imports.
	targetDirs := make(map[string]bool)
	changedSet := make(map[string]bool, len(changedFiles))

	for _, cf := range changedFiles {
		changedSet[cf] = true
		dir := filepath.Dir(cf)
		targetDirs[dir] = true
	}

	seen := make(map[string]bool)
	var results []string

	for _, file := range allFiles {
		// Skip changed files themselves.
		if changedSet[file] {
			continue
		}

		// Only read files with parseable extensions.
		if !hasParseableExtension(file) {
			continue
		}

		absPath := filepath.Join(rootDir, file)
		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		imports := ParseImports(file, string(content))
		importerDir := filepath.Dir(file)

		for _, imp := range imports {
			resolved := resolveImportToPath(imp, importerDir, rootDir)
			if resolved == "" {
				continue
			}

			// Check if the resolved import matches any changed file or its directory.
			if changedSet[resolved] || targetDirs[resolved] {
				if !seen[file] {
					seen[file] = true
					results = append(results, file)
					slog.Debug("found importer",
						"file", file,
						"imports", imp,
						"resolved", resolved,
					)
				}
				break
			}

			// Also check if the resolved import's directory matches a changed file's directory.
			resolvedDir := filepath.Dir(resolved)
			if targetDirs[resolvedDir] {
				if !seen[file] {
					seen[file] = true
					results = append(results, file)
					slog.Debug("found importer via directory match",
						"file", file,
						"imports", imp,
						"resolved_dir", resolvedDir,
					)
				}
				break
			}
		}
	}

	sort.Strings(results)
	return results
}

// findTestCandidates generates candidate test file paths for a given file
// path. The candidates are language-specific patterns based on file extension.
// The returned paths are not checked for existence.
func findTestCandidates(filePath string) []string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var candidates []string

	switch ext {
	case ".go":
		// Go: foo.go -> foo_test.go
		candidates = append(candidates, filepath.Join(dir, name+"_test.go"))

	case ".ts", ".tsx":
		// TypeScript: foo.ts -> foo.test.ts, foo.spec.ts, __tests__/foo.ts
		candidates = append(candidates,
			filepath.Join(dir, name+".test"+ext),
			filepath.Join(dir, name+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
		)

	case ".js", ".jsx":
		// JavaScript: foo.js -> foo.test.js, foo.spec.js, __tests__/foo.js
		candidates = append(candidates,
			filepath.Join(dir, name+".test"+ext),
			filepath.Join(dir, name+".spec"+ext),
			filepath.Join(dir, "__tests__", base),
		)

	case ".py":
		// Python: foo.py -> test_foo.py, foo_test.py, tests/test_foo.py
		candidates = append(candidates,
			filepath.Join(dir, "test_"+base),
			filepath.Join(dir, name+"_test.py"),
			filepath.Join(dir, "tests", "test_"+base),
		)

	default:
		// Generic: check for *_test.* or *.test.* or *.spec.* in same directory.
		candidates = append(candidates,
			filepath.Join(dir, name+"_test"+ext),
			filepath.Join(dir, name+".test"+ext),
			filepath.Join(dir, name+".spec"+ext),
		)
	}

	// Normalize paths (clean up "./" prefixes).
	for i, c := range candidates {
		candidates[i] = filepath.Clean(c)
	}

	return candidates
}

// resolveImportToPath resolves a parsed import path to a relative file path
// in the repository. For relative imports (starting with "./" or "../"), the
// path is resolved relative to the importer's directory. For Go module imports
// containing "/internal/", "/cmd/", or "/pkg/", the module prefix is stripped
// to produce a relative path. Returns empty string if the import cannot be
// resolved.
func resolveImportToPath(importPath, importerDir, rootDir string) string {
	if importPath == "" {
		return ""
	}

	// Handle relative imports (./foo, ../bar).
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		resolved := filepath.Join(importerDir, importPath)
		return filepath.Clean(resolved)
	}

	// Handle Go module imports containing /internal/, /cmd/, or /pkg/.
	for _, marker := range []string{"/internal/", "/cmd/", "/pkg/"} {
		idx := strings.Index(importPath, marker)
		if idx >= 0 {
			// Strip the module prefix: everything from the marker onward is the relative path.
			relPath := importPath[idx+1:]
			return filepath.Clean(relPath)
		}
	}

	// For Go imports starting with a well-known prefix.
	for _, prefix := range []string{"internal/", "cmd/", "pkg/"} {
		if strings.HasPrefix(importPath, prefix) {
			return filepath.Clean(importPath)
		}
	}

	return ""
}

// fileExistsInSet reports whether the given path exists in the file set.
func fileExistsInSet(path string, fileSet map[string]bool) bool {
	return fileSet[path]
}

// hasParseableExtension reports whether the file has an extension that
// supports import parsing.
func hasParseableExtension(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py":
		return true
	default:
		return false
	}
}

// filterExcluding returns a new sorted slice with entries in excludeSet
// removed.
func filterExcluding(items []string, excludeSet map[string]bool) []string {
	var filtered []string
	for _, item := range items {
		if !excludeSet[item] {
			filtered = append(filtered, item)
		}
	}
	sort.Strings(filtered)
	return filtered
}
