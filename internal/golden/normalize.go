// Package golden provides infrastructure for golden file testing in Harvx.
// Golden tests run the full pipeline against curated sample repositories and
// compare output byte-for-byte against reference files stored in
// testdata/expected-output/. When output intentionally changes, the -update
// flag regenerates the golden files.
//
// Non-deterministic values (timestamps, hashes, timing data, absolute paths)
// are normalized before comparison so that golden tests remain stable across
// machines and runs.
package golden

import (
	"regexp"
	"strings"
)

// Normalize replaces non-deterministic values in output with stable placeholders:
//   - ISO 8601 timestamps -> [TIMESTAMP]
//   - Content hashes (hex after "Hash:" or "hash") -> [HASH]
//   - Timing values (e.g., "123ms", "1.5s") -> [TIME]
//   - Absolute paths containing the working directory -> relative paths
//
// The working directory is provided via the dir parameter. If dir is empty,
// absolute path normalization is skipped.
func Normalize(content []byte, dir string) []byte {
	s := string(content)

	// Replace ISO 8601 timestamps.
	s = reTimestamp.ReplaceAllString(s, "[TIMESTAMP]")

	// Replace hex hashes after "Hash:" or "hash:" or "hash=".
	s = reHash.ReplaceAllString(s, "${1}[HASH]")

	// Replace content_hash XML element values.
	s = reContentHashXML.ReplaceAllString(s, "<content_hash>[HASH]</content_hash>")

	// Replace timing values.
	s = reTiming.ReplaceAllString(s, "[TIME]")

	// Replace absolute paths with relative paths.
	if dir != "" {
		// Normalize the directory path to use forward slashes for consistency.
		normalizedDir := strings.ReplaceAll(dir, `\`, "/")
		if !strings.HasSuffix(normalizedDir, "/") {
			normalizedDir += "/"
		}
		s = strings.ReplaceAll(s, normalizedDir, "")

		// Also handle the dir without trailing slash (for exact path matches).
		dirNoSlash := strings.TrimSuffix(normalizedDir, "/")
		s = strings.ReplaceAll(s, dirNoSlash, ".")
	}

	return []byte(s)
}

// reTimestamp matches ISO 8601 timestamps:
//
//	2025-01-01T00:00:00Z
//	2025-01-01T00:00:00+00:00
//	2026-02-26T14:30:45.123Z
var reTimestamp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?[Z+\-][\d:]*`)

// reHash matches hex hash values after "Hash:" or "hash:" or "hash=":
//
//	Content Hash | abc123def456
//	content_hash>abc123def456
//	hash: abc123def456
var reHash = regexp.MustCompile(`(?i)(hash[":\s|>=]+)([0-9a-f]{8,})`)

// reContentHashXML matches <content_hash>...</content_hash> elements.
var reContentHashXML = regexp.MustCompile(`<content_hash>[0-9a-f]+</content_hash>`)

// reTiming matches timing/duration values:
//
//	123ms, 1.5s, 200µs, 50ns, 2.3ms
var reTiming = regexp.MustCompile(`\d+(\.\d+)?(ms|µs|ns|s)\b`)