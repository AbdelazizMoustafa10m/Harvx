package discovery

import (
	"log/slog"
)

// Ignorer is the interface for all ignore-pattern matchers in the harvx
// pipeline. Each Ignorer implementation evaluates whether a given path should
// be excluded from context generation. The path must be relative to the
// repository root, using forward slashes. The isDir parameter indicates
// whether the path represents a directory (needed for directory-only patterns).
type Ignorer interface {
	IsIgnored(path string, isDir bool) bool
}

// CompositeIgnorer chains multiple Ignorer implementations and returns true
// if ANY source matches the given path. The evaluation order follows the
// ignore chain defined in the PRD: defaults, .gitignore, .harvxignore,
// CLI --exclude flags.
type CompositeIgnorer struct {
	ignorers []Ignorer
	logger   *slog.Logger
}

// NewCompositeIgnorer creates a new CompositeIgnorer that chains the provided
// ignorers. A path is considered ignored if any single ignorer matches it.
// Nil ignorers in the variadic list are silently skipped.
func NewCompositeIgnorer(ignorers ...Ignorer) *CompositeIgnorer {
	filtered := make([]Ignorer, 0, len(ignorers))
	for _, ig := range ignorers {
		if ig != nil {
			filtered = append(filtered, ig)
		}
	}

	return &CompositeIgnorer{
		ignorers: filtered,
		logger:   slog.Default().With("component", "composite-ignorer"),
	}
}

// IsIgnored reports whether the given path should be ignored according to any
// of the chained ignore sources. Returns true if ANY ignorer matches the path.
func (c *CompositeIgnorer) IsIgnored(path string, isDir bool) bool {
	for _, ig := range c.ignorers {
		if ig.IsIgnored(path, isDir) {
			return true
		}
	}
	return false
}

// IgnorerCount returns the number of active ignorers in the chain. This is
// useful for diagnostics and logging.
func (c *CompositeIgnorer) IgnorerCount() int {
	return len(c.ignorers)
}

// Compile-time interface compliance check.
var _ Ignorer = (*CompositeIgnorer)(nil)
