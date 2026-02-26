package search

import "strings"

// Match performs case-insensitive matching on path. It first tries a simple
// substring match (most common case) and falls back to a subsequence match
// for fuzzy behavior. Returns true if the query matches the path, and the
// byte indices of matched characters for highlighting.
func Match(path, query string) (bool, []int) {
	if query == "" {
		return true, nil
	}

	lowerPath := strings.ToLower(path)
	lowerQuery := strings.ToLower(query)

	// First try simple substring match (most common case).
	if idx := strings.Index(lowerPath, lowerQuery); idx != -1 {
		indices := make([]int, len(lowerQuery))
		for i := range lowerQuery {
			indices[i] = idx + i
		}
		return true, indices
	}

	// Fall back to subsequence match for fuzzy behavior.
	var indices []int
	qi := 0
	for pi := 0; pi < len(lowerPath) && qi < len(lowerQuery); pi++ {
		if lowerPath[pi] == lowerQuery[qi] {
			indices = append(indices, pi)
			qi++
		}
	}

	if qi == len(lowerQuery) {
		return true, indices
	}

	return false, nil
}

// MatchSubstring performs a simple case-insensitive substring check.
// This is faster than Match when you do not need match indices.
func MatchSubstring(path, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(path), strings.ToLower(query))
}
