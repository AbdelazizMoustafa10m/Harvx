// Package security provides secret detection and redaction capabilities for
// LLM context documents. It scans file content for credentials, tokens, and
// other sensitive values before they are included in generated output.
//
// The package is structured as follows:
//   - types.go: shared data types (Confidence, match/summary/config structs)
//   - rule.go: RedactionRule struct and FormatReplacement helper
//   - registry.go: PatternRegistry for storing and querying rules
//   - redactor.go: Redactor interface satisfied by concrete implementations
//
// Patterns are registered at program initialisation (T-035) then treated as
// read-only; concurrent read access to the registry is therefore safe without
// additional locking.
package security

// Confidence represents the certainty level of a secret detection match.
// It is encoded as a string so it round-trips cleanly through JSON and TOML.
type Confidence string

const (
	// ConfidenceHigh indicates a strong structural match (e.g. a regex that
	// captures the exact format of an AWS access key). False-positive rate is
	// very low. High-confidence matches are always redacted.
	ConfidenceHigh Confidence = "high"

	// ConfidenceMedium indicates a probable match based on keyword proximity
	// combined with an entropy heuristic. Some false positives are expected.
	ConfidenceMedium Confidence = "medium"

	// ConfidenceLow indicates a weak signal such as a high-entropy string
	// without supporting keywords. Use with caution; false-positive rate is
	// elevated.
	ConfidenceLow Confidence = "low"
)

// RedactionMatch records a single detected secret within a file. All fields
// are exported with JSON tags for structured logging and reporting.
type RedactionMatch struct {
	// RuleID is the unique identifier of the RedactionRule that triggered
	// this match (e.g. "aws-access-key", "generic-api-key").
	RuleID string `json:"rule_id"`

	// SecretType is a human-readable category label (e.g. "AWS Access Key",
	// "GitHub Token", "Generic API Key").
	SecretType string `json:"secret_type"`

	// Confidence is the detection certainty level for this match.
	Confidence Confidence `json:"confidence"`

	// FilePath is the path of the file in which the secret was found.
	// It is relative to the repository root when available.
	FilePath string `json:"file_path"`

	// LineNumber is the 1-based line number where the secret begins.
	LineNumber int `json:"line_number"`

	// StartCol is the 0-based byte offset of the match start within LineNumber.
	StartCol int `json:"start_col"`

	// EndCol is the 0-based byte offset one past the last byte of the match
	// within LineNumber.
	EndCol int `json:"end_col"`

	// Replacement is the string that was written in place of the redacted
	// value (e.g. "[REDACTED:aws_access_key]").
	Replacement string `json:"replacement"`
}

// RedactionSummary aggregates match statistics for a single redaction pass.
// It is included in pipeline output metadata and CLI reports.
type RedactionSummary struct {
	// TotalCount is the total number of individual matches found across all
	// files in the pass.
	TotalCount int `json:"total_count"`

	// ByType maps each SecretType label to its match count.
	ByType map[string]int `json:"by_type"`

	// ByConfidence maps each Confidence level to its match count.
	ByConfidence map[Confidence]int `json:"by_confidence"`

	// FileCount is the number of distinct files that contained at least one
	// match.
	FileCount int `json:"file_count"`
}

// RedactionConfig controls the runtime behaviour of the secret redaction
// subsystem. It is distinct from config.RedactionConfig (which is used for
// TOML parsing) because it carries a typed Confidence field and supports
// custom pattern definitions that are not expressed in the config file.
type RedactionConfig struct {
	// Enabled turns secret redaction on or off. When false the Redactor
	// implementation must return the input content unchanged.
	Enabled bool `json:"enabled"`

	// ExcludePaths is a list of doublestar glob patterns. Files whose paths
	// match any of these patterns are skipped by the redactor. Useful for
	// excluding test fixtures and documentation that legitimately contain
	// example secrets.
	ExcludePaths []string `json:"exclude_paths"`

	// ConfidenceThreshold is the minimum confidence level for a match to
	// trigger redaction. Matches below this level are reported but not
	// replaced. Valid values are ConfidenceLow, ConfidenceMedium, and
	// ConfidenceHigh.
	ConfidenceThreshold Confidence `json:"confidence_threshold"`

	// CustomPatterns holds additional redaction rules supplied at runtime
	// (e.g. from a harvx.toml custom_patterns block). They are appended to
	// the built-in rules in the PatternRegistry.
	CustomPatterns []CustomPatternConfig `json:"custom_patterns"`
}

// CustomPatternConfig describes a user-defined redaction rule that can be
// added to the PatternRegistry at runtime. It is serialisable so it can be
// round-tripped through JSON configuration or CLI flags.
type CustomPatternConfig struct {
	// ID is the unique identifier for the custom rule. Must not collide with
	// any built-in rule ID.
	ID string `json:"id"`

	// Description is a short human-readable explanation of what the pattern
	// detects (shown in reports and --verbose output).
	Description string `json:"description"`

	// Pattern is a Go RE2-compatible regular expression string. The regexp
	// engine is Go's stdlib regexp package; lookaheads and backreferences are
	// not supported.
	Pattern string `json:"pattern"`

	// SecretType is the category label included in the redaction replacement
	// string (e.g. "my_company_api_key").
	SecretType string `json:"secret_type"`

	// Confidence is the certainty level assigned to all matches from this rule.
	Confidence Confidence `json:"confidence"`
}
