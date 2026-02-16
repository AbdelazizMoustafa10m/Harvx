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

// HarvxignoreMatcher loads and evaluates .harvxignore patterns hierarchically.
// It uses the same gitignore pattern syntax and hierarchical model as
// GitignoreMatcher, but searches for .harvxignore files instead of .gitignore.
// This allows users to define tool-specific ignore patterns separate from
// their .gitignore configuration.
//
// Reference: PRD Section 5.1 -- "Respects .harvxignore for tool-specific
// ignore patterns."
type HarvxignoreMatcher struct {
	root     string
	matchers map[string]*gitignore.GitIgnore
	// dirs stores the sorted list of directory keys for deterministic
	// iteration from root toward the file's parent directory.
	dirs   []string
	logger *slog.Logger
}

// NewHarvxignoreMatcher creates a new HarvxignoreMatcher rooted at the given
// directory. It walks rootDir to discover all .harvxignore files and compiles
// their patterns using sabhiram/go-gitignore.
//
// If no .harvxignore files exist, the matcher returns successfully and
// IsIgnored will always return false. Missing or unreadable .harvxignore files
// at individual directory levels are logged and skipped without error.
func NewHarvxignoreMatcher(rootDir string) (*HarvxignoreMatcher, error) {
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

	logger := slog.Default().With("component", "harvxignore")

	m := &HarvxignoreMatcher{
		root:     absRoot,
		matchers: make(map[string]*gitignore.GitIgnore),
		logger:   logger,
	}

	if err := m.discoverHarvxignoreFiles(); err != nil {
		return nil, fmt.Errorf("discovering .harvxignore files in %s: %w", absRoot, err)
	}

	logger.Debug("harvxignore matcher initialized",
		"root", absRoot,
		"harvxignore_count", len(m.matchers),
	)

	return m, nil
}

// discoverHarvxignoreFiles walks the root directory tree to find all
// .harvxignore files and compiles each one.
func (m *HarvxignoreMatcher) discoverHarvxignoreFiles() error {
	err := filepath.WalkDir(m.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			m.logger.Debug("skipping unreadable path", "path", path, "error", err)
			return filepath.SkipDir
		}

		// Skip .git directory entirely.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		// We only care about .harvxignore files.
		if d.IsDir() || d.Name() != ".harvxignore" {
			return nil
		}

		dirPath := filepath.Dir(path)
		relDir, err := filepath.Rel(m.root, dirPath)
		if err != nil {
			m.logger.Debug("skipping .harvxignore, cannot compute relative path",
				"path", path, "error", err)
			return nil
		}

		compiled, err := gitignore.CompileIgnoreFile(path)
		if err != nil {
			m.logger.Debug("skipping unreadable .harvxignore",
				"path", path, "error", err)
			return nil
		}

		// Normalize to use "." for the root directory.
		if relDir == "" {
			relDir = "."
		}

		m.matchers[relDir] = compiled
		m.logger.Debug("loaded .harvxignore", "dir", relDir, "path", path)

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
// the loaded .harvxignore rules. The path must be relative to the root
// directory (using forward slashes or OS-native separators). The isDir
// parameter indicates whether the path represents a directory, which is
// needed for directory-only patterns (patterns ending in /).
//
// The matcher evaluates .harvxignore files from the root directory down to
// the file's parent directory. A file is ignored if any ancestor's
// .harvxignore matches it. Negation patterns in a .harvxignore can override
// matches from the same .harvxignore file.
func (m *HarvxignoreMatcher) IsIgnored(path string, isDir bool) bool {
	normalizedPath := filepath.ToSlash(path)
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	if normalizedPath == "" || normalizedPath == "." {
		return false
	}

	matchPath := normalizedPath
	if isDir && !strings.HasSuffix(matchPath, "/") {
		matchPath += "/"
	}

	for _, dir := range m.dirs {
		matcher := m.matchers[dir]

		if dir != "." {
			prefix := dir + "/"
			if !strings.HasPrefix(normalizedPath, prefix) {
				continue
			}
		}

		var relPath string
		if dir == "." {
			relPath = matchPath
		} else {
			relPath = strings.TrimPrefix(matchPath, dir+"/")
		}

		if matcher.MatchesPath(relPath) {
			m.logger.Debug("path matched harvxignore",
				"path", normalizedPath,
				"harvxignore_dir", dir,
				"rel_path", relPath,
			)
			return true
		}
	}

	return false
}

// PatternCount returns the total number of .harvxignore files that were loaded
// and compiled. This is useful for diagnostics and logging.
func (m *HarvxignoreMatcher) PatternCount() int {
	return len(m.matchers)
}

// Compile-time interface compliance check.
var _ Ignorer = (*HarvxignoreMatcher)(nil)
