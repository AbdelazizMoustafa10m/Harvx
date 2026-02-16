package discovery

import (
	"log/slog"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// DefaultIgnorePatterns contains the built-in ignore patterns that Harvx
// always applies unless explicitly overridden. These patterns follow the
// gitignore syntax and cover common directories, environment files,
// certificates, sensitive naming patterns, lock files, compiled artifacts,
// and OS/editor metadata files.
//
// Reference: PRD Sections 5.1, 5.8, 7.3
var DefaultIgnorePatterns = []string{
	// Version control and build directories
	".git/",
	"node_modules/",
	"dist/",
	"build/",
	"coverage/",
	"__pycache__/",
	".next/",
	"target/",
	"vendor/",
	".harvx/",

	// Environment files (PRD 7.3)
	".env",
	".env.*",

	// Certificate and key files (PRD 7.3)
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",

	// Sensitive naming patterns (PRD 7.3)
	"*secret*",
	"*credential*",
	"*password*",

	// Lock files
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"Gemfile.lock",
	"Cargo.lock",
	"go.sum",
	"poetry.lock",

	// Compiled artifacts
	"*.pyc",
	"*.pyo",
	"*.class",
	"*.o",
	"*.obj",
	"*.exe",
	"*.dll",
	"*.so",
	"*.dylib",

	// OS and editor files
	".DS_Store",
	"Thumbs.db",
	".idea/",
	".vscode/",
	"*.swp",
	"*.swo",
}

// SensitivePatterns contains the subset of DefaultIgnorePatterns that
// represent security-sensitive files. When these defaults are overridden
// (e.g., a user explicitly includes *.pem), a warning is logged. This
// slice is exported so that downstream code can check whether an override
// affects a sensitive pattern.
//
// Reference: PRD Section 7.3
var SensitivePatterns = []string{
	// Environment files
	".env",
	".env.*",

	// Certificate and key files
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",

	// Sensitive naming patterns
	"*secret*",
	"*credential*",
	"*password*",
}

// DefaultIgnoreMatcher compiles the DefaultIgnorePatterns into a matcher that
// implements the Ignorer interface. It uses the same sabhiram/go-gitignore
// library as GitignoreMatcher for consistent pattern evaluation.
type DefaultIgnoreMatcher struct {
	matcher *gitignore.GitIgnore
	logger  *slog.Logger
}

// NewDefaultIgnoreMatcher creates a new DefaultIgnoreMatcher by compiling all
// DefaultIgnorePatterns. This function does not return an error because the
// default patterns are compile-time constants that are always valid.
func NewDefaultIgnoreMatcher() *DefaultIgnoreMatcher {
	compiled := gitignore.CompileIgnoreLines(DefaultIgnorePatterns...)

	logger := slog.Default().With("component", "default-ignore")
	logger.Debug("default ignore matcher initialized",
		"pattern_count", len(DefaultIgnorePatterns),
	)

	return &DefaultIgnoreMatcher{
		matcher: compiled,
		logger:  logger,
	}
}

// IsIgnored reports whether the given path matches any of the default ignore
// patterns. The path must be relative to the repository root (using forward
// slashes or OS-native separators). The isDir parameter indicates whether the
// path represents a directory, which is needed for directory-only patterns.
func (d *DefaultIgnoreMatcher) IsIgnored(path string, isDir bool) bool {
	normalizedPath := filepath.ToSlash(path)
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	if normalizedPath == "" || normalizedPath == "." {
		return false
	}

	// For directory paths, append a trailing slash so that directory-only
	// patterns (e.g., "build/") can match correctly.
	matchPath := normalizedPath
	if isDir && !strings.HasSuffix(matchPath, "/") {
		matchPath += "/"
	}

	if d.matcher.MatchesPath(matchPath) {
		d.logger.Debug("path matched default ignore",
			"path", normalizedPath,
		)
		return true
	}

	return false
}

// PatternCount returns the number of default ignore patterns. This is useful
// for diagnostics and logging.
func (d *DefaultIgnoreMatcher) PatternCount() int {
	return len(DefaultIgnorePatterns)
}

// sensitiveMatcher is a pre-compiled matcher for SensitivePatterns. It is
// initialized once and reused across calls to IsSensitivePath.
var sensitiveMatcher = gitignore.CompileIgnoreLines(SensitivePatterns...)

// IsSensitivePath reports whether the given path matches any of the
// SensitivePatterns. This is used to emit warnings when sensitive defaults
// are overridden by user configuration.
func IsSensitivePath(path string) bool {
	normalizedPath := filepath.ToSlash(path)
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	if normalizedPath == "" {
		return false
	}

	return sensitiveMatcher.MatchesPath(normalizedPath)
}

// Compile-time interface compliance check.
var _ Ignorer = (*DefaultIgnoreMatcher)(nil)
