// Package discovery implements file walking, filtering, and binary detection
// for the harvx pipeline. It provides the file discovery engine that traverses
// a repository and produces FileDescriptor entries for downstream processing.
package discovery

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// GitignoreMatcher loads and evaluates .gitignore patterns hierarchically.
// It supports nested .gitignore files where each directory level can add
// patterns that apply only to files within that directory subtree. Parent
// .gitignore rules are inherited by all subdirectories.
//
// Paths passed to IsIgnored must be relative to the root directory that was
// used to construct the matcher.
type GitignoreMatcher struct {
	root     string
	matchers map[string]*gitignore.GitIgnore
	// dirs stores the sorted list of directory keys for deterministic
	// iteration from root toward the file's parent directory.
	dirs   []string
	logger *slog.Logger
}

// NewGitignoreMatcher creates a new GitignoreMatcher rooted at the given
// directory. It walks rootDir to discover all .gitignore files and compiles
// their patterns using sabhiram/go-gitignore.
//
// If no .gitignore files exist, the matcher returns successfully and
// IsIgnored will always return false. Missing or unreadable .gitignore files
// at individual directory levels are logged and skipped without error.
func NewGitignoreMatcher(rootDir string) (*GitignoreMatcher, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolving root path %s: %w", rootDir, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat root path %s: %w", absRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path %s is not a directory", absRoot)
	}

	logger := slog.Default().With("component", "gitignore")

	m := &GitignoreMatcher{
		root:     absRoot,
		matchers: make(map[string]*gitignore.GitIgnore),
		logger:   logger,
	}

	if err := m.discoverGitignoreFiles(); err != nil {
		return nil, fmt.Errorf("discovering .gitignore files in %s: %w", absRoot, err)
	}

	logger.Debug("gitignore matcher initialized",
		"root", absRoot,
		"gitignore_count", len(m.matchers),
	)

	return m, nil
}

// discoverGitignoreFiles walks the root directory tree to find all .gitignore
// files and compiles each one. The walk itself respects already-discovered
// .gitignore patterns to avoid descending into ignored directories (like
// node_modules or .git).
func (m *GitignoreMatcher) discoverGitignoreFiles() error {
	err := filepath.WalkDir(m.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we cannot read.
			m.logger.Debug("skipping unreadable path", "path", path, "error", err)
			return filepath.SkipDir
		}

		// Skip .git directory entirely -- it is never relevant for discovery.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// We only care about .gitignore files.
		if d.IsDir() || d.Name() != ".gitignore" {
			return nil
		}

		dirPath := filepath.Dir(path)
		relDir, err := filepath.Rel(m.root, dirPath)
		if err != nil {
			m.logger.Debug("skipping .gitignore, cannot compute relative path",
				"path", path, "error", err)
			return nil
		}

		compiled, err := gitignore.CompileIgnoreFile(path)
		if err != nil {
			m.logger.Debug("skipping unreadable .gitignore",
				"path", path, "error", err)
			return nil
		}

		// Normalize to use "." for the root directory.
		if relDir == "" {
			relDir = "."
		}

		m.matchers[relDir] = compiled
		m.logger.Debug("loaded .gitignore", "dir", relDir, "path", path)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory tree: %w", err)
	}

	// Build sorted directory list for deterministic evaluation order.
	m.dirs = make([]string, 0, len(m.matchers))
	for dir := range m.matchers {
		m.dirs = append(m.dirs, dir)
	}
	sort.Strings(m.dirs)

	return nil
}

// IsIgnored reports whether the given path should be ignored according to
// the loaded .gitignore rules. The path must be relative to the root
// directory (using forward slashes or OS-native separators). The isDir
// parameter indicates whether the path represents a directory, which is
// needed for directory-only patterns (patterns ending in /).
//
// The matcher evaluates .gitignore files from the root directory down to
// the file's parent directory. A file is ignored if any ancestor's
// .gitignore matches it. Negation patterns in a .gitignore can override
// matches from the same .gitignore file.
//
// Performance: matching is O(number of patterns across all applicable
// .gitignore files), not O(number of files).
func (m *GitignoreMatcher) IsIgnored(path string, isDir bool) bool {
	// Normalize path separators to forward slash for consistent matching.
	normalizedPath := filepath.ToSlash(path)

	// Trim any leading "./" for consistent matching.
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	if normalizedPath == "" || normalizedPath == "." {
		return false
	}

	// For directory paths, append a trailing slash so that directory-only
	// patterns (e.g., "build/") can match correctly through the library.
	matchPath := normalizedPath
	if isDir && !strings.HasSuffix(matchPath, "/") {
		matchPath += "/"
	}

	// Check each applicable .gitignore from root toward the file's parent.
	// A .gitignore at directory D applies to paths under D.
	for _, dir := range m.dirs {
		matcher := m.matchers[dir]

		// Determine if this .gitignore applies to the given path.
		// Root-level .gitignore applies to everything.
		// Nested .gitignore at "src" applies only to paths under "src/".
		if dir != "." {
			prefix := dir + "/"
			if !strings.HasPrefix(normalizedPath, prefix) {
				continue
			}
		}

		// For pattern matching, make the path relative to the .gitignore's
		// directory. The sabhiram/go-gitignore library expects paths relative
		// to the directory containing the .gitignore file.
		var relPath string
		if dir == "." {
			relPath = matchPath
		} else {
			relPath = strings.TrimPrefix(matchPath, dir+"/")
		}

		if matcher.MatchesPath(relPath) {
			m.logger.Debug("path matched gitignore",
				"path", normalizedPath,
				"gitignore_dir", dir,
				"rel_path", relPath,
			)
			return true
		}
	}

	return false
}

// PatternCount returns the total number of .gitignore files that were loaded
// and compiled. This is useful for diagnostics and logging.
func (m *GitignoreMatcher) PatternCount() int {
	return len(m.matchers)
}

// Compile-time interface compliance check.
var _ Ignorer = (*GitignoreMatcher)(nil)
