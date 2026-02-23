// Package tokenizer provides token counting implementations for LLM context
// documents. This file implements report data structures and formatters for
// presenting token count summaries to the user via the CLI.
package tokenizer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/harvx/harvx/internal/pipeline"
)

// TierLabel maps tier numbers to human-readable display names.
var TierLabel = map[int]string{
	0: "Config",
	1: "Source",
	2: "Secondary",
	3: "Tests",
	4: "Docs",
	5: "CI/Lock",
}

// tierLabelFor returns the display name for a tier number, falling back to
// a generated label for tiers outside the known range.
func tierLabelFor(tier int) string {
	if label, ok := TierLabel[tier]; ok {
		return label
	}
	return fmt.Sprintf("Tier%d", tier)
}

// TierReportStat holds per-tier file and token counts.
type TierReportStat struct {
	// FileCount is the number of files in this tier.
	FileCount int

	// TokenCount is the total number of tokens across all files in this tier.
	TokenCount int
}

// TokenReport holds the summary data for a full token count report.
type TokenReport struct {
	// TokenizerName is the encoding name used (e.g., "cl100k_base").
	TokenizerName string

	// TotalFiles is the total number of files included in the report.
	TotalFiles int

	// TotalTokens is the sum of token counts across all files.
	TotalTokens int

	// Budget is the configured max token budget (0 means unlimited).
	Budget int

	// TierStats maps tier number to per-tier statistics.
	TierStats map[int]*TierReportStat
}

// NewTokenReport builds a TokenReport from a set of file descriptors.
// tokenizerName is the encoding name (e.g., "cl100k_base").
// budget is the configured max token budget (0 = unlimited).
func NewTokenReport(files []*pipeline.FileDescriptor, tokenizerName string, budget int) *TokenReport {
	r := &TokenReport{
		TokenizerName: tokenizerName,
		Budget:        budget,
		TierStats:     make(map[int]*TierReportStat),
	}

	for _, fd := range files {
		if fd == nil {
			continue
		}
		r.TotalFiles++
		r.TotalTokens += fd.TokenCount

		stat, ok := r.TierStats[fd.Tier]
		if !ok {
			stat = &TierReportStat{}
			r.TierStats[fd.Tier] = stat
		}
		stat.FileCount++
		stat.TokenCount += fd.TokenCount
	}

	return r
}

// Format renders the token report as a plain-text string suitable for printing
// to stderr. Uses unicode box-drawing chars for the separator line.
func (r *TokenReport) Format() string {
	var sb strings.Builder

	title := fmt.Sprintf("Token Report (%s)", r.TokenizerName)
	separator := strings.Repeat("─", len(title)+2)

	sb.WriteString(title + "\n")
	sb.WriteString(separator + "\n")
	fmt.Fprintf(&sb, "Total files:  %s\n", FormatInt(r.TotalFiles))
	fmt.Fprintf(&sb, "Total tokens: %s\n", FormatInt(r.TotalTokens))

	if r.Budget > 0 {
		pct := int(float64(r.TotalTokens) / float64(r.Budget) * 100)
		fmt.Fprintf(&sb, "Budget:       %s (%d%% used)\n", FormatInt(r.Budget), pct)
	} else {
		sb.WriteString("Budget:       unlimited\n")
	}

	if len(r.TierStats) > 0 {
		sb.WriteString("\nBy Tier:\n")
		tiers := make([]int, 0, len(r.TierStats))
		for t := range r.TierStats {
			tiers = append(tiers, t)
		}
		sort.Ints(tiers)

		for _, tier := range tiers {
			stat := r.TierStats[tier]
			label := tierLabelFor(tier)
			fmt.Fprintf(&sb, "  Tier %d (%s): %s files  %s tokens\n",
				tier,
				label,
				FormatInt(stat.FileCount),
				FormatInt(stat.TokenCount),
			)
		}
	}

	return sb.String()
}

// TopFilesEntry holds data for a single file in the top-N listing.
type TopFilesEntry struct {
	// Path is the relative file path.
	Path string

	// TokenCount is the number of tokens in this file.
	TokenCount int

	// Tier is the relevance tier of this file.
	Tier int
}

// TopFilesReport holds the top-N files by token count.
type TopFilesReport struct {
	// N is the requested limit (0 means all files were included).
	N int

	// Files is the sorted list of entries (descending by TokenCount).
	Files []TopFilesEntry
}

// NewTopFilesReport builds a TopFilesReport from file descriptors.
// Files are sorted by TokenCount descending. n=0 includes all files.
func NewTopFilesReport(files []*pipeline.FileDescriptor, n int) *TopFilesReport {
	entries := make([]TopFilesEntry, 0, len(files))
	for _, fd := range files {
		if fd == nil {
			continue
		}
		entries = append(entries, TopFilesEntry{
			Path:       fd.Path,
			TokenCount: fd.TokenCount,
			Tier:       fd.Tier,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TokenCount > entries[j].TokenCount
	})

	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}

	return &TopFilesReport{N: n, Files: entries}
}

// Format renders the top-N files report as a plain-text string.
func (r *TopFilesReport) Format() string {
	var sb strings.Builder

	label := "All Files"
	if r.N > 0 {
		label = fmt.Sprintf("Top %d Files", r.N)
	}

	title := fmt.Sprintf("%s by Token Count:", label)
	separator := strings.Repeat("─", len(title)+2)

	sb.WriteString(title + "\n")
	sb.WriteString(separator + "\n")

	if len(r.Files) == 0 {
		sb.WriteString("  (no files)\n")
		return sb.String()
	}

	for i, entry := range r.Files {
		tierLabel := tierLabelFor(entry.Tier)
		fmt.Fprintf(&sb, " %2d. %-50s  %s tokens  (Tier %d: %s)\n",
			i+1,
			entry.Path,
			FormatInt(entry.TokenCount),
			entry.Tier,
			tierLabel,
		)
	}

	return sb.String()
}

// HeatmapEntry holds data for a single file in the token density heatmap.
type HeatmapEntry struct {
	// Path is the relative file path.
	Path string

	// Lines is the number of lines in the file.
	Lines int

	// Tokens is the number of tokens in the file.
	Tokens int

	// Density is the token density: tokens per line.
	// Files with 0 lines get density 0 (no division by zero).
	Density float64

	// Tier is the relevance tier of this file.
	Tier int
}

// HeatmapReport holds files sorted by token density (tokens per line) descending.
type HeatmapReport struct {
	// Files is the list of entries sorted by Density descending.
	Files []HeatmapEntry
}

// NewHeatmapReport builds a HeatmapReport from file descriptors.
// lineCounts maps fd.Path -> number of lines in that file.
// Files with 0 lines get density 0 (no division by zero).
// Nil files and nil lineCounts are handled gracefully.
func NewHeatmapReport(files []*pipeline.FileDescriptor, lineCounts map[string]int) *HeatmapReport {
	entries := make([]HeatmapEntry, 0, len(files))

	for _, fd := range files {
		if fd == nil {
			continue
		}

		lines := 0
		if lineCounts != nil {
			lines = lineCounts[fd.Path]
		}

		var density float64
		if lines > 0 {
			density = float64(fd.TokenCount) / float64(lines)
		}

		entries = append(entries, HeatmapEntry{
			Path:    fd.Path,
			Lines:   lines,
			Tokens:  fd.TokenCount,
			Density: density,
			Tier:    fd.Tier,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Density > entries[j].Density
	})

	return &HeatmapReport{Files: entries}
}

// Format renders the heatmap as a plain-text string sorted by density descending.
func (r *HeatmapReport) Format() string {
	var sb strings.Builder

	title := "Token Heatmap (tokens per line):"
	separator := strings.Repeat("─", len(title)+2)

	sb.WriteString(title + "\n")
	sb.WriteString(separator + "\n")

	if len(r.Files) == 0 {
		sb.WriteString("  (no files)\n")
		return sb.String()
	}

	for i, entry := range r.Files {
		fmt.Fprintf(&sb, " %2d. %-50s  %.1f tok/line  (%s lines, %s tokens)\n",
			i+1,
			entry.Path,
			entry.Density,
			FormatInt(entry.Lines),
			FormatInt(entry.Tokens),
		)
	}

	return sb.String()
}

// FormatInt formats an integer with comma separators (e.g., 89420 -> "89,420").
// Exported for use in CLI formatting code.
func FormatInt(n int) string {
	if n < 0 {
		return "-" + FormatInt(-n)
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas every 3 digits from the right.
	var result []byte
	start := len(s) % 3
	if start == 0 {
		start = 3
	}
	result = append(result, s[:start]...)
	for i := start; i < len(s); i += 3 {
		result = append(result, ',')
		result = append(result, s[i:i+3]...)
	}

	return string(result)
}
