// Package stats implements the real-time statistics sidebar panel for the
// Harvx TUI. It displays live token counts, a visual budget bar, file counts,
// size estimates, compression savings, redaction count, tier breakdown, and
// profile/tokenizer information.
package stats

import (
	"fmt"
	"strings"
)

// FormatThousands formats an integer with comma-separated thousands.
// For example, 89420 becomes "89,420" and -1234 becomes "-1,234".
func FormatThousands(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	// Build digits from right to left.
	var b strings.Builder
	b.Grow(16)

	digits := fmt.Sprintf("%d", n)
	length := len(digits)

	for i, ch := range digits {
		if i > 0 && (length-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}

	if negative {
		return "-" + b.String()
	}
	return b.String()
}

// FormatSize formats a byte count as a human-readable size string.
// Uses KB for values < 1 MB and MB for values >= 1 MB.
func FormatSize(bytes int) string {
	if bytes < 0 {
		return "0 B"
	}

	const (
		kb = 1024
		mb = 1024 * 1024
	)

	switch {
	case bytes < kb:
		return fmt.Sprintf("%d B", bytes)
	case bytes < mb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	}
}

// EstimateOutputSize returns an estimated output size in bytes based on the
// token count. Uses the heuristic of ~4 bytes per token (average English text).
func EstimateOutputSize(tokens int) int {
	if tokens < 0 {
		return 0
	}
	return tokens * 4
}

// FormatPercentage formats a float64 as a percentage string with one decimal
// place. Values are clamped to the range [0, 100].
func FormatPercentage(pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// Truncate truncates a string to the given maximum width, appending an
// ellipsis if truncation occurred. If maxWidth is less than 4, the string
// is hard-truncated without an ellipsis.
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth < 4 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}
