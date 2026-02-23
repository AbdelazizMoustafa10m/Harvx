package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeMatch builds a RedactionMatch with the supplied fields and sensible
// defaults for the rest. t.Helper() is called so failures point to the
// call site.
func makeMatch(t *testing.T, filePath string, lineNum int, secretType string, conf Confidence) RedactionMatch {
	t.Helper()
	return RedactionMatch{
		RuleID:      "test-rule",
		SecretType:  secretType,
		Confidence:  conf,
		FilePath:    filePath,
		LineNumber:  lineNum,
		StartCol:    0,
		EndCol:      20,
		Replacement: FormatReplacement(secretType),
	}
}

// makeSummary builds a RedactionSummary from a slice of matches.
// fileCount must be provided explicitly because the caller decides how many
// distinct files the matches belong to.
func makeSummary(t *testing.T, matches []RedactionMatch, fileCount int) RedactionSummary {
	t.Helper()
	byType := make(map[string]int)
	byConf := make(map[Confidence]int)
	for _, m := range matches {
		byType[m.SecretType]++
		byConf[m.Confidence]++
	}
	return RedactionSummary{
		TotalCount:   len(matches),
		ByType:       byType,
		ByConfidence: byConf,
		FileCount:    fileCount,
	}
}

// newGenerator is a convenience wrapper so each test doesn't repeat
// the constructor call.
func newGenerator() *ReportGenerator {
	return NewReportGenerator()
}

// ---------------------------------------------------------------------------
// NewReportGenerator
// ---------------------------------------------------------------------------

func TestNewReportGenerator_NotNil(t *testing.T) {
	g := NewReportGenerator()
	require.NotNil(t, g)
}

// ---------------------------------------------------------------------------
// BuildReport
// ---------------------------------------------------------------------------

func TestBuildReport_ZeroMatches(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}

	report := g.BuildReport(summary, nil, "default", "high")

	assert.Equal(t, "default", report.Profile)
	assert.Equal(t, "high", report.ConfidenceThreshold)
	assert.Equal(t, 0, report.Summary.TotalRedactions)
	assert.Equal(t, 0, report.Summary.FilesAffected)
	assert.Empty(t, report.Redactions)
	// GeneratedAt must be recent (within 5 seconds of now).
	assert.WithinDuration(t, time.Now().UTC(), report.GeneratedAt, 5*time.Second)
}

func TestBuildReport_SingleMatch(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "config/db.go", 42, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)

	report := g.BuildReport(summary, matches, "finvault", "medium")

	require.Len(t, report.Redactions, 1)
	entry := report.Redactions[0]
	assert.Equal(t, "config/db.go", entry.File)
	assert.Equal(t, 42, entry.Line)
	assert.Equal(t, "aws_access_key", entry.SecretType)
	assert.Equal(t, ConfidenceHigh, entry.Confidence)
	assert.Equal(t, "[REDACTED:aws_access_key]", entry.Replacement)
	assert.Equal(t, "test-rule", entry.RuleID)
	assert.Equal(t, 1, report.Summary.FilesAffected)
}

func TestBuildReport_MultipleMatchesSortedByFileThenLine(t *testing.T) {
	g := newGenerator()

	// Deliberately supply matches in reverse order to verify sorting.
	matches := []RedactionMatch{
		makeMatch(t, "z_file.go", 100, "github_token", ConfidenceHigh),
		makeMatch(t, "a_file.go", 50, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "a_file.go", 10, "jwt_token", ConfidenceMedium),
		makeMatch(t, "m_file.go", 5, "connection_string", ConfidenceMedium),
	}
	summary := makeSummary(t, matches, 3)

	report := g.BuildReport(summary, matches, "default", "low")

	require.Len(t, report.Redactions, 4)
	// After sort: a_file.go:10, a_file.go:50, m_file.go:5, z_file.go:100
	assert.Equal(t, "a_file.go", report.Redactions[0].File)
	assert.Equal(t, 10, report.Redactions[0].Line)
	assert.Equal(t, "a_file.go", report.Redactions[1].File)
	assert.Equal(t, 50, report.Redactions[1].Line)
	assert.Equal(t, "m_file.go", report.Redactions[2].File)
	assert.Equal(t, 5, report.Redactions[2].Line)
	assert.Equal(t, "z_file.go", report.Redactions[3].File)
	assert.Equal(t, 100, report.Redactions[3].Line)
}

func TestBuildReport_ByConfidenceUsesStringKeys(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "f.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "f.go", 2, "github_token", ConfidenceMedium),
		makeMatch(t, "f.go", 3, "generic_api_key", ConfidenceLow),
	}
	summary := makeSummary(t, matches, 1)

	report := g.BuildReport(summary, matches, "default", "low")

	// ByConfidence must have string keys, not Confidence type keys.
	assert.Equal(t, 1, report.Summary.ByConfidence["high"])
	assert.Equal(t, 1, report.Summary.ByConfidence["medium"])
	assert.Equal(t, 1, report.Summary.ByConfidence["low"])
}

func TestBuildReport_FilesAffectedMatchesSummaryFileCount(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "b.go", 2, "github_token", ConfidenceHigh),
		makeMatch(t, "c.go", 3, "jwt_token", ConfidenceMedium),
	}
	summary := makeSummary(t, matches, 3)

	report := g.BuildReport(summary, matches, "default", "high")

	assert.Equal(t, 3, report.Summary.FilesAffected)
}

func TestBuildReport_NoSecretValuesInEntries(t *testing.T) {
	g := newGenerator()

	// Construct a match that carries a realistic-looking secret value in
	// the Replacement field only — the Replacement must be the sanitized
	// marker, never the raw secret.
	match := RedactionMatch{
		RuleID:      "aws-access-key-id",
		SecretType:  "aws_access_key_id",
		Confidence:  ConfidenceHigh,
		FilePath:    "config.env",
		LineNumber:  1,
		StartCol:    18,
		EndCol:      38,
		Replacement: "[REDACTED:aws_access_key_id]",
	}
	summary := makeSummary(t, []RedactionMatch{match}, 1)

	report := g.BuildReport(summary, []RedactionMatch{match}, "default", "high")

	require.Len(t, report.Redactions, 1)
	entry := report.Redactions[0]
	// The entry's Replacement field must be the marker, not the real key.
	assert.Equal(t, "[REDACTED:aws_access_key_id]", entry.Replacement)
	// No raw AKIA prefix should appear anywhere in the entry.
	assert.NotContains(t, entry.Replacement, "AKIA")
}

func TestBuildReport_ByTypeDeepCopy(t *testing.T) {
	// Mutating the returned Report's ByType must not affect the original summary map.
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "f.go", 1, "aws_access_key", ConfidenceHigh),
	}
	summary := makeSummary(t, matches, 1)
	originalByType := summary.ByType["aws_access_key"]

	report := g.BuildReport(summary, matches, "default", "high")

	// Mutate the report's copy.
	report.Summary.ByType["aws_access_key"] = 9999

	// Original summary must be unchanged.
	assert.Equal(t, originalByType, summary.ByType["aws_access_key"])
}

func TestBuildReport_GeneratedAtIsUTC(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "p", "high")
	// Location must be UTC.
	assert.Equal(t, "UTC", report.GeneratedAt.Location().String())
}

// ---------------------------------------------------------------------------
// GenerateJSON
// ---------------------------------------------------------------------------

func TestGenerateJSON_ValidJSONStructure(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "src/app.go", 7, "github_token", ConfidenceHigh)
	summary := makeSummary(t, []RedactionMatch{match}, 1)
	report := g.BuildReport(summary, []RedactionMatch{match}, "finvault", "high")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Must round-trip through json.Unmarshal without error.
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))

	// Required top-level keys.
	for _, key := range []string{"generated_at", "profile", "confidence_threshold", "summary", "redactions"} {
		assert.Contains(t, decoded, key, "JSON must contain top-level key %q", key)
	}
}

func TestGenerateJSON_ZeroRedactionsStructure(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))

	summaryField, ok := decoded["summary"].(map[string]interface{})
	require.True(t, ok, "summary must be a JSON object")
	assert.Equal(t, float64(0), summaryField["total_redactions"])

	redactionsField, ok := decoded["redactions"]
	require.True(t, ok)
	// redactions must be a JSON array (may be empty or null — both acceptable).
	// When it is null it will be nil; when it is [] it will be a slice.
	if redactionsField != nil {
		_, isSlice := redactionsField.([]interface{})
		assert.True(t, isSlice, "redactions must be a JSON array when non-null")
	}
}

func TestGenerateJSON_ProfileAndThresholdPresent(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "myprofile", "medium")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "myprofile", decoded["profile"])
	assert.Equal(t, "medium", decoded["confidence_threshold"])
}

func TestGenerateJSON_NoActualSecretValues(t *testing.T) {
	g := newGenerator()
	// Use a realistic-looking (but fake) key value; it must never appear in the
	// JSON output because BuildReport only stores the Replacement marker.
	match := RedactionMatch{
		RuleID:      "aws-access-key-id",
		SecretType:  "aws_access_key_id",
		Confidence:  ConfidenceHigh,
		FilePath:    "config.env",
		LineNumber:  1,
		StartCol:    18,
		EndCol:      38,
		Replacement: "[REDACTED:aws_access_key_id]",
	}
	summary := makeSummary(t, []RedactionMatch{match}, 1)
	report := g.BuildReport(summary, []RedactionMatch{match}, "default", "high")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)

	jsonStr := string(data)
	// The JSON must never contain raw AWS key prefixes or other sentinel patterns
	// that would indicate an actual secret leaked.
	assert.NotContains(t, jsonStr, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, jsonStr, "ghp_")
	assert.NotContains(t, jsonStr, "sk_live_")
}

func TestGenerateJSON_IndentedOutput(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)

	// Indented JSON contains newlines and leading spaces.
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "\n", "GenerateJSON must produce indented (multi-line) JSON")
	assert.Contains(t, jsonStr, "  ", "GenerateJSON must use 2-space indentation")
}

// ---------------------------------------------------------------------------
// GenerateText
// ---------------------------------------------------------------------------

func TestGenerateText_ZeroRedactions(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	text := g.GenerateText(report)
	assert.Equal(t, "No secrets detected.", text)
}

func TestGenerateText_Header(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "src/app.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	text := g.GenerateText(report)

	assert.Contains(t, text, "Harvx Redaction Report\n======================")
}

func TestGenerateText_GeneratedProfileThresholdLines(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "src/app.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "myprofile", "medium")

	text := g.GenerateText(report)

	assert.Contains(t, text, "Generated:")
	assert.Contains(t, text, "Profile: myprofile")
	assert.Contains(t, text, "Threshold: medium")
}

func TestGenerateText_SingularRedaction(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "file.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	text := g.GenerateText(report)

	// "Summary: 1 redaction in N files" — singular, not "redactions".
	assert.Contains(t, text, "Summary: 1 redaction in")
	assert.NotContains(t, text, "Summary: 1 redactions in")
}

func TestGenerateText_PluralRedactions(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "b.go", 2, "github_token", ConfidenceHigh),
		makeMatch(t, "c.go", 3, "jwt_token", ConfidenceMedium),
	}
	summary := makeSummary(t, matches, 3)
	report := g.BuildReport(summary, matches, "default", "high")

	text := g.GenerateText(report)

	assert.Contains(t, text, "Summary: 3 redactions in")
}

func TestGenerateText_DetailsSectionPresent(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "src/app.go", 99, "github_token", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	text := g.GenerateText(report)

	assert.Contains(t, text, "Details:")
}

func TestGenerateText_DetailLineContainsFileLineAndType(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "internal/config/settings.go", 42, "connection_string", ConfidenceMedium)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "medium")

	text := g.GenerateText(report)

	assert.Contains(t, text, "internal/config/settings.go:42")
	assert.Contains(t, text, "connection_string")
	assert.Contains(t, text, "medium")
}

func TestGenerateText_AllThreeConfidenceLevels(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "b.go", 2, "github_token", ConfidenceMedium),
		makeMatch(t, "c.go", 3, "generic_api_key", ConfidenceLow),
	}
	summary := makeSummary(t, matches, 3)
	report := g.BuildReport(summary, matches, "default", "low")

	text := g.GenerateText(report)

	assert.Contains(t, text, "high")
	assert.Contains(t, text, "medium")
	assert.Contains(t, text, "low")
}

func TestGenerateText_NoDetailsWhenZeroRedactions(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	text := g.GenerateText(report)

	assert.NotContains(t, text, "Details:")
}

// ---------------------------------------------------------------------------
// WriteReport
// ---------------------------------------------------------------------------

func TestWriteReport_JSONExtension(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	match := makeMatch(t, "src/app.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded), "WriteReport to .json must produce valid JSON")
	assert.Contains(t, decoded, "generated_at")
	assert.Contains(t, decoded, "summary")
}

func TestWriteReport_TxtExtension(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")

	match := makeMatch(t, "src/app.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(data), "Harvx Redaction Report",
		"WriteReport to .txt must produce text format")
}

func TestWriteReport_TextExtension(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.text")

	match := makeMatch(t, "src/app.go", 1, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(data), "Harvx Redaction Report",
		"WriteReport to .text must produce text format")
}

func TestWriteReport_NoExtensionIsJSON(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report") // no extension

	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded),
		"WriteReport with no extension must produce JSON")
}

func TestWriteReport_CreatesParentDirectories(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	// Two levels of non-existent sub-directories.
	path := filepath.Join(dir, "deep", "nested", "report.json")

	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	// File must exist.
	_, statErr := os.Stat(path)
	require.NoError(t, statErr, "report file must exist after WriteReport")
}

func TestWriteReport_FilePermissions(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	// File must be written with 0644 permissions.
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestWriteReport_ReturnsNoErrorOnSuccess(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	report := g.BuildReport(summary, nil, "default", "high")

	err := g.WriteReport(report, path)
	assert.NoError(t, err)
}

func TestWriteReport_CaseInsensitiveExtension(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "REPORT.TXT")

	match := makeMatch(t, "src/app.go", 1, "jwt_token", ConfidenceMedium)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "medium")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(data), "Harvx Redaction Report",
		"WriteReport with uppercase .TXT extension must produce text format")
}

// ---------------------------------------------------------------------------
// FormatInlineSummary
// ---------------------------------------------------------------------------

func TestFormatInlineSummary_ZeroRedactions(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   0,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{},
		FileCount:    0,
	}
	assert.Equal(t, "No secrets detected", g.FormatInlineSummary(summary))
}

func TestFormatInlineSummary_OneRedactionOneType(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   1,
		ByType:       map[string]int{"aws_access_key": 1},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 1},
		FileCount:    1,
	}
	got := g.FormatInlineSummary(summary)
	// singular "redaction", singular label "AWS access key"
	assert.Equal(t, "1 redaction (1 AWS access key)", got)
}

func TestFormatInlineSummary_ThreeRedactionsOneType(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   3,
		ByType:       map[string]int{"aws_access_key": 3},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 3},
		FileCount:    1,
	}
	got := g.FormatInlineSummary(summary)
	// plural "redactions", plural label "AWS access keys"
	assert.Equal(t, "3 redactions (3 AWS access keys)", got)
}

func TestFormatInlineSummary_MultipleTypesSortedByCountDescThenNameAsc(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount: 7,
		ByType: map[string]int{
			"generic_api_key":   3,
			"connection_string": 2,
			"private_key_block": 1,
			"jwt_token":         1,
		},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 7},
		FileCount:    4,
	}
	got := g.FormatInlineSummary(summary)
	// Expected order: 3 API keys, 2 connection strings, 1 JWT token, 1 private key block
	// (ties broken by secret type name ascending: jwt_token < private_key_block)
	assert.Equal(t, "7 redactions (3 API keys, 2 connection strings, 1 JWT token, 1 private key block)", got)
}

func TestFormatInlineSummary_ByTypeEmptyButTotalNonZero(t *testing.T) {
	g := newGenerator()
	// ByType may be empty when the summary was constructed externally without
	// per-type accounting.
	summary := RedactionSummary{
		TotalCount:   5,
		ByType:       map[string]int{},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 5},
		FileCount:    2,
	}
	got := g.FormatInlineSummary(summary)
	// No parenthetical breakdown when ByType is empty.
	assert.Equal(t, "5 redactions", got)
}

func TestFormatInlineSummary_UnknownTypeUnderscoreToSpace(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   1,
		ByType:       map[string]int{"my_custom_secret_type": 1},
		ByConfidence: map[Confidence]int{ConfidenceLow: 1},
		FileCount:    1,
	}
	got := g.FormatInlineSummary(summary)
	// Unknown type: underscores replaced, first char uppercased.
	// "my_custom_secret_type" → "My custom secret type"
	assert.Equal(t, "1 redaction (1 My custom secret type)", got)
}

func TestFormatInlineSummary_SingleRedactionIsNotPlural(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount:   1,
		ByType:       map[string]int{"github_token": 1},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 1},
		FileCount:    1,
	}
	got := g.FormatInlineSummary(summary)
	assert.Contains(t, got, "1 redaction (", "must use singular 'redaction' for count == 1")
	assert.NotContains(t, got, "redactions")
}

// ---------------------------------------------------------------------------
// humanLabel (internal function – white-box test)
// ---------------------------------------------------------------------------

func TestHumanLabel_KnownTypes(t *testing.T) {
	tests := []struct {
		secretType string
		want       string
	}{
		{"aws_access_key", "AWS access key"},
		{"aws_secret_key", "AWS secret key"},
		{"github_token", "GitHub token"},
		{"gitlab_token", "GitLab token"},
		{"private_key_block", "private key block"},
		{"connection_string", "connection string"},
		{"jwt_token", "JWT token"},
		{"generic_api_key", "API key"},
		{"high_entropy_secret", "high entropy secret"},
		{"slack_token", "Slack token"},
		{"stripe_secret_key", "Stripe secret key"},
		{"google_api_key", "Google API key"},
		{"twilio_api_key", "Twilio API key"},
		{"sendgrid_api_key", "SendGrid API key"},
		{"mailchimp_api_key", "Mailchimp API key"},
		{"heroku_api_key", "Heroku API key"},
		{"basic_auth", "basic auth credential"},
		{"bearer_token", "bearer token"},
	}

	for _, tt := range tests {
		t.Run(tt.secretType, func(t *testing.T) {
			got := humanLabel(tt.secretType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHumanLabel_UnknownTypeFallback(t *testing.T) {
	tests := []struct {
		name       string
		secretType string
		want       string
	}{
		{
			name:       "underscores replaced by spaces, first char uppercased",
			secretType: "my_custom_key",
			want:       "My custom key",
		},
		{
			name:       "no underscores, first char uppercased",
			secretType: "customtype",
			want:       "Customtype",
		},
		{
			name:       "empty string returns empty string",
			secretType: "",
			want:       "",
		},
		{
			name:       "already uppercased remains valid",
			secretType: "MY_TYPE",
			want:       "MY TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanLabel(tt.secretType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// pluralize (internal function – white-box test)
// ---------------------------------------------------------------------------

func TestPluralize(t *testing.T) {
	tests := []struct {
		name  string
		label string
		count int
		want  string
	}{
		{name: "count 1 no suffix", label: "API key", count: 1, want: "API key"},
		{name: "count 2 adds s", label: "API key", count: 2, want: "API keys"},
		{name: "count 0 adds s", label: "AWS access key", count: 0, want: "AWS access keys"},
		{name: "count 100 adds s", label: "token", count: 100, want: "tokens"},
		{name: "empty label count 1", label: "", count: 1, want: ""},
		{name: "empty label count 2", label: "", count: 2, want: "s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluralize(tt.label, tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestBuildReport_LargeNumberOfMatches(t *testing.T) {
	g := newGenerator()
	const n = 1000

	matches := make([]RedactionMatch, n)
	for i := range matches {
		matches[i] = makeMatch(t, "big_file.go", i+1, "aws_access_key", ConfidenceHigh)
	}
	summary := makeSummary(t, matches, 1)

	report := g.BuildReport(summary, matches, "default", "high")

	assert.Len(t, report.Redactions, n)
	// Verify sort stability: lines must be in ascending order.
	for i := 1; i < len(report.Redactions); i++ {
		assert.LessOrEqual(t, report.Redactions[i-1].Line, report.Redactions[i].Line)
	}
}

func TestGenerateJSON_LargeReport(t *testing.T) {
	g := newGenerator()
	const n = 1000

	matches := make([]RedactionMatch, n)
	for i := range matches {
		matches[i] = makeMatch(t, "big_file.go", i+1, "github_token", ConfidenceHigh)
	}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "high")

	data, err := g.GenerateJSON(report)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))
}

func TestBuildReport_FilePathsWithSpacesAndUnicode(t *testing.T) {
	g := newGenerator()
	unicodePath := "src/フォルダ/my file with spaces.go"
	match := makeMatch(t, unicodePath, 5, "jwt_token", ConfidenceMedium)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)

	report := g.BuildReport(summary, matches, "default", "medium")

	require.Len(t, report.Redactions, 1)
	assert.Equal(t, unicodePath, report.Redactions[0].File,
		"file paths with spaces and Unicode must be preserved verbatim")
}

func TestGenerateText_FilePathsWithSpacesAndUnicode(t *testing.T) {
	g := newGenerator()
	unicodePath := "src/フォルダ/my file with spaces.go"
	match := makeMatch(t, unicodePath, 5, "jwt_token", ConfidenceMedium)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "medium")

	text := g.GenerateText(report)
	assert.Contains(t, text, unicodePath)
}

func TestBuildReport_AllThreeConfidenceLevelsInByConfidence(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "a.go", 2, "github_token", ConfidenceMedium),
		makeMatch(t, "a.go", 3, "generic_api_key", ConfidenceLow),
	}
	summary := makeSummary(t, matches, 1)

	report := g.BuildReport(summary, matches, "default", "low")

	assert.Equal(t, 1, report.Summary.ByConfidence["high"])
	assert.Equal(t, 1, report.Summary.ByConfidence["medium"])
	assert.Equal(t, 1, report.Summary.ByConfidence["low"])
	// Ensure no extra confidence levels appear.
	assert.Len(t, report.Summary.ByConfidence, 3)
}

func TestWriteReport_JSONContainsEntries(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "aws_access_key", ConfidenceHigh),
		makeMatch(t, "b.go", 2, "github_token", ConfidenceHigh),
	}
	summary := makeSummary(t, matches, 2)
	report := g.BuildReport(summary, matches, "default", "high")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &decoded))

	redactions, ok := decoded["redactions"].([]interface{})
	require.True(t, ok)
	assert.Len(t, redactions, 2)
}

func TestWriteReport_TextContainsDetailLines(t *testing.T) {
	g := newGenerator()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")

	matches := []RedactionMatch{
		makeMatch(t, "internal/db/conn.go", 77, "connection_string", ConfidenceMedium),
	}
	summary := makeSummary(t, matches, 1)
	report := g.BuildReport(summary, matches, "default", "medium")

	err := g.WriteReport(report, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	text := string(data)
	assert.Contains(t, text, "internal/db/conn.go:77")
	assert.Contains(t, text, "connection_string")
}

// ---------------------------------------------------------------------------
// dominantConfidenceForType (internal function – white-box test)
// ---------------------------------------------------------------------------

func TestDominantConfidenceForType_ReturnsHighestConfidence(t *testing.T) {
	entries := []ReportEntry{
		{SecretType: "aws_access_key", Confidence: ConfidenceLow},
		{SecretType: "aws_access_key", Confidence: ConfidenceHigh},
		{SecretType: "aws_access_key", Confidence: ConfidenceMedium},
	}
	got := dominantConfidenceForType(entries, "aws_access_key")
	assert.Equal(t, "high", got)
}

func TestDominantConfidenceForType_NoMatchingType(t *testing.T) {
	entries := []ReportEntry{
		{SecretType: "github_token", Confidence: ConfidenceHigh},
	}
	got := dominantConfidenceForType(entries, "aws_access_key")
	assert.Equal(t, "unknown", got)
}

func TestDominantConfidenceForType_EmptyEntries(t *testing.T) {
	got := dominantConfidenceForType(nil, "aws_access_key")
	assert.Equal(t, "unknown", got)
}

func TestDominantConfidenceForType_SingleEntry(t *testing.T) {
	entries := []ReportEntry{
		{SecretType: "jwt_token", Confidence: ConfidenceMedium},
	}
	got := dominantConfidenceForType(entries, "jwt_token")
	assert.Equal(t, "medium", got)
}

// ---------------------------------------------------------------------------
// FormatInlineSummary — determinism under repeated calls
// ---------------------------------------------------------------------------

func TestFormatInlineSummary_DeterministicOutput(t *testing.T) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount: 10,
		ByType: map[string]int{
			"generic_api_key":   4,
			"github_token":      3,
			"connection_string": 2,
			"jwt_token":         1,
		},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 10},
		FileCount:    5,
	}

	// Call multiple times; output must be identical.
	first := g.FormatInlineSummary(summary)
	for i := 0; i < 10; i++ {
		got := g.FormatInlineSummary(summary)
		assert.Equal(t, first, got, "FormatInlineSummary must be deterministic (call %d)", i+1)
	}
}

// ---------------------------------------------------------------------------
// JSON round-trip: GenerateJSON then unmarshal back into Report
// ---------------------------------------------------------------------------

func TestGenerateJSON_RoundTrip(t *testing.T) {
	g := newGenerator()
	match := makeMatch(t, "src/main.go", 12, "aws_access_key", ConfidenceHigh)
	matches := []RedactionMatch{match}
	summary := makeSummary(t, matches, 1)
	original := g.BuildReport(summary, matches, "roundtrip-profile", "medium")

	data, err := g.GenerateJSON(original)
	require.NoError(t, err)

	var decoded Report
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.Profile, decoded.Profile)
	assert.Equal(t, original.ConfidenceThreshold, decoded.ConfidenceThreshold)
	assert.Equal(t, original.Summary.TotalRedactions, decoded.Summary.TotalRedactions)
	assert.Equal(t, original.Summary.FilesAffected, decoded.Summary.FilesAffected)
	require.Len(t, decoded.Redactions, 1)
	assert.Equal(t, original.Redactions[0].File, decoded.Redactions[0].File)
	assert.Equal(t, original.Redactions[0].Line, decoded.Redactions[0].Line)
	assert.Equal(t, original.Redactions[0].SecretType, decoded.Redactions[0].SecretType)
	assert.Equal(t, original.Redactions[0].Replacement, decoded.Redactions[0].Replacement)
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkBuildReport_1000Matches(b *testing.B) {
	g := newGenerator()
	const n = 1000
	matches := make([]RedactionMatch, n)
	for i := range matches {
		matches[i] = RedactionMatch{
			RuleID:      "aws-access-key-id",
			SecretType:  "aws_access_key_id",
			Confidence:  ConfidenceHigh,
			FilePath:    "src/config.go",
			LineNumber:  i + 1,
			StartCol:    18,
			EndCol:      38,
			Replacement: "[REDACTED:aws_access_key_id]",
		}
	}
	summary := RedactionSummary{
		TotalCount:   n,
		ByType:       map[string]int{"aws_access_key_id": n},
		ByConfidence: map[Confidence]int{ConfidenceHigh: n},
		FileCount:    1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.BuildReport(summary, matches, "default", "high")
	}
}

func BenchmarkGenerateJSON_1000Entries(b *testing.B) {
	g := newGenerator()
	const n = 1000
	matches := make([]RedactionMatch, n)
	for i := range matches {
		matches[i] = RedactionMatch{
			RuleID:      "aws-access-key-id",
			SecretType:  "aws_access_key_id",
			Confidence:  ConfidenceHigh,
			FilePath:    "src/config.go",
			LineNumber:  i + 1,
			StartCol:    18,
			EndCol:      38,
			Replacement: "[REDACTED:aws_access_key_id]",
		}
	}
	summary := RedactionSummary{
		TotalCount:   n,
		ByType:       map[string]int{"aws_access_key_id": n},
		ByConfidence: map[Confidence]int{ConfidenceHigh: n},
		FileCount:    1,
	}
	report := g.BuildReport(summary, matches, "default", "high")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateJSON(report)
	}
}

func BenchmarkFormatInlineSummary(b *testing.B) {
	g := newGenerator()
	summary := RedactionSummary{
		TotalCount: 100,
		ByType: map[string]int{
			"generic_api_key":   40,
			"github_token":      30,
			"connection_string": 20,
			"jwt_token":         10,
		},
		ByConfidence: map[Confidence]int{ConfidenceHigh: 100},
		FileCount:    10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.FormatInlineSummary(summary)
	}
}

// ---------------------------------------------------------------------------
// GenerateText — type breakdown lines present when redactions exist
// ---------------------------------------------------------------------------

func TestGenerateText_TypeBreakdownLines(t *testing.T) {
	g := newGenerator()
	matches := []RedactionMatch{
		makeMatch(t, "a.go", 1, "generic_api_key", ConfidenceHigh),
		makeMatch(t, "a.go", 2, "generic_api_key", ConfidenceHigh),
		makeMatch(t, "b.go", 3, "github_token", ConfidenceMedium),
	}
	summary := makeSummary(t, matches, 2)
	report := g.BuildReport(summary, matches, "default", "high")

	text := g.GenerateText(report)

	// The breakdown line for generic_api_key (2 → plural "API keys") must appear.
	assert.Contains(t, text, "2 API keys")
	// The breakdown line for github_token (1 → singular "GitHub token").
	assert.Contains(t, text, "1 GitHub token")
	// Confidence label for each type.
	assert.True(t,
		strings.Contains(text, "high confidence") || strings.Contains(text, "medium confidence"),
	)
}
