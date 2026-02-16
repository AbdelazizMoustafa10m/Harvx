package discovery

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// PatternFilter applies include, exclude, and extension-based filtering to file
// paths during discovery. It implements the filtering logic described in PRD
// Section 5.1 for --include, --exclude, and -f flags.
//
// Filtering rules:
//   - When no include patterns or extension filters are set, all files pass.
//   - Include patterns and extension filters are combined with OR logic: a file
//     must match at least one include pattern or one extension filter to be kept.
//   - Exclude patterns take precedence over includes: if a file matches any
//     exclude pattern, it is removed regardless of include matches.
//   - Extension matching is case-insensitive.
//   - Patterns use doublestar syntax (e.g., "**/*.ts" matches deeply nested files).
type PatternFilter struct {
	includes   []string
	excludes   []string
	extensions []string // normalized to lowercase, without leading dot
	logger     *slog.Logger
}

// PatternFilterOptions holds the configuration for creating a new PatternFilter.
type PatternFilterOptions struct {
	// Includes is a list of doublestar glob patterns. If any are set, only
	// files matching at least one pattern (or one extension) are kept.
	Includes []string

	// Excludes is a list of doublestar glob patterns. Files matching any
	// exclude pattern are removed, regardless of include matches.
	Excludes []string

	// Extensions is a list of file extensions (without leading dots). This is
	// the shorthand for -f flag. Extensions are case-insensitive.
	Extensions []string
}

// NewPatternFilter creates a new PatternFilter from the provided options.
// Extension values are normalized to lowercase with leading dots stripped.
// Copies are made of all input slices to prevent external mutation.
func NewPatternFilter(opts PatternFilterOptions) *PatternFilter {
	// Copy and normalize extensions: strip leading dots, lowercase.
	extensions := make([]string, len(opts.Extensions))
	for i, ext := range opts.Extensions {
		ext = strings.TrimLeft(ext, ".")
		extensions[i] = strings.ToLower(ext)
	}

	// Copy includes and excludes.
	includes := make([]string, len(opts.Includes))
	copy(includes, opts.Includes)

	excludes := make([]string, len(opts.Excludes))
	copy(excludes, opts.Excludes)

	logger := slog.Default().With("component", "pattern-filter")
	logger.Debug("pattern filter initialized",
		"includes", len(includes),
		"excludes", len(excludes),
		"extensions", len(extensions),
	)

	return &PatternFilter{
		includes:   includes,
		excludes:   excludes,
		extensions: extensions,
		logger:     logger,
	}
}

// Matches reports whether the given path should be included in the output.
// The path should be relative to the repository root, using forward slashes.
// Returns true if the file passes all filter criteria.
//
// Logic:
//  1. If the path matches any exclude pattern, return false (exclude wins).
//  2. If no include patterns and no extension filters are set, return true (pass-through).
//  3. If the path matches any include pattern OR any extension filter, return true.
//  4. Otherwise, return false.
func (f *PatternFilter) Matches(path string) bool {
	// Normalize to forward slashes for consistent matching.
	normalizedPath := filepath.ToSlash(path)
	normalizedPath = strings.TrimPrefix(normalizedPath, "./")

	if normalizedPath == "" {
		return false
	}

	// Step 1: Check excludes first (exclude always wins).
	for _, pattern := range f.excludes {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			f.logger.Debug("invalid exclude pattern",
				"pattern", pattern,
				"error", err,
			)
			continue
		}
		if matched {
			f.logger.Debug("path excluded by pattern",
				"path", normalizedPath,
				"pattern", pattern,
			)
			return false
		}
	}

	// Step 2: If no include patterns and no extension filters, pass through.
	if len(f.includes) == 0 && len(f.extensions) == 0 {
		return true
	}

	// Step 3: Check include patterns (OR logic).
	for _, pattern := range f.includes {
		matched, err := doublestar.Match(pattern, normalizedPath)
		if err != nil {
			f.logger.Debug("invalid include pattern",
				"pattern", pattern,
				"error", err,
			)
			continue
		}
		if matched {
			return true
		}
	}

	// Step 4: Check extension filters (OR logic, case-insensitive).
	if len(f.extensions) > 0 {
		ext := strings.TrimLeft(filepath.Ext(normalizedPath), ".")
		ext = strings.ToLower(ext)
		for _, filterExt := range f.extensions {
			if ext == filterExt {
				return true
			}
		}
	}

	// No include or extension matched.
	return false
}

// HasFilters reports whether any include, exclude, or extension filters are
// configured. When false, the filter is a pass-through and Matches always
// returns true.
func (f *PatternFilter) HasFilters() bool {
	return len(f.includes) > 0 || len(f.excludes) > 0 || len(f.extensions) > 0
}
