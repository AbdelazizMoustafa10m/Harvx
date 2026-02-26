// Package util provides common utility functions for gosync.
package util

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSize parses a human-readable file size string like "50MB" into bytes.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing size %q: %w", s, err)
			}
			return int64(num * float64(mult)), nil
		}
	}

	return 0, fmt.Errorf("unknown size format: %q", s)
}

// FormatSize converts bytes into a human-readable string.
func FormatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// Unique returns a new slice with duplicate strings removed, preserving order.
func Unique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))

	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}

	return result
}