package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// secretTypeLabels maps internal SecretType identifiers to human-readable
// label strings used in inline summaries and text reports.
var secretTypeLabels = map[string]string{
	"aws_access_key":       "AWS access key",
	"aws_secret_key":       "AWS secret key",
	"github_token":         "GitHub token",
	"gitlab_token":         "GitLab token",
	"private_key_block":    "private key block",
	"connection_string":    "connection string",
	"jwt_token":            "JWT token",
	"generic_api_key":      "API key",
	"high_entropy_secret":  "high entropy secret",
	"slack_token":          "Slack token",
	"stripe_secret_key":    "Stripe secret key",
	"google_api_key":       "Google API key",
	"twilio_api_key":       "Twilio API key",
	"sendgrid_api_key":     "SendGrid API key",
	"mailchimp_api_key":    "Mailchimp API key",
	"heroku_api_key":       "Heroku API key",
	"basic_auth":           "basic auth credential",
	"bearer_token":         "bearer token",
}

// ReportEntry is a single redaction entry in the detailed report. It mirrors
// RedactionMatch but is flattened for JSON/text output and never reveals the
// actual secret value.
type ReportEntry struct {
	File        string     `json:"file"`
	Line        int        `json:"line"`
	RuleID      string     `json:"rule_id"`
	SecretType  string     `json:"secret_type"`
	Confidence  Confidence `json:"confidence"`
	Replacement string     `json:"replacement"`
}

// ReportSummary is the aggregated statistics section of the detailed report.
type ReportSummary struct {
	TotalRedactions int            `json:"total_redactions"`
	FilesAffected   int            `json:"files_affected"`
	ByType          map[string]int `json:"by_type"`
	ByConfidence    map[string]int `json:"by_confidence"`
}

// Report is the top-level struct serialised to JSON (or text) when
// --redaction-report is active.
type Report struct {
	GeneratedAt         time.Time     `json:"generated_at"`
	Profile             string        `json:"profile"`
	ConfidenceThreshold string        `json:"confidence_threshold"`
	Summary             ReportSummary `json:"summary"`
	Redactions          []ReportEntry `json:"redactions"`
}

// ReportGenerator builds and writes redaction reports.
type ReportGenerator struct{}

// NewReportGenerator returns a new ReportGenerator ready for use.
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// BuildReport builds a Report from a RedactionSummary, a slice of individual
// matches, and metadata strings for the profile and confidence threshold.
//
// Matches are sorted by FilePath then LineNumber for consistent output.
// The actual secret values are never included; only metadata is recorded.
func (g *ReportGenerator) BuildReport(summary RedactionSummary, matches []RedactionMatch, profile, threshold string) Report {
	// Sort matches by FilePath ascending, then LineNumber ascending.
	sorted := make([]RedactionMatch, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FilePath != sorted[j].FilePath {
			return sorted[i].FilePath < sorted[j].FilePath
		}
		return sorted[i].LineNumber < sorted[j].LineNumber
	})

	// Map RedactionMatch → ReportEntry (never include secret value).
	entries := make([]ReportEntry, 0, len(sorted))
	for _, m := range sorted {
		entries = append(entries, ReportEntry{
			File:        m.FilePath,
			Line:        m.LineNumber,
			RuleID:      m.RuleID,
			SecretType:  m.SecretType,
			Confidence:  m.Confidence,
			Replacement: m.Replacement,
		})
	}

	// Build the ByConfidence map with string keys ("high", "medium", "low").
	byConf := make(map[string]int, len(summary.ByConfidence))
	for k, v := range summary.ByConfidence {
		byConf[string(k)] = v
	}

	// Deep-copy ByType so the Report owns its own map.
	byType := make(map[string]int, len(summary.ByType))
	for k, v := range summary.ByType {
		byType[k] = v
	}

	return Report{
		GeneratedAt:         time.Now().UTC(),
		Profile:             profile,
		ConfidenceThreshold: threshold,
		Summary: ReportSummary{
			TotalRedactions: summary.TotalCount,
			FilesAffected:   summary.FileCount,
			ByType:          byType,
			ByConfidence:    byConf,
		},
		Redactions: entries,
	}
}

// GenerateJSON serialises report to indented JSON bytes.
func (g *ReportGenerator) GenerateJSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling report to JSON: %w", err)
	}
	return data, nil
}

// GenerateText returns a human-readable text representation of report.
//
// When there are no redactions, the output is simply "No secrets detected."
//
// Example (with redactions):
//
//	Harvx Redaction Report
//	======================
//	Generated: 2026-02-16 10:30:00 UTC
//	Profile: finvault
//	Threshold: high
//
//	Summary: 7 redactions in 4 files
//	  3 API keys (high confidence)
//	  ...
//
//	Details:
//	  config/database.go:42    [connection_string]  high
//	  ...
func (g *ReportGenerator) GenerateText(report Report) string {
	if report.Summary.TotalRedactions == 0 {
		return "No secrets detected."
	}

	var b strings.Builder

	// Header.
	b.WriteString("Harvx Redaction Report\n")
	b.WriteString("======================\n")
	fmt.Fprintf(&b, "Generated: %s\n", report.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "Profile: %s\n", report.Profile)
	fmt.Fprintf(&b, "Threshold: %s\n", report.ConfidenceThreshold)

	// Summary line.
	b.WriteString("\n")
	redactionWord := "redactions"
	if report.Summary.TotalRedactions == 1 {
		redactionWord = "redaction"
	}
	fmt.Fprintf(&b, "Summary: %d %s in %d files\n",
		report.Summary.TotalRedactions, redactionWord, report.Summary.FilesAffected)

	// Per-type breakdown sorted by count descending, then type name ascending.
	type typeCount struct {
		secretType string
		count      int
	}
	typeCounts := make([]typeCount, 0, len(report.Summary.ByType))
	for t, c := range report.Summary.ByType {
		typeCounts = append(typeCounts, typeCount{t, c})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		if typeCounts[i].count != typeCounts[j].count {
			return typeCounts[i].count > typeCounts[j].count
		}
		return typeCounts[i].secretType < typeCounts[j].secretType
	})

	for _, tc := range typeCounts {
		label := humanLabel(tc.secretType)
		pluralLabel := pluralize(label, tc.count)
		// Find the dominant confidence for this type from the entries.
		conf := dominantConfidenceForType(report.Redactions, tc.secretType)
		fmt.Fprintf(&b, "  %d %s (%s confidence)\n", tc.count, pluralLabel, conf)
	}

	// Details section.
	b.WriteString("\nDetails:\n")
	for _, entry := range report.Redactions {
		loc := fmt.Sprintf("%s:%d", entry.File, entry.Line)
		fmt.Fprintf(&b, "  %-40s [%-25s]  %s\n", loc, entry.SecretType, string(entry.Confidence))
	}

	return b.String()
}

// WriteReport writes report to path, auto-detecting the format from the file
// extension. Files ending in ".txt" or ".text" use text format; all others
// use JSON format.
//
// Parent directories are created with permissions 0755 if they do not exist.
// The file is written with permissions 0644.
func (g *ReportGenerator) WriteReport(report Report, path string) error {
	// Detect format from extension.
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	if ext == ".txt" || ext == ".text" {
		data = []byte(g.GenerateText(report))
	} else {
		var err error
		data, err = g.GenerateJSON(report)
		if err != nil {
			return fmt.Errorf("writing report to %s: %w", path, err)
		}
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("writing report to %s: %w", path, err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing report to %s: %w", path, err)
	}
	return nil
}

// FormatInlineSummary produces a compact inline summary suitable for embedding
// in output documents.
//
// Examples:
//
//	"7 redactions (3 API keys, 2 connection strings, 1 private key block, 1 JWT token)"
//	"1 redaction (1 API key)"
//	"No secrets detected"
//	"5 redactions"   (when ByType is empty)
func (g *ReportGenerator) FormatInlineSummary(summary RedactionSummary) string {
	if summary.TotalCount == 0 {
		return "No secrets detected"
	}

	redactionWord := "redactions"
	if summary.TotalCount == 1 {
		redactionWord = "redaction"
	}

	if len(summary.ByType) == 0 {
		return fmt.Sprintf("%d %s", summary.TotalCount, redactionWord)
	}

	// Sort types by count descending, then type name ascending for determinism.
	type typeCount struct {
		secretType string
		count      int
	}
	typeCounts := make([]typeCount, 0, len(summary.ByType))
	for t, c := range summary.ByType {
		typeCounts = append(typeCounts, typeCount{t, c})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		if typeCounts[i].count != typeCounts[j].count {
			return typeCounts[i].count > typeCounts[j].count
		}
		return typeCounts[i].secretType < typeCounts[j].secretType
	})

	parts := make([]string, 0, len(typeCounts))
	for _, tc := range typeCounts {
		label := humanLabel(tc.secretType)
		parts = append(parts, fmt.Sprintf("%d %s", tc.count, pluralize(label, tc.count)))
	}

	return fmt.Sprintf("%d %s (%s)", summary.TotalCount, redactionWord, strings.Join(parts, ", "))
}

// humanLabel returns the human-readable label for a secret type. Known types
// are looked up in secretTypeLabels; unknown types have underscores replaced
// with spaces and the first rune uppercased.
func humanLabel(secretType string) string {
	if label, ok := secretTypeLabels[secretType]; ok {
		return label
	}
	// Fallback: replace underscores with spaces and capitalise first rune.
	s := strings.ReplaceAll(secretType, "_", " ")
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// pluralize appends "s" to label when count != 1.
func pluralize(label string, count int) string {
	if count == 1 {
		return label
	}
	return label + "s"
}

// dominantConfidenceForType returns the highest confidence level observed
// among the entries whose SecretType equals secretType, or "unknown" when
// no matching entry exists.
func dominantConfidenceForType(entries []ReportEntry, secretType string) string {
	best := -1
	bestLabel := "unknown"
	for _, e := range entries {
		if e.SecretType != secretType {
			continue
		}
		lvl := confidenceLevel(e.Confidence)
		if lvl > best {
			best = lvl
			bestLabel = string(e.Confidence)
		}
	}
	return bestLabel
}
