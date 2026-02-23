package security

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// sensitiveFilePatterns is the complete list of glob patterns identifying files
// that likely contain credentials or sensitive material. Files matching these
// patterns are excluded from discovery by default and trigger heightened
// scanning mode in the StreamRedactor when they do appear.
//
// Reference: PRD Section 7.3
var sensitiveFilePatterns = []string{
	// Environment files
	".env",
	".env.*",
	"**/.env",
	"**/.env.*",
	"*.env",

	// Certificate and key files
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",
	"*.jks",
	"*.keystore",

	// SSH private keys (exact filenames)
	"id_rsa",
	"id_dsa",
	"id_ecdsa",
	"id_ed25519",
	"**/id_rsa",
	"**/id_dsa",
	"**/id_ecdsa",
	"**/id_ed25519",

	// Auth and credential files
	".htpasswd",
	".netrc",
	".npmrc",
	"**/.htpasswd",
	"**/.netrc",
	"**/.npmrc",

	// Encrypted/signed files
	"*.gpg",
	"*.asc",

	// Sensitive naming patterns
	"*secret*",
	"*credential*",
	"*password*",
}

// SensitiveFilePatterns returns a copy of the default sensitive file
// exclusion patterns. The returned slice is safe to mutate; each call
// returns an independent copy.
func SensitiveFilePatterns() []string {
	result := make([]string, len(sensitiveFilePatterns))
	copy(result, sensitiveFilePatterns)
	return result
}

// IsSensitiveFile reports whether filePath matches any of the known sensitive
// file patterns. Matching uses doublestar glob semantics, identical to the
// rest of the Harvx filtering pipeline.
//
// filePath should be relative to the repository root. Both forward and
// backward slashes are normalized before matching.
func IsSensitiveFile(filePath string) bool {
	normalized := filepath.ToSlash(filePath)
	// Also check just the base file name for simple patterns like "*.pem".
	base := filepath.Base(normalized)
	for _, pattern := range sensitiveFilePatterns {
		// Try matching against the full path.
		if ok, _ := doublestar.Match(pattern, normalized); ok {
			return true
		}
		// Try matching pattern against just the base name for non-path patterns
		// (e.g. "*.pem" should match "config/server.pem").
		if !strings.Contains(pattern, "/") && !strings.HasPrefix(pattern, "**/") {
			if ok, _ := doublestar.Match(pattern, base); ok {
				return true
			}
		}
	}
	return false
}

// WarnIfSensitiveFile emits a slog.Warn log entry when filePath matches any
// sensitive file pattern, unless suppressWarning is true. This is called by
// the walker when a sensitive file is discovered (meaning the default
// exclusion was overridden by the user's profile configuration).
//
// The warning is informational only and does not block processing.
func WarnIfSensitiveFile(filePath string, suppressWarning bool, logger *slog.Logger) {
	if suppressWarning {
		return
	}
	normalized := filepath.ToSlash(filePath)
	base := filepath.Base(normalized)
	for _, pattern := range sensitiveFilePatterns {
		matched := false
		if ok, _ := doublestar.Match(pattern, normalized); ok {
			matched = true
		} else if !strings.Contains(pattern, "/") && !strings.HasPrefix(pattern, "**/") {
			if ok, _ := doublestar.Match(pattern, base); ok {
				matched = true
			}
		}
		if matched {
			logger.Warn("sensitive file included by profile override",
				"path", filePath,
				"matched_pattern", pattern,
			)
			return // Warn once per file (first matching pattern).
		}
	}
}
